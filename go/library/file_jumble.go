// Package library provides a simple "cyclic jumble" for regular files.
// It filters input paths to real regular files, shuffles, then cycles content.
// Destination files keep their original metadata (mode, uid/gid, xattrs best-effort, atime/mtime).
package library

import (
	"bytes"
	datatypes "chaos-agent/library/types"
	cryptorand "crypto/rand"
	"errors"
	"fmt"
	"io"
	"math/big"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

// CyclicJumble takes absolute file paths, filters to real regular files via validatePaths,
// shuffles them using crypto/rand, then performs a cycle so that paths[i]’s content
// becomes paths[(i+1)%n], while preserving each destination’s original metadata.
func CyclicJumble(paths []string) error {
	paths = validatePaths(paths)
	if len(paths) < 2 {
		return errors.New("need at least two real regular files after validation")
	}

	// Snapshot destination metadata (what we’ll restore after writing).
	destMeta := make(map[string]datatypes.FileMeta, len(paths))
	for _, p := range paths {
		m, err := captureMeta(p)
		if err != nil {
			return fmt.Errorf("capture meta %s: %w", p, err)
		}
		destMeta[p] = m
	}

	// Stage all sources to temp copies so we can overwrite freely.
	staged, cleanup, err := stageAll(paths)
	if err != nil {
		return err
	}
	defer cleanup()

	// Secure shuffle using crypto/rand (Fisher–Yates).
	if err := shuffleCrypto(paths); err != nil {
		return fmt.Errorf("shuffle: %w", err)
	}

	// Apply the cycle: write staged content of paths[i] into paths[(i+1)%n]
	// while restoring the destination’s original metadata.
	n := len(paths)
	for i := 0; i < n; i++ {
		src := paths[i]
		dst := paths[(i+1)%n]
		if err := writeOverWithMeta(staged[src], dst, destMeta[dst]); err != nil {
			return fmt.Errorf("apply to %s: %w", dst, err)
		}
	}
	return nil
}

// validatePaths returns a new slice containing only absolute, existing, regular files.
// Symlinks, dirs, missing paths, devices, FIFOs, etc. are discarded.
func validatePaths(paths []string) []string {
	out := make([]string, 0, len(paths))
	seen := make(map[string]struct{}, len(paths))

	for _, p := range paths {
		if p == "" || !filepath.IsAbs(p) {
			continue
		}
		if _, ok := seen[p]; ok {
			continue
		}
		// Lstat to ensure we don’t follow symlinks.
		fi, err := os.Lstat(p)
		if err != nil {
			continue
		}
		if fi.Mode()&os.ModeSymlink != 0 {
			continue
		}
		if !fi.Mode().IsRegular() {
			continue
		}
		out = append(out, p)
		seen[p] = struct{}{}
	}
	return out
}

/* -------------------- internals (simple + Linux-friendly) -------------------- */
func captureMeta(path string) (datatypes.FileMeta, error) {
	var m datatypes.FileMeta
	fi, err := os.Lstat(path)
	if err != nil {
		return m, err
	}
	if !fi.Mode().IsRegular() {
		return m, fmt.Errorf("not a regular file: %s", path)
	}
	m.Mode = fi.Mode()

	st, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		return m, errors.New("unexpected stat type")
	}
	m.UID = int(st.Uid)
	m.GID = int(st.Gid)

	// atime/mtime from Stat_t (Linux)
	m.Atime = time.Unix(int64(st.Atim.Sec), int64(st.Atim.Nsec))
	m.Mtime = time.Unix(int64(st.Mtim.Sec), int64(st.Mtim.Nsec))

	// xattrs: best-effort
	if xa, err := readXattrs(path); err == nil {
		m.XAttr = xa
	} else {
		m.XAttr = map[string][]byte{}
	}
	return m, nil
}

// stageAll creates temp copies of each source file’s content.
// Returns a map[srcPath]tmpCopy and a cleanup func.
func stageAll(paths []string) (map[string]string, func(), error) {
	out := make(map[string]string, len(paths))
	cleanup := func() {
		for _, t := range out {
			_ = os.Remove(t)
		}
	}
	for _, src := range paths {
		tmp, err := stageCopy(src)
		if err != nil {
			cleanup()
			return nil, func() {}, fmt.Errorf("stage copy %s: %w", src, err)
		}
		out[src] = tmp
	}
	return out, cleanup, nil
}

