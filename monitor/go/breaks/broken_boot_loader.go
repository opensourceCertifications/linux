package main

import (
	"log"
	"os"
	"fmt"
	"strconv"
	"math/rand"
//	"time"
	"github.com/opensourceCertifications/linux/shared/library"
	"path/filepath"
)


var (
	MonitorIP     string
	MonitorPortStr string
	MonitorPort   int
	Token         string
	EncryptionKey string
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
}


func main() {
	fmt.Println("port is", MonitorPort)
	patterns := []string{
	"/boot/vmlinuz-*",
	"/boot/initramfs-*.img",
	"/boot/grub2/grub.cfg",          // exact paths are fine: Glob returns it if it exists
	"/boot/loader/entries/*.conf",
	"/boot/grub2/grubenv",
	}

	vmlinuzFiles := make([]string, 0, 64)
	for _, pat := range patterns {
		if matches, _ := filepath.Glob(pat); len(matches) > 0 {
			vmlinuzFiles = append(vmlinuzFiles, matches...)
		}
	}

	//rand.Seed(time.Now().UnixNano())

	library.SendMessage(MonitorIP, MonitorPort, "chaos_report", fmt.Sprintf("found vmlinuz files: %v", vmlinuzFiles), Token, EncryptionKey)
	file := vmlinuzFiles[rand.Intn(len(vmlinuzFiles))]
	corrupted_file, err := library.CorruptFile(file, 100)
	if err != nil {
		library.SendMessage(MonitorIP, MonitorPort, "chaos_report", fmt.Sprintf("corrupting kernel failed: %v", err), Token, EncryptionKey)
		log.Fatalf("❌ error: %v", err)
	} else {
		library.SendMessage(MonitorIP, MonitorPort, "chaos_report", fmt.Sprintf("corrupted kernel file %s", corrupted_file), Token, EncryptionKey)
		library.SendMessage(MonitorIP, MonitorPort, "variable", fmt.Sprintf("corruptedBootFiles,%s", file), Token, EncryptionKey)
	}
}
