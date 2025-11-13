// Package library provides utility functions for file selection.
// It includes functionality to pick random binaries from common system directories.
package library

import (
	cryptorand "crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
)

var denyExact = map[string]struct{}{
	// Dynamic loader(s)
	"/lib64/ld-linux-x86-64.so.2": {},
	"/lib/ld-linux.so.2":          {}, // 32-bit interpreter if present

	// C runtime (glibc)
	"/lib64/libc.so.6": {},

	// Auth & linker config
	"/etc/passwd":  {},
	"/etc/sudoers": {},
}

var denyPrefixes = []string{
	// PAM stack & sudo includes
	"/etc/pam.d/",
}

// Extra pattern checks for versioned realfiles (glibc/ld)
func isVersionedPlatformBinary(path string) bool {
	base := filepath.Base(path)
	dir := filepath.Dir(path)
	if dir == "/lib64" && strings.HasPrefix(base, "ld-") && strings.HasSuffix(base, ".so") {
		return true // e.g. /lib64/ld-2.34.so
	}
	if dir == "/lib64" && strings.HasPrefix(base, "libc-") && strings.HasSuffix(base, ".so") {
		return true // e.g. /lib64/libc-2.34.so
	}
	// Cross-distro (Debian/Ubuntu style loader)
	if strings.HasSuffix(path, "/x86_64-linux-gnu/ld-linux-x86-64.so.2") {
		return true
	}
	return false
}

func isDangerous(path string) bool {
	// Normalize
	path = filepath.Clean(path)

	// Exact hits
	if _, ok := denyExact[path]; ok {
		return true
	}
	// Prefix hits
	for _, pre := range denyPrefixes {
		if strings.HasPrefix(path, pre) {
			return true
		}
	}
	// Versioned loader/glibc realfiles
	if isVersionedPlatformBinary(path) {
		return true
	}
	return false
}

// resolveRegularTarget follows symlinks starting at startPath until it reaches
// detects cycles, and enforces a maximum symlink depth.
func resolveRegularTarget(startPath string, maxSymlinkDepth int) (string, error) {
	normalizedStart := filepath.Clean(startPath)
	if !filepath.IsAbs(normalizedStart) {
		return "", fmt.Errorf("path is not absolute: %s", normalizedStart)
	}

	currentPath := normalizedStart
	visitedPaths := make(map[string]struct{}, 4)

	for depth := 0; depth < maxSymlinkDepth; depth++ {
		fileInfo, err := os.Lstat(currentPath)
		if err != nil {
			return "", fmt.Errorf("lstat %s: %w", currentPath, err)
		}

		// Success: reached a regular file
		if fileInfo.Mode().IsRegular() {
			return currentPath, nil
		}

		// Only follow symlinks
		if fileInfo.Mode()&os.ModeSymlink == 0 {
			return "", fmt.Errorf("not a regular file or symlink (mode=%s): %s",
				fileInfo.Mode().String(), currentPath)
		}

		// Detect cycles
		if _, seen := visitedPaths[currentPath]; seen {
			return "", fmt.Errorf("symlink cycle detected at %s", currentPath)
		}
		visitedPaths[currentPath] = struct{}{}

		// Readlink and resolve relative targets against the link's directory
		linkTarget, err := os.Readlink(currentPath)
		if err != nil {
			return "", fmt.Errorf("readlink %s: %w", currentPath, err)
		}
		if !filepath.IsAbs(linkTarget) {
			linkTarget = filepath.Join(filepath.Dir(currentPath), linkTarget)
		}
		nextPath := filepath.Clean(linkTarget)

		currentPath = nextPath
	}

	return "", fmt.Errorf("symlink depth exceeded (%d) for %s", maxSymlinkDepth, currentPath)
}

// collectBinariesFromDir scans a directory, follows symlinks safely to their
// regular-file targets, and appends unique resolved paths into results.
// De-duplication is done on the resolved (final) path via seenTargets.
// collectBinariesFromDir scans a directory, resolves symlinks safely to a
// regular-file target, appends unique resolved paths into results, and
// returns a joined error capturing any per-entry failures encountered.
func collectBinariesFromDir(
	directory string,
	seenTargets map[string]struct{},
	results *[]string,
) error {

	dirEntries, err := os.ReadDir(directory)
	if err != nil {
		// Directory missing/unreadable is a real error for the caller to decide on.
		return fmt.Errorf("read dir %s: %w", directory, err)
	}

	var errs []error

	for _, entry := range dirEntries {
		candidatePath := filepath.Join(directory, entry.Name())

		resolvedPath, rerr := resolveRegularTarget(candidatePath, 16) // depth cap
		if rerr != nil {
			// Keep going, but remember the error.
			errs = append(errs, fmt.Errorf("%s: %w", candidatePath, rerr))
			continue
		}

		if _, alreadyAdded := seenTargets[resolvedPath]; alreadyAdded {
			continue
		}
		seenTargets[resolvedPath] = struct{}{}
		if !isDangerous(resolvedPath) {
			*results = append(*results, resolvedPath)
		}
	}

	// Fold per-entry errors into one (nil if none)
	return errors.Join(errs...)
}

// PickRandomBinaries selects between 15 and 20 unique binary file paths
func PickRandomBinaries() ([]string, error) {
	dirs := []string{
		"/usr/bin",
		"/usr/sbin",
		"/sbin",
		"/bin",
	}

	seen := make(map[string]struct{})
	all := make([]string, 0, 1024)

	var aggErr error
	// for _, dir := range []string{"/usr/local/bin", "/usr/bin", "/usr/local/sbin", "/usr/local/sbing", "/usr/sbin"} {
	for _, dir := range dirs {
		if err := collectBinariesFromDir(dir, seen, &all); err != nil {
			aggErr = errors.Join(aggErr, err)
		}
	}

	// If you want to *log* all the issues but still proceed:
	if aggErr != nil {
		// Send to monitor (or log locally)
		return nil, fmt.Errorf("collector encountered issues: %w", aggErr)
	}

	if len(all) == 0 {
		return nil, errors.New("no candidate files found in target directories")
	}

	// Random K in [15,20], clamped to available count
	k, err := randIntInclusive(15, 20) // your existing crypto-based function
	if err != nil {
		return nil, fmt.Errorf("rng failure: %w", err)
	}
	if k > len(all) {
		k = len(all)
	}

	// Crypto-safe shuffle (no math/rand)
	if err := shuffleStringsCrypto(all); err != nil {
		return nil, fmt.Errorf("shuffle: %w", err)
	}
	return all[:k], nil
}

// randIntInclusive returns a cryptographically-strong random int in [min, max].
func randIntInclusive(low, high int) (int, error) {
	if high < low {
		return 0, fmt.Errorf("invalid range %d..%d", low, high)
	}
	span := high - low + 1
	nBig, err := cryptorand.Int(cryptorand.Reader, big.NewInt(int64(span)))
	if err != nil {
		return 0, err
	}
	return low + int(nBig.Int64()), nil
}

// shuffleStringsCrypto shuffles a slice of strings in place using crypto/rand.
func shuffleStringsCrypto(a []string) error {
	for i := len(a) - 1; i > 0; i-- {
		r, err := cryptorand.Int(cryptorand.Reader, big.NewInt(int64(i+1)))
		if err != nil {
			return err
		}
		j := int(r.Int64())
		a[i], a[j] = a[j], a[i]
	}
	return nil
}
