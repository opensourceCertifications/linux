// Package library provides functions to corrupt files by flipping bits and adding an offset.
package library

import (
	crand "crypto/rand"
	"log"
	"math/big"
	"os"
)

func clampPercent(p int) int {
	if p <= 0 || p > 100 {
		return 100
	}
	return p
}

// func corruptAll(data []byte, offset byte) {
//	for i := range data {
//		data[i] = (data[i] ^ 0xFF) + offset
//	}
//}
//
// func corruptSample(data []byte, percent int, offset byte, r *rand.Rand) {
//	n := len(data)
//	k := n * percent / 100
//	if k == 0 && percent > 0 {
//		k = 1
//	}
//	perm := r.Perm(n)
//	for i := 0; i < k; i++ {
//		idx := perm[i]
//		data[idx] = (data[idx] ^ 0xFF) + offset
//	}
//}

// CorruptPercent overwrites approximately percent% of bytes (at least 1 if percent>0).
func CorruptPercent(data []byte, percent int) error {
	n := len(data)
	if n == 0 || percent <= 0 {
		return nil
	}
	if percent >= 100 {
		_, err := crand.Read(data)
		return err
	}

	// Compute how many bytes to corrupt (at least 1 if percent > 0)
	k := n * percent / 100
	if k >= n {
		_, err := crand.Read(data)
		return err
	}

	// Choose k unique indices using a partial Fisher–Yates shuffle
	// Build index list 0..n-1
	idx := make([]int, n)
	for i := 0; i < n; i++ {
		idx[i] = i
	}
	// Randomly swap to select the last k positions as our unique picks
	for i := n - 1; i >= n-k; i-- {
		jBig, err := crand.Int(crand.Reader, big.NewInt(int64(i+1))) // 0..i
		if err != nil {
			return err
		}
		j := int(jBig.Int64())
		idx[i], idx[j] = idx[j], idx[i]
	}

	// Overwrite those k positions with cryptographically random bytes
	for i := n - k; i < n; i++ {
		p := idx[i]
		if _, err := crand.Read(data[p : p+1]); err != nil {
			return err
		}
	}
	return nil
}

// CorruptFile corrupts a file at the given path by flipping bits and adding an offset to a percentage of its bytes.
// It returns the path of the corrupted file or an error if something goes wrong.
func CorruptFile(path string, percent int) (string, error) {
	// Read file
	// #nosec G304 // We want to read arbitrary files
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
	// r := rand.New(rand.NewSource(time.Now().UnixNano()))
	// offset := byte(r.Intn(256))

	// Corrupt bytes
	err = CorruptPercent(data, percent)
	if err != nil {
		log.Printf("failed to corrupt data for file %s: %v", path, err)
		return "", err
	}

	// if percent == 100 {
	//	corruptAll(data, offset)
	// } else {
	//	corruptSample(data, percent, offset, r)
	//}

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
