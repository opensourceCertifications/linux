// Package library provides functions to corrupt files by flipping bits and adding an offset.
package library

import (
	"log"
	"math/rand"
	"os"
	"time"
)

func clampPercent(p int) int {
	if p <= 0 || p > 100 {
		return 100
	}
	return p
}

func corruptAll(data []byte, offset byte) {
	for i := range data {
		data[i] = (data[i] ^ 0xFF) + offset
	}
}

func corruptSample(data []byte, percent int, offset byte, r *rand.Rand) {
	n := len(data)
	k := n * percent / 100
	if k == 0 && percent > 0 {
		k = 1
	}
	perm := r.Perm(n)
	for i := 0; i < k; i++ {
		idx := perm[i]
		data[idx] = (data[idx] ^ 0xFF) + offset
	}
}

// CorruptFile corrupts a file at the given path by flipping bits and adding an offset to a percentage of its bytes.
// It returns the path of the corrupted file or an error if something goes wrong.
func CorruptFile(path string, percent int) (string, error) {
	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		log.Printf("failed to read file %s: %v", path, err)
		return "", err
	}
	if len(data) == 0 {
		log.Printf("file %s is empty, nothing to corrupt", path)
		// Return a real error instead of nil
		return "", os.ErrInvalid
	}

	// Save original modtime
	info, err := os.Stat(path)
	if err != nil {
		log.Printf("failed to stat file %s: %v", path, err)
		return "", err
	}
	origModTime := info.ModTime()

	// Normalize percent and set up local RNG (no deprecated rand.Seed)
	percent = clampPercent(percent)
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	offset := byte(r.Intn(256))

	// Corrupt bytes
	if percent == 100 {
		corruptAll(data, offset)
	} else {
		corruptSample(data, percent, offset, r)
	}

	// Write back
	if err := os.WriteFile(path, data, info.Mode()); err != nil {
		log.Printf("failed to write file %s: %v", path, err)
		return "", err
	}

	// Restore original modtime
	if err := os.Chtimes(path, origModTime, origModTime); err != nil {
		log.Printf("failed to restore times on file %s: %v", path, err)
		return "", err
	}

	log.Printf("File %s corrupted (flip+offset, %d%% of bytes)", path, percent)
	// Return the path as the "status" so callers have a useful value.
	return path, nil
}
