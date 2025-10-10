// shared/library/file_jumble.go
package library

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

type fileMeta struct {
	Mode  os.FileMode
	UID   int
	GID   int
	Atime time.Time
	Mtime time.Time
	XAttr map[string][]byte // best-effort; may be empty if not permitted
}

// CyclicJumble takes a list of absolute file paths (regular files) and performs a cycle:
// paths[i] content -> paths[(i+1)%n] for all i.
//
// It preserves each destination file's original metadata as much as possible:
// - permissions (mode), ownership (uid/gid), atime/mtime, and extended attributes (best-effort).
// - Operations are atomic per destination via rename(2) of a same-dir temp file.
// Notes/limits:
// - ctime cannot be preserved on Linux.
// - Non-regular files (dirs/symlinks/devices) are rejected.
// - Paths must be unique.
func CyclicJumble(paths []string) error {
	if len(paths) < 2 {
		return errors.New("need at least two paths for a cyclic jumble")
	}
	// Basic validation and de-dup.
	seen := make(map[string]struct{}, len(paths))
	for i, p := range paths {
		if !filepath.IsAbs(p) {
			return fmt.Errorf("path %d not absolute: %s", i, p)
		}
		if _, ok := seen[p]; ok {
			return fmt.Errorf("duplicate path detected: %s", p)
		}
		seen[p] = struct{}{}
		fi, err := os.Lstat(p)
		if err != nil {
			return fmt.Errorf("stat %s: %w", p, err)
		}
		if !fi.Mode().IsRegular() {
			return fmt.Errorf("not a regular file: %s", p)
		}
	}

	// Snapshot destination metadata and stage source contents into temp files.
	n := len(paths)
	destMeta := make(map[string]fileMeta, n)
	srcTemp := make(map[string]string, n) // original path -> temp file holding its content

	for _, dest := range paths {
		m, err := captureMeta(dest)
		if err != nil {
			return fmt.Errorf("capture meta %s: %w", dest, err)
		}
		destMeta[dest] = m
	}

	for _, src := range paths {
		tmp, err := stageCopy(src)
		if err != nil {
			// Cleanup previously staged temps on error.
			for _, t := range srcTemp {
				_ = os.Remove(t)
			}
			return fmt.Errorf("stage copy %s: %w", src, err)
		}
		srcTemp[src] = tmp
	}

	// Perform the cycle writes: content of paths[i] -> paths[(i+1)%n]
	for i, src := range paths {
		dest := paths[(i+1)%n]
		if err := writeOverWithMeta(srcTemp[src], dest, destMeta[dest]); err != nil {
			// Abort on first failure.
			return fmt.Errorf("apply to %s: %w", dest, err)
		}
	}

	// Cleanup temps
	for _, t := range srcTemp {
		_ = os.Remove(t)
	}
	return nil
}

// captureMeta collects mode, uid/gid, atime/mtime, and xattrs (best-effort).
func captureMeta(path string) (fileMeta, error) {
	var m fileMeta
	fi, err := os.Lstat(path)
	if err != nil {
		return m, err
	}
	m.Mode = fi.Mode().Perm()

	st, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		return m, errors.New("unexpected stat type")
	}
	m.UID = int(st.Uid)
	m.GID = int(st.Gid)

	// atime/mtime from Stat_t
	m.Atime = time.Unix(int64(st.Atim.Sec), int64(st.Atim.Nsec))
	m.Mtime = time.Unix(int64(st.Mtim.Sec), int64(st.Mtim.Nsec))

	// xattrs (best-effort; ignore permission/unsupported errors)
	m.XAttr = make(map[string][]byte)
	buf := make([]byte, 4096)
	for {
		n, err := unix.Listxattr(path, buf)
		if err != nil {
			break
		}
		if n == len(buf) {
			buf = make([]byte, len(buf)*2)
			continue
		}
		names := splitZ(buf[:n])
		for _, name := range names {
			val := make([]byte, 4096)
			for {
				vn, gerr := unix.Getxattr(path, name, val)
				if gerr == unix.ERANGE || vn == len(val) {
					val = make([]byte, len(val)*2)
					continue
				}
				if gerr == nil {
					m.XAttr[name] = append([]byte(nil), val[:vn]...)
				}
				break
			}
		}
		break
	}
	return m, nil
}

// stageCopy writes the file's content into a same-dir temp file and fsyncs it.
func stageCopy(src string) (string, error) {
	dir := filepath.Dir(src)
	tmp, err := os.CreateTemp(dir, ".jumble-src-*")
	if err != nil {
		return "", err
	}
	defer tmp.Close()

	sf, err := os.Open(src)
	if err != nil {
		_ = os.Remove(tmp.Name())
		return "", err
	}
	defer sf.Close()

	if _, err = io.Copy(tmp, sf); err != nil {
		_ = os.Remove(tmp.Name())
		return "", err
	}
	if err = tmp.Sync(); err != nil {
		_ = os.Remove(tmp.Name())
		return "", err
	}
	return tmp.Name(), nil
}

// writeOverWithMeta replaces dest with the contents of stagedSrc, restoring dest's original metadata.
func writeOverWithMeta(stagedSrc, dest string, meta fileMeta) error {
	dir := filepath.Dir(dest)
	tmp, err := os.CreateTemp(dir, ".jumble-dst-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer func() { _ = tmp.Close() }()

	// Copy bytes from stagedSrc into tmp
	sf, err := os.Open(stagedSrc)
	if err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	if _, err = io.Copy(tmp, sf); err != nil {
		_ = os.Remove(tmpPath)
		_ = sf.Close()
		return err
	}
	_ = sf.Close()

	// Flush content
	if err = tmp.Sync(); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}

	// Restore metadata on tmp BEFORE rename so ownership/mode persist across atomic replace.
	if err = os.Chmod(tmpPath, meta.Mode); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	if err = os.Chown(tmpPath, meta.UID, meta.GID); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}

	// Restore xattrs (best-effort)
	for k, v := range meta.XAttr {
		if err := unix.Setxattr(tmpPath, k, v, 0); err != nil && !isIgnorableXErr(err) {
			// Ignore unsupported/permission errors silently.
		}
	}

	// Atomic replace
	if err = os.Rename(tmpPath, dest); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}

	// Set atime/mtime after rename (ctime cannot be set)
	if err = os.Chtimes(dest, meta.Atime, meta.Mtime); err != nil {
		return fmt.Errorf("renamed but failed to restore times: %w", err)
	}
	return nil
}

func splitZ(b []byte) []string {
	var out []string
	for len(b) > 0 {
		i := bytes.IndexByte(b, 0)
		if i < 0 {
			break
		}
		if i > 0 {
			out = append(out, string(b[:i]))
		}
		b = b[i+1:]
	}
	return out
}

func isIgnorableXErr(err error) bool {
	// On Linux ENOTSUP == EOPNOTSUPP; keep only ENOTSUP to avoid duplicate cases.
	switch err {
	case unix.ENOTSUP, unix.EPERM, unix.EACCES:
		return true
	default:
		return false
	}
}
