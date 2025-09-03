package main

import (
	"io/ioutil"
	"log"
	"os"
	"strings"
	"fmt"
	"strconv"
	"math/rand"
	"time"
	"github.com/opensourceCertifications/linux/shared/library"
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
	entries, err := ioutil.ReadDir("/boot")
	if err != nil {
		log.Fatalf("failed to read /boot: %v", err)
	}
	var vmlinuzFiles []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasPrefix(e.Name(), "vmlinuz-") {
			vmlinuzFiles = append(vmlinuzFiles, "/boot/"+e.Name())
		}
	}
	vmlinuzFiles = append(vmlinuzFiles, "/boot/efi/EFI/almalinux/grubx64.efi", "/boot/efi/EFI/almalinux/grub.cfg")

	rand.Seed(time.Now().UnixNano())

	library.SendMessage(MonitorIP, MonitorPort, "chaos_report", fmt.Sprintf("found vmlinuz files: %v", vmlinuzFiles), Token, EncryptionKey)
	file, err := library.CorruptFile(vmlinuzFiles[rand.Intn(len(vmlinuzFiles))], 100)
	if err != nil {
		library.SendMessage(MonitorIP, MonitorPort, "chaos_report", fmt.Sprintf("corrupting kernel failed: %v", err), Token, EncryptionKey)
		log.Fatalf("❌ error: %v", err)
	}
	library.SendMessage(MonitorIP, MonitorPort, "chaos_report", fmt.Sprintf("corrupted kernel file %s", file), Token, EncryptionKey)
}
