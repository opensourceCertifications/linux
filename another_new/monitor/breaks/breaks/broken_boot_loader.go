package main

import (
	"io/ioutil"
	"log"
	"os"
//	"os/exec"
	"strings"
	"fmt"
	"strconv"
	"math/rand"
	"time"
//	"golang.org/x/sys/unix"
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

// CorruptFile flips one random byte in the file at path.
// The file size stays the same, and its modtime is restored.
//func CorruptFile_disable(path string) error {
//	// Read the file
//	data, err := os.ReadFile(path)
//	if err != nil {
//		library.SendMessage(MonitorIP, MonitorPort, "chaos_report", fmt.Sprintf("failed to read file: %w", err), Token, EncryptionKey)
//		return fmt.Errorf("failed to read file: %w", err)
//	}
//	if len(data) == 0 {
//		library.SendMessage(MonitorIP, MonitorPort, "chaos_report", "file is empty, nothing to corrupt", Token, EncryptionKey)
//		return fmt.Errorf("file is empty, nothing to corrupt")
//	}
//
//	// Save current modtime for sneaky effect
//	info, err := os.Stat(path)
//	if err != nil {
//		library.SendMessage(MonitorIP, MonitorPort, "chaos_report", fmt.Sprintf("failed to read file: %w", err), Token, EncryptionKey)
//		return fmt.Errorf("failed to stat file: %w", err)
//	}
//	origModTime := info.ModTime()
//
//	// Pick a random byte
//	rand.Seed(time.Now().UnixNano())
//	i := rand.Intn(len(data))
//	orig := data[i]
//	// Flip some bits
//	data[i] = orig ^ 0xFF
//
//	//fmt.Printf("Corrupted byte %d: %x -> %x\n", i, orig, data[i])
//	library.SendMessage(MonitorIP, MonitorPort, "chaos_report", fmt.Sprintf("Corrupted byte %d: %x -> %x", i, orig, data[i]), Token, EncryptionKey)
//
//	// Write it back
//	if err := os.WriteFile(path, data, info.Mode()); err != nil {
//		library.SendMessage(MonitorIP, MonitorPort, "chaos_report", fmt.Sprintf("failed to write file: %w", err), Token, EncryptionKey)
//		return fmt.Errorf("failed to write file: %w", err)
//	}
//
//	// Restore original modtime
//	if err := os.Chtimes(path, origModTime, origModTime); err != nil {
//		library.SendMessage(MonitorIP, MonitorPort, "chaos_report", fmt.Sprintf("failed to restore times: %w", err), Token, EncryptionKey)
//		return fmt.Errorf("failed to restore times: %w", err)
//	}
//
//	return nil
//}
//
//func BreakBootLoader() error {
//	grubPath := findGrubCfg()
//	if err := library.SendMessage(MonitorIP, MonitorPort, "chaos_report", fmt.Sprint("found grub file %s", grubPath), Token, EncryptionKey); err != nil {
//		fmt.Printf("Error sending message: %v\n", err)
//		return err
//	}
//
//	if grubPath == "" {
//		if err := library.SendMessage(MonitorIP, MonitorPort, "chaos_report", "no grub.cfg found", Token, EncryptionKey); err != nil {
//			fmt.Printf("Error sending message: %v\n", err)
//			return err
//		}
//		return nill// logError("No grub.cfg found!")
//	}
//
//	if err := library.SendMessage(MonitorIP, MonitorPort, "chaos_report", fmt.Sprint("Deleted grub file %s", grubPath) , Token, EncryptionKey); err != nil {
//		fmt.Printf("Error sending message: %v\n", err)
//		return err
//	}
//
//	var uts unix.Utsname
//	if err := unix.Uname(&uts); err != nil {
//		panic(err)
//	}
//
//	// uts.Release is a fixed-size [65]int8 array, so convert to string
//	release := make([]byte, len(uts.Release))
//	for i, v := range uts.Release {
//		if v == 0 {
//			release = release[:i]
//			break
//		}
//		release[i] = byte(v)
//	}
//
//	library.SendMessage(MonitorIP, MonitorPort, "chaos_report", fmt.Sprintf("corrupting kernel version %s", release), Token, EncryptionKey)
//	entries, err := ioutil.ReadDir("/boot")
//	if err != nil {
//		log.Fatalf("failed to read /boot: %v", err)
//	}
//	var vmlinuzFiles []string
//	for _, e := range entries {
//		if !e.IsDir() && strings.HasPrefix(e.Name(), "vmlinuz-") {
//			vmlinuzFiles = append(vmlinuzFiles, "/boot/"+e.Name())
//		}
//	}
//	vmlinuzFiles = append(vmlinuzFiles, "/boot/efi/EFI/grubx64.efi", "/boot/efi/EFI/grub.cfg")
//
//	rand.Seed(time.Now().UnixNano())
//
//	//if err := CorruptFile("/boot/vmlinuz-" + string(release)); err != nil {
//	//if err := CorruptFile(vmlinuzFiles[rand.Intn(len(vmlinuzFiles))]); err != nil {
//	file := CorruptFile(vmlinuzFiles[rand.Intn(len(vmlinuzFiles))]); {
//	library.SendMessage(MonitorIP, MonitorPort, "chaos_report", fmt.Sprintf("corrupted kernel file %s", file), Token, EncryptionKey)
//
//	if err := exec.Command("logger", "-p", "err", "Simulated kernel error: This is a test error for demonstration").Run(); err != nil {
//		log.Printf("Failed to inject fake kernel error: %v", err)
//	}
//
//	library.SendMessage(MonitorIP, MonitorPort, "operation_complete", "all done", Token, EncryptionKey)
//	log.Println("Bootloader sabotage complete.")
//	return nil
//}

//func findGrubCfg() string {
//	paths := []string{"/boot/grub2/grub.cfg", "/boot/grub/grub.cfg"}
//	for _, path := range paths {
//		if _, err := os.Stat(path); err == nil {
//			return path
//		}
//	}
//	return ""
//}

//func removeRootLines(filepath string) error {
//	data, err := ioutil.ReadFile(filepath)
//	if err != nil {
//		return err
//	}
//	lines := strings.Split(string(data), "\n")
//	var newLines []string
//	for _, line := range lines {
//		if !strings.Contains(line, "root=") {
//			newLines = append(newLines, line)
//		}
//	}
//	return ioutil.WriteFile(filepath, []byte(strings.Join(newLines, "\n")), 0644)
//}

//func logError(msg string) error {
//	log.Println(msg)
//	return fmt.Errorf(msg)
//}

// CorruptFile flips all bits in each byte, then adds a random offset [0–255] to each byte.
// The file size stays the same, and its modtime is restored.
func CorruptFile(path string) (string, error) {
	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}
	if len(data) == 0 {
		return "", fmt.Errorf("file is empty, nothing to corrupt")
	}

	// Save original modtime
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("failed to stat file: %w", err)
	}
	origModTime := info.ModTime()

	// Pick a random offset (0–255)
	rand.Seed(time.Now().UnixNano())
	offset := byte(rand.Intn(256))
	//fmt.Printf("⚡ Using random offset: %d\n", offset)
	library.SendMessage(MonitorIP, MonitorPort, "chaos_report", fmt.Sprintf("Using random offset: %d", offset), Token, EncryptionKey)

	// Flip + add offset for every byte
	for i, b := range data {
		// flip bits, then add offset with wraparound
		data[i] = (b ^ 0xFF) + offset
	}

	// Write back
	if err := os.WriteFile(path, data, info.Mode()); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	// Restore original modtime
	if err := os.Chtimes(path, origModTime, origModTime); err != nil {
		return "", fmt.Errorf("failed to restore times: %w", err)
	}

	//fmt.Printf("✅ File %s corrupted (flip+offset)\n", path)
	library.SendMessage(MonitorIP, MonitorPort, "chaos_report", fmt.Sprintf("File %s corrupted (flip+offset)", path), Token, EncryptionKey)
	return fmt.Sprintf("File %s corrupted (flip+offset)", path), nil
}


