// Description: This program searches for Linux kernel files in the /boot directory and corrupts one of them to simulate a broken bootloader scenario. It reports its actions to a monitoring server.
package main

import (
	"chaos-agent/library"
	"crypto/rand"
	"fmt"
	"log"
	"math/big"
	"os"
	"path/filepath"
	"strconv"
)

var (
	MonitorIP      string
	MonitorPortStr string
	MonitorPort    int
	EncryptionKey  string
	Token          string// = library.GenerateToken(16)
)

func init() {
	if MonitorPortStr != "" {
		p, err := strconv.Atoi(MonitorPortStr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid MonitorPortStr %q: %v\n", MonitorPortStr, err)
		} else {
			MonitorPort = p
		}
	}
	var err error
	Token, err = library.GenerateToken(16)
	if err != nil {
		log.Fatalf("failed to generate token: %v", err)
	}
	library.SendMessage(MonitorIP, MonitorPort, "init", Token, Token, EncryptionKey)
}

// randIndex returns a uniform random int in [0, n).
func randIndex(n int) (int, error) {
	if n <= 0 {
		return 0, fmt.Errorf("empty set")
	}
	k, err := rand.Int(rand.Reader, big.NewInt(int64(n)))
	if err != nil {
		return 0, fmt.Errorf("crypto rand: %w", err)
	}
	return int(k.Int64()), nil
}

func main() {
	patterns := []string{
		"/boot/vmlinuz-*",
		"/boot/initramfs-*.img",
		"/boot/grub2/grub.cfg", // exact paths are fine: Glob returns it if it exists
		"/boot/loader/entries/*.conf",
		"/boot/grub2/grubenv",
	}

	vmlinuzFiles := make([]string, 0, 64)
	for _, pat := range patterns {
		if matches, _ := filepath.Glob(pat); len(matches) > 0 {
			vmlinuzFiles = append(vmlinuzFiles, matches...)
		}
	}

	library.SendMessage(MonitorIP, MonitorPort, "chaos_report", fmt.Sprintf("found vmlinuz files: %v", vmlinuzFiles), Token, EncryptionKey)
	if len(vmlinuzFiles) == 0 {
		library.SendMessage(MonitorIP, MonitorPort, "chaos_report",
			"no matching kernel/initramfs/grub files found", Token, EncryptionKey)
		log.Fatalf("no candidate files to corrupt")
	}

	idx, err := randIndex(len(vmlinuzFiles))
	if err != nil {
		log.Fatalf("random index failed: %v", err)
	}
	file := vmlinuzFiles[idx]
	err = library.CorruptFile(file, 100)
	if err != nil {
		library.SendMessage(MonitorIP, MonitorPort, "chaos_report", fmt.Sprintf("corrupting kernel failed: %v", err), Token, EncryptionKey)
		log.Fatalf("❌ error: %v", err)
	}
	library.SendMessage(MonitorIP, MonitorPort, "chaos_report", fmt.Sprintf("corrupted kernel file %s", file), Token, EncryptionKey)
	library.SendMessage(MonitorIP, MonitorPort, "variable", fmt.Sprintf("BrokenFiles,%s", file), Token, EncryptionKey)
}