// stageCopy copies src into a temp file in the same directory (same filesystem).
func stageCopy(src string) (tmpPath string, err error) {
	dir := filepath.Dir(src)
	fTmp, err := os.CreateTemp(dir, ".jumble-src-*")
	if err != nil {
		return "", err
	}
	tmpPath = fTmp.Name()
	//nolint:gosec // G304: srcPath is validated by validatePaths
	fSrc, err := os.Open(src)
	if err != nil {
		_ = os.Remove(tmpPath)
		return "", err
	}
	defer func() {
		if closeErr := fTmp.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("closing temp %q: %w", tmpPath, closeErr))
		}
	}()

	if _, err := io.Copy(fTmp, fSrc); err != nil {
		_ = os.Remove(tmpPath)
		return "", err
	}
	if err := fTmp.Sync(); err != nil {
		_ = os.Remove(tmpPath)
		return "", err
	}
	return tmpPath, nil
}

// small helper to keep writeOverWithMeta simple
func copyPathToFile(srcPath string, dst *os.File) (err error) {
	//nolint:gosec // G304: srcPath is validated by validatePaths
	src, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	// defer src.Close()
	defer func() {
		if closeErr := src.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("closing temp %q: %w", srcPath, closeErr))
		}
	}()
	_, err = io.Copy(dst, src)
	return err
}

// writeOverWithMeta writes stagedSrc content into dest, applying meta before/after.
func writeOverWithMeta(stagedSrc, dest string, meta datatypes.FileMeta) error {
	dir := filepath.Dir(dest)
	tmp, err := os.CreateTemp(dir, ".jumble-dst-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	// Always close and attempt to remove the temp; after a successful rename
	// the remove is a no-op (ENOENT), which we ignore. This avoids branching.
	defer func() {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
	}()

	// Copy content and flush
	if err := copyPathToFile(stagedSrc, tmp); err != nil {
		return err
	}
	if err := tmp.Sync(); err != nil {
		return err
	}

	// Pre-rename metadata, then atomic replace
	if err := applyPreMeta(tmpPath, meta); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, dest); err != nil {
		return err
	}

	// Post-rename times
	if err := applyPostMeta(dest, meta); err != nil {
		return fmt.Errorf("renamed but failed to restore times: %w", err)
	}
	return nil
}

func applyPreMeta(path string, meta datatypes.FileMeta) error {
	// Ownership (ignore EPERM/EACCES so non-root still works best-effort)
	if err := os.Chown(path, meta.UID, meta.GID); err != nil {
		if !errors.Is(err, unix.EPERM) && !errors.Is(err, unix.EACCES) {
			return err
		}
	}
	// Mode (preserve suid/sgid/sticky if present)
	const keep = os.ModePerm | os.ModeSetuid | os.ModeSetgid | os.ModeSticky
	if err := os.Chmod(path, meta.Mode&keep); err != nil {
		return err
	}
	// xattrs: best-effort; ignore common non-fatal errors
	for k, v := range meta.XAttr {
		if err := unix.Setxattr(path, k, v, 0); err != nil && !isIgnorableXErr(err) {
			return err
		}
	}
	return nil
}

func applyPostMeta(path string, meta datatypes.FileMeta) error {
	return os.Chtimes(path, meta.Atime, meta.Mtime)
}

/* ------------------------------- helpers ----------------------------------- */

func shuffleCrypto(a []string) error {
	// Fisher–Yates with crypto/rand
	n := len(a)
	for i := n - 1; i > 0; i-- {
		jBig, err := cryptorand.Int(cryptorand.Reader, big.NewInt(int64(i+1)))
		if err != nil {
			return err
		}
		j := int(jBig.Int64())
		a[i], a[j] = a[j], a[i]
	}
	return nil
}

func readXattrs(path string) (map[string][]byte, error) {
	out := make(map[string][]byte)
	n, err := unix.Listxattr(path, nil)
	if err != nil || n <= 0 {
		return out, err
	}
	buf := make([]byte, n)
	n, err = unix.Listxattr(path, buf)
	if err != nil || n <= 0 {
		return out, err
	}
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

func isIgnorableXErr(err error) bool {
	if err == nil {
		return true
	}
	// On Linux ENOTSUP == EOPNOTSUPP; checking ENOTSUP covers both.
	return errors.Is(err, unix.ENOTSUP) ||
		errors.Is(err, unix.EPERM) ||
		errors.Is(err, unix.EACCES) ||
		errors.Is(err, unix.EINVAL) ||
		errors.Is(err, unix.ENODATA)
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