func main() {
	fmt.Println("port is", MonitorPort)
	//p, err := strconv.Atoi(MonitorPortStr)
	//if err != nil {
	//	log.Fatalf("invalid MonitorPortStr %q: %v", MonitorPortStr, err)
	//}
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

	//if err := CorruptFile("/boot/vmlinuz-" + string(release)); err != nil {
	//if err := CorruptFile(vmlinuzFiles[rand.Intn(len(vmlinuzFiles))]); err != nil {
	//file := CorruptFile(vmlinuzFiles[rand.Intn(len(vmlinuzFiles))]);
	//msg, err := CorruptFile("/tmp/foo")
	//file, err := CorruptFile(vmlinuzFiles[rand.Intn(len(vmlinuzFiles))])
	library.SendMessage(MonitorIP, MonitorPort, "chaos_report", fmt.Sprintf("found vmlinuz files: %v", vmlinuzFiles), Token, EncryptionKey)
	file, err := library.CorruptFile(vmlinuzFiles[rand.Intn(len(vmlinuzFiles))], 100)
	if err != nil {
		library.SendMessage(MonitorIP, MonitorPort, "chaos_report", fmt.Sprintf("corrupting kernel failed: %v", err), Token, EncryptionKey)
		log.Fatalf("❌ error: %v", err)
	}
	library.SendMessage(MonitorIP, MonitorPort, "chaos_report", fmt.Sprintf("corrupted kernel file %s", file), Token, EncryptionKey)
	//MonitorPort = p
	//if err := BreakBootLoader(); err != nil {
	//	log.Fatalf("❌ BreakBootLoader failed: %v", err)
	//}
}
