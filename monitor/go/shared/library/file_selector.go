// shared/library/filepicker.go
package library

import (
	cryptorand "crypto/rand"
	"errors"
	"fmt"
	"io/fs"
	"math/big"
	"math/rand"
	"os"
	"path/filepath"
	"time"
)

// PickRandomBinaries scans common system binary dirs, picks a random count between 15 and 20,
// then returns that many unique file paths (or fewer if not enough exist).
// - Non-existent directories are skipped silently.
// - Only regular files are considered (no dirs/symlinks).
// - Selection is without replacement.
func PickRandomBinaries() ([]string, error) {
	dirs := []string{
		"/usr/local/bin",
		"/usr/bin",
		"/usr/local/sbin", // (note: tolerate common typo below)
		"/usr/local/sbing",
		"/usr/sbin",
	}

	var all []string
	seen := make(map[string]struct{})

	for _, d := range dirs {
		entries, err := os.ReadDir(d)
		if err != nil {
			// Skip missing or unreadable directories without failing the whole operation.
			continue
		}
		for _, e := range entries {
			// Only direct children; do not recurse.
			if !e.Type().IsRegular() {
				// If it's a symlink, resolve to file info and include only if it points to a regular file.
				if e.Type()&fs.ModeSymlink != 0 {
					full := filepath.Join(d, e.Name())
					if fi, err := os.Stat(full); err == nil && fi.Mode().IsRegular() {
						if _, ok := seen[full]; !ok {
							seen[full] = struct{}{}
							all = append(all, full)
						}
					}
				}
				continue
			}
			full := filepath.Join(d, e.Name())
			if _, ok := seen[full]; !ok {
				seen[full] = struct{}{}
				all = append(all, full)
			}
		}
	}

	if len(all) == 0 {
		return nil, errors.New("no candidate files found in target directories")
	}

	// Random K in [15,20], clamped to available file count.
	k, err := randIntInclusive(15, 20)
	if err != nil {
		return nil, fmt.Errorf("rng failure: %w", err)
	}
	if k > len(all) {
		k = len(all)
	}

	// Shuffle and take first k.
	seedFastRNG()
	rand.Shuffle(len(all), func(i, j int) { all[i], all[j] = all[j], all[i] })

	return all[:k], nil
}

// randIntInclusive returns a cryptographically-strong random int in [min, max].
func randIntInclusive(min, max int) (int, error) {
	if max < min {
		return 0, fmt.Errorf("invalid range %d..%d", min, max)
	}
	span := max - min + 1
	nBig, err := cryptorand.Int(cryptorand.Reader, big.NewInt(int64(span)))
	if err != nil {
		return 0, err
	}
	return min + int(nBig.Int64()), nil
}

// seedFastRNG seeds math/rand for shuffling with a high-entropy seed.
func seedFastRNG() {
	if _, ok := rand.Read(make([]byte, 1)); ok == nil {
		// On Go 1.20+, rand.Read uses a global, already-seeded source — nothing to do.
		return
	}
	rand.Seed(time.Now().UnixNano())
}
