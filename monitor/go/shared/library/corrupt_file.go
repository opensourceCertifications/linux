package library

import (
	"log"
	"math/rand"
	"os"
	"time"
)

func CorruptFile(path string, percent int) (status string, err error) {
	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		log.Printf("failed to read file %s: %v", path, err)
		return "", err
	}
	if len(data) == 0 {
		log.Printf("file %s is empty, nothing to corrupt", path)
		return "", err
	}

	// Save original modtime
	info, err := os.Stat(path)
	if err != nil {
		log.Printf("failed to stat file %s: %v", path, err)
		return "", err
	}
	origModTime := info.ModTime()

	// Normalize percent -> [1..100]; default to 100 if out of range
	if percent <= 0 || percent > 100 {
		percent = 100
	}

	// Pick one random offset (0–255) and seed RNG
	rand.Seed(time.Now().UnixNano())
	offset := byte(rand.Intn(256))

	// Corrupt bytes
	if percent == 100 {
		// Fast path: all bytes
		for i, b := range data {
			data[i] = (b ^ 0xFF) + offset
		}
	} else {
		// Sample ~percent% unique positions without replacement
		n := len(data)
		k := n * percent / 100
		if k == 0 && percent > 0 { // ensure we touch at least 1 byte for tiny files
			k = 1
		}
		// Get a random permutation of indices and take first k
		perm := rand.Perm(n)
		for i := 0; i < k; i++ {
			idx := perm[i]
			data[idx] = (data[idx] ^ 0xFF) + offset
		}
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
	return
}
