// Package library provides file corruption utilities.
package library

import (
	crand "crypto/rand"
	"errors"
	"fmt"
	"io"
	"math"
	"math/big"
	"os"
	"sort"
	"time"
)

// CorruptFile overwrites ~percent% of a file's bytes in-place using cryptographically random data.
// Contract:
//   - percent < 0  -> error
//   - percent == 0 -> no-op
//   - 0 < percent < 100 -> overwrite exactly k = max(1, floor(size*percent/100)) unique byte positions
//   - percent >= 100    -> full overwrite (exactly size bytes), without changing file size
//
// Implementation notes:
//   - No full-file loads; O(k) memory where k is the number of bytes to corrupt.
//   - Uses CSPRNG for both index selection and bytes.
//   - Coalesces adjacent positions to reduce syscalls.
//   - Restores mtime (atime best-effort via mtime for portability).
func CorruptFile(path string, percent int) (retErr error) {
	p, err := validatePercent(percent)
	if err != nil || p == 0 {
		return err // 0 => no-op
	}

	f, info, err := openRegular(path)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := f.Close(); cerr != nil {
			// don't lose the original error if there was one
			if retErr != nil {
				retErr = errors.Join(retErr, fmt.Errorf("close %q: %w", path, cerr))
			} else {
				retErr = fmt.Errorf("close %q: %w", path, cerr)
			}
		}
	}()
	size := info.Size()
	if size == 0 {
		return nil
	}
	origMod := info.ModTime()

	k := computeK(size, p)
	if k >= size {
		if err := doFullOverwrite(f, size); err != nil {
			return err
		}
	} else {
		if err := doPartialOverwrite(f, size, k); err != nil {
			return err
		}
	}
	return finalize(path, f, origMod)
}

// ---- helpers (small, focused) ----

func validatePercent(percent int) (int, error) {
	if percent < 0 {
		return 0, fmt.Errorf("percent must be >= 0, got %d", percent)
	}
	if percent == 0 {
		return 0, nil
	}
	if percent > 100 {
		return 100, nil
	}
	return percent, nil
}

func openRegular(path string) (*os.File, os.FileInfo, error) {
	// #nosec G304 -- path is constructed by our own target selector (not user input).
	f, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		return nil, nil, fmt.Errorf("open %q: %w", path, err)
	}
	info, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return nil, nil, fmt.Errorf("stat %q: %w", path, err)
	}
	if !info.Mode().IsRegular() {
		_ = f.Close()
		return nil, nil, fmt.Errorf("corrupt: %q is not a regular file", path)
	}
	return f, info, nil
}

func computeK(size int64, percent int) int64 {
	k := (size * int64(percent)) / 100
	if percent > 0 && k == 0 {
		k = 1 // min-1 rule for tiny files/percents
	}
	return k
}

func doFullOverwrite(f *os.File, size int64) error {
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("seek start: %w", err)
	}
	if _, err := io.CopyN(f, crand.Reader, size); err != nil {
		return fmt.Errorf("full overwrite: %w", err)
	}
	if err := f.Sync(); err != nil {
		return fmt.Errorf("fsync: %w", err)
	}
	// Defensive: ensure size unchanged
	if err := f.Truncate(size); err != nil {
		return fmt.Errorf("truncate: %w", err)
	}
	return nil
}

func doPartialOverwrite(f *os.File, total, k int64) error {
	// Guard: k must fit in 'int' for buffer allocation on this platform.
	if k > int64(math.MaxInt) {
		return fmt.Errorf("selection size too large for this platform (k=%d)", k)
	}

	positions, err := sampleKUniqueCrypto64(total, k)
	if err != nil {
		return fmt.Errorf("sample positions: %w", err)
	}

	// Sort to coalesce adjacent positions into runs (reduces syscalls).
	sort.Slice(positions, func(i, j int) bool { return positions[i] < positions[j] })

	randBytes := make([]byte, int(k))
	if _, err := crand.Read(randBytes); err != nil {
		return fmt.Errorf("read random bytes: %w", err)
	}

	runStart := 0
	for i := 1; i <= len(positions); i++ {
		// Continue run while consecutive
		if i < len(positions) && positions[i] == positions[i-1]+1 {
			continue
		}
		// Close and write current run [runStart, i)
		offset := positions[runStart]
		if _, err := f.WriteAt(randBytes[runStart:i], offset); err != nil {
			return fmt.Errorf("writeAt offset=%d len=%d: %w", offset, i-runStart, err)
		}
		runStart = i
	}

	if err := f.Sync(); err != nil {
		return fmt.Errorf("fsync: %w", err)
	}
	return nil
}

func finalize(path string, f *os.File, mtime time.Time) error {
	// Best-effort timestamp preservation (portable): set atime=mtime
	if err := os.Chtimes(path, mtime, mtime); err != nil {
		return fmt.Errorf("chtimes: %w", err)
	}
	// Flush the inode metadata update
	if err := f.Sync(); err != nil {
		return fmt.Errorf("fsync after chtimes: %w", err)
	}
	return nil
}

// ---- crypto-safe sampling utilities ----

// sampleKUniqueCrypto64 selects k unique integers from [0, total) using Floyd's algorithm,
// with cryptographically secure randomness. Runs in O(k) time/space.
func sampleKUniqueCrypto64(total, k int64) ([]int64, error) {
	if k <= 0 {
		return []int64{}, nil
	}
	if k > total {
		return nil, fmt.Errorf("k(%d) > total(%d)", k, total)
	}
	selected := make(map[int64]struct{}, min64(k, 1<<16)) // pre-cap; grows as needed
	for t := total - k; t < total; t++ {
		m, err := cryptoIntn64(t + 1) // uniform in [0..t]
		if err != nil {
			return nil, err
		}
		if _, exists := selected[m]; exists {
			selected[t] = struct{}{}
		} else {
			selected[m] = struct{}{}
		}
	}
	out := make([]int64, 0, len(selected))
	for v := range selected {
		out = append(out, v)
	}
	return out, nil
}

func cryptoIntn64(n int64) (int64, error) {
	if n <= 0 {
		return 0, fmt.Errorf("cryptoIntn64: n must be > 0, got %d", n)
	}
	x, err := crand.Int(crand.Reader, big.NewInt(n)) // 0..n-1
	if err != nil {
		return 0, err
	}
	return x.Int64(), nil
}

func min64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}
