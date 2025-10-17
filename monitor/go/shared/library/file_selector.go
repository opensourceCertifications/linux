// Package library provides utility functions for file selection.
// It includes functionality to pick random binaries from common system directories.
package library

import (
	cryptorand "crypto/rand"
	"errors"
	"fmt"
	"io/fs"
	"math/big"
	"os"
	"path/filepath"
)

// PickRandomBinaries scans common system binary dirs, picks a random count between 15 and 20,
// then returns that many unique file paths (or fewer if not enough exist).
// - Non-existent directories are skipped silently.
// - Only regular files are considered (no dirs/symlinks).
// - Selection is without replacement.
// helper: collect direct regular files (and symlinks -> regular) from one dir
func collectBinariesFromDir(dir string, seen map[string]struct{}, out *[]string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		// Skip unreadable/missing directories without failing the whole op.
		return
	}
	for _, e := range entries {
		full := filepath.Join(dir, e.Name())

		// Fast path: direct regular file
		if e.Type().IsRegular() {
			if _, ok := seen[full]; !ok {
				seen[full] = struct{}{}
				*out = append(*out, full)
			}
			continue
		}

		// If symlink, include only when it targets a regular file
		if e.Type()&fs.ModeSymlink != 0 {
			if fi, err := os.Stat(full); err == nil && fi.Mode().IsRegular() {
				if _, ok := seen[full]; !ok {
					seen[full] = struct{}{}
					*out = append(*out, full)
				}
			}
		}
	}
}

// PickRandomBinaries selects between 15 and 20 unique binary file paths
func PickRandomBinaries() ([]string, error) {
	dirs := []string{
		"/usr/local/bin",
		"/usr/bin",
		"/usr/local/sbin", // keep; tolerate common typo below
		"/usr/local/sbing",
		"/usr/sbin",
	}

	seen := make(map[string]struct{})
	all := make([]string, 0, 1024)

	for _, d := range dirs {
		collectBinariesFromDir(d, seen, &all)
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
