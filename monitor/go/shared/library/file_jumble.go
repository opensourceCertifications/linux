// Package library provides file manipulation utilities.
// It includes functions for cyclically jumbling file contents while preserving metadata.
// It ensures safe file operations by preventing path traversal and symlink attacks.
// It is designed for use in controlled environments where file integrity is critical.
package library

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
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

var errTooFewPaths = errors.New("need at least two paths for a cyclic jumble")

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
// CyclicJumble writes content of paths[i] into paths[(i+1)%n],
// preserving the destination's metadata captured prior to the swap.
func CyclicJumble(paths []string) error {
	if len(paths) < 2 {
		return errTooFewPaths
	}
	return doCyclicJumble(paths)
}

func doCyclicJumble(paths []string) error {
	if err := validatePaths(paths); err != nil {
		return err
	}
	destMeta, err := snapshotDestMeta(paths)
	if err != nil {
		return err
	}
	srcTemp, cleanup, err := stageAll(paths)
	if err != nil {
		return err
	}
	defer cleanup()
	return applyCycle(paths, srcTemp, destMeta)
}

// --- helpers (each kept simple) ---

func validatePaths(paths []string) error {
	seen := make(map[string]struct{}, len(paths))
	for i, p := range paths {
		if !filepath.IsAbs(p) {
			return fmt.Errorf("path %d not absolute: %s", i, p)
		}
		if _, dup := seen[p]; dup {
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
	return nil
}

func snapshotDestMeta(paths []string) (map[string]fileMeta, error) {
	destMeta := make(map[string]fileMeta, len(paths))
	for _, dest := range paths {
		m, err := captureMeta(dest)
		if err != nil {
			return nil, fmt.Errorf("capture meta %s: %w", dest, err)
		}
		destMeta[dest] = m
	}
	return destMeta, nil
}

func stageAll(paths []string) (map[string]string, func(), error) {
	srcTemp := make(map[string]string, len(paths))
	cleanup := func() {
		for _, t := range srcTemp {
			_ = os.Remove(t)
		}
	}
	for _, src := range paths {
		tmp, err := stageCopy(src)
		if err != nil {
			cleanup()
			return nil, func() {}, fmt.Errorf("stage copy %s: %w", src, err)
		}
		srcTemp[src] = tmp
	}
	return srcTemp, cleanup, nil
}

func applyCycle(paths []string, srcTemp map[string]string, destMeta map[string]fileMeta) error {
	n := len(paths)
	for i, src := range paths {
		dest := paths[(i+1)%n]
		if err := writeOverWithMeta(srcTemp[src], dest, destMeta[dest]); err != nil {
			return fmt.Errorf("apply to %s: %w", dest, err)
		}
	}
	return nil
}

// readXattrs returns all xattrs for path (best-effort).
// It uses the size-probe pattern (nil buffer first) to avoid inner retry loops.
func readXattrs(path string) (map[string][]byte, error) {
	out := make(map[string][]byte)

	// 1) List names
	n, err := unix.Listxattr(path, nil)
	if err != nil || n <= 0 {
		// Not supported / none present -> best-effort: return what we have
		return out, err
	}
	buf := make([]byte, n)
	n, err = unix.Listxattr(path, buf)
	if err != nil || n <= 0 {
		return out, err
	}

	// 2) For each name, size-probe then fetch value
	for _, name := range splitZ(buf[:n]) {
		sz, gerr := unix.Getxattr(path, name, nil)
		if gerr != nil || sz <= 0 {
			continue
		}
		val := make([]byte, sz)
		got, gerr := unix.Getxattr(path, name, val)
		if gerr == nil && got >= 0 {
			out[name] = append([]byte(nil), val[:got]...)
		}
	}
	return out, nil
}

// captureMeta captures the file metadata of path (mode, uid/gid, atime/mtime, xattrs).
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

	// atime/mtime from Stat_t (Linux)
	m.Atime = time.Unix(int64(st.Atim.Sec), int64(st.Atim.Nsec))
	m.Mtime = time.Unix(int64(st.Mtim.Sec), int64(st.Mtim.Nsec))

	// xattrs (best-effort)
	if xa, xerr := readXattrs(path); xerr == nil {
		m.XAttr = xa
	} else {
		// keep best-effort behavior; if you prefer, log/trace xerr
		m.XAttr = map[string][]byte{}
	}

	return m, nil
}

// stageCopy writes src's content into a same-dir temp file, fsyncs it,
// closes both files (checking errors), and returns the temp path.
func stageCopy(src string) (tmpPath string, err error) {
	dir := filepath.Dir(src)

	tmp, e := os.CreateTemp(dir, ".jumble-src-*")
	if e != nil {
		return "", e
	}
	tmpPath = tmp.Name()

	// Always close tmp; if close fails and no prior error, surface it.
	// Also remove the temp file on any error.
	defer func() {
		if cerr := tmp.Close(); cerr != nil && err == nil {
			err = cerr
		}
		if err != nil {
			_ = os.Remove(tmpPath)
		}
	}()

	// ✅ Hardened open: validates path under allowed roots + forbids symlinks
	sf, e := safeOpenRead(src)
	if e != nil {
		err = e
		return
	}
	defer func() {
		if cerr := sf.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	if _, e = io.Copy(tmp, sf); e != nil {
		err = e
		return
	}
	if e = tmp.Sync(); e != nil {
		err = e
		return
	}

	// Success: err == nil; deferred cleanup won't remove the temp file.
	return tmpPath, nil
}

var allowedPrefixes = []string{
	"/usr/bin/",
	"/usr/sbin/",
	"/usr/local/bin/",
	"/usr/local/sbin/",
	"/boot/",
}

func allowedPath(p string) bool {
	p = filepath.Clean(p)
	for _, pre := range allowedPrefixes {
		if strings.HasPrefix(p, pre) {
			return true
		}
	}
	return false
}

// chooseAllowedRoot returns the matching allowed root and the relative path.
func chooseAllowedRoot(clean string) (root, rel string, err error) {
	for _, pre := range allowedPrefixes {
		if strings.HasPrefix(clean, pre) {
			root = pre
			rel = strings.TrimPrefix(strings.TrimPrefix(clean, pre), "/")
			return
		}
	}
	return "", "", fmt.Errorf("no matching allowed root for %s", clean)
}

func isOpenat2Unsupported(err error) bool {
	return errors.Is(err, unix.ENOSYS) || errors.Is(err, unix.EINVAL)
}

func fileFromFdIfRegular(fd int, name string) (*os.File, error) {
	var st unix.Stat_t
	if err := unix.Fstat(fd, &st); err != nil {
		_ = unix.Close(fd)
		return nil, err
	}
	if st.Mode&unix.S_IFMT != unix.S_IFREG {
		_ = unix.Close(fd)
		return nil, fmt.Errorf("not a regular file: %s", name)
	}
	return os.NewFile(uintptr(fd), name), nil
}

// safeOpenRead opens p read-only while preventing path traversal & symlinks.
func safeOpenRead(p string) (*os.File, error) {
	clean := filepath.Clean(p)
	if !allowedPath(clean) {
		return nil, fmt.Errorf("disallowed path: %s", clean)
	}

	root, rel, err := chooseAllowedRoot(clean)
	if err != nil {
		return nil, err
	}

	rootfd, err := unix.Open(root, unix.O_PATH|unix.O_DIRECTORY|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, fmt.Errorf("open root %s: %w", root, err)
	}
	defer func() { _ = unix.Close(rootfd) }()

	how := &unix.OpenHow{
		Flags:   uint64(unix.O_RDONLY | unix.O_CLOEXEC),
		Resolve: unix.RESOLVE_BENEATH | unix.RESOLVE_NO_SYMLINKS | unix.RESOLVE_NO_MAGICLINKS,
	}
	fd, err := unix.Openat2(rootfd, rel, how)
	if err == nil {
		return fileFromFdIfRegular(fd, clean)
	}
	if isOpenat2Unsupported(err) {
		return fallbackSafeOpen(filepath.Join(root, rel))
	}
	return nil, err
}

func fallbackSafeOpen(abs string) (*os.File, error) {
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return nil, fmt.Errorf("resolve symlinks: %w", err)
	}
	if !allowedPath(resolved) {
		return nil, fmt.Errorf("symlink escapes allowed roots: %s -> %s", abs, resolved)
	}
	fd, err := unix.Open(resolved, unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		if errors.Is(err, unix.ELOOP) {
			return nil, fmt.Errorf("refusing symlink final component: %s", resolved)
		}
		return nil, err
	}
	var st unix.Stat_t
	if err := unix.Fstat(fd, &st); err != nil {
		_ = unix.Close(fd)
		return nil, err
	}
	if st.Mode&unix.S_IFMT != unix.S_IFREG {
		_ = unix.Close(fd)
		return nil, fmt.Errorf("not a regular file: %s", resolved)
	}
	return os.NewFile(uintptr(fd), resolved), nil
}

// createTempSibling creates a temp file in the same dir as dest.
// It returns the file, its path, and a cleanup that will remove the temp
// if an error occurs (based on the named return in the caller).
func createTempSibling(dest string) (f *os.File, tmpPath string, cleanup func(*error), err error) {
	dir := filepath.Dir(dest)
	f, err = os.CreateTemp(dir, ".jumble-dst-*")
	if err != nil {
		return nil, "", nil, err
	}
	tmpPath = f.Name()

	cleanup = func(retErr *error) {
		_ = f.Close()
		// If we failed, or the temp still exists after the caller returns, remove it.
		if retErr != nil && *retErr != nil {
			_ = os.Remove(tmpPath)
			return
		}
		if _, stErr := os.Lstat(tmpPath); stErr == nil {
			_ = os.Remove(tmpPath)
		}
	}
	return
}

func copyFromPath(dst *os.File, srcPath string) (err error) {
	sf, err := safeOpenRead(srcPath)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := sf.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	_, err = io.Copy(dst, sf)
	return
}

// Apply mode/owner and xattrs to a path (pre-rename).
func applyPreMeta(path string, meta fileMeta) error {
	if err := os.Chmod(path, meta.Mode); err != nil {
		return err
	}
	if err := os.Chown(path, meta.UID, meta.GID); err != nil {
		return err
	}
	for k, v := range meta.XAttr {
		if err := unix.Setxattr(path, k, v, 0); err != nil && !isIgnorableXErr(err) {
			// only *non-ignorable* xattr errors should fail the operation
			return err
		}
	}
	return nil
}

// Apply timestamps after rename (ctime cannot be set).
func applyPostMeta(path string, meta fileMeta) error {
	return os.Chtimes(path, meta.Atime, meta.Mtime)
}

// isIgnorableXErr returns true if an xattr get/set error can be safely ignored.
func isIgnorableXErr(err error) bool {
	if err == nil {
		return true
	}
	// NOTE: On Linux ENOTSUP == EOPNOTSUPP; checking ENOTSUP covers both.
	return errors.Is(err, unix.ENOTSUP) || // covers EOPNOTSUPP too
		errors.Is(err, unix.EPERM) ||
		errors.Is(err, unix.EACCES) ||
		errors.Is(err, unix.EINVAL) ||
		errors.Is(err, unix.ENODATA) // "no attribute" on Linux
}

// writeOverWithMeta replaces dest with the contents of stagedSrc, restoring dest's original metadata.
func writeOverWithMeta(stagedSrc, dest string, meta fileMeta) (retErr error) {
	tmp, tmpPath, cleanup, err := createTempSibling(dest)
	if err != nil {
		return err
	}
	defer cleanup(&retErr)

	if err := copyFromPath(tmp, stagedSrc); err != nil {
		return err
	}
	if err := tmp.Sync(); err != nil {
		return err
	}
	if err := applyPreMeta(tmpPath, meta); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, dest); err != nil {
		return err
	}
	if err := applyPostMeta(dest, meta); err != nil {
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

// func isIgnorableXErr(err error) bool {
//	// On Linux ENOTSUP == EOPNOTSUPP; keep only ENOTSUP to avoid duplicate cases.
//	switch err {
//	case unix.ENOTSUP, unix.EPERM, unix.EACCES:
//		return true
//	default:
//		return false
//	}
//}
