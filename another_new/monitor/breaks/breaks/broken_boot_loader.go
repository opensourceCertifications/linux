package main

import (
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
	"fmt"
	"strconv"
	"github.com/opensourceCertifications/linux/shared/library"
)


var (
	MonitorIP     string
	MonitorPortStr string
	MonitorPort   int
	Token         string
	EncryptionKey string
)

func BreakBootLoader() error {
	grubPath := findGrubCfg()
	if err := library.SendMessage(MonitorIP, MonitorPort, "log", fmt.Sprint("found grub file %s", grubPath), Token, EncryptionKey); err != nil {
		fmt.Printf("Error sending message: %v\n", err)
		return err
	}

	if grubPath == "" {
		if err := library.SendMessage(MonitorIP, MonitorPort, "chaos_report", "no grub.cfg found", Token, EncryptionKey); err != nil {
			fmt.Printf("Error sending message: %v\n", err)
			return err
		}
		return logError("No grub.cfg found!")
	}

	log.Println("Sabotaging GRUB at:", grubPath)
	if err := library.SendMessage(MonitorIP, MonitorPort, "chaos_report", fmt.Sprint("Deleted grub file %s", grubPath) , Token, EncryptionKey); err != nil {
		fmt.Printf("Error sending message: %v\n", err)
		return err
	}

	if err := removeRootLines(grubPath); err != nil {
		return err
	}

	if err := exec.Command("logger", "-p", "err", "Simulated kernel error: This is a test error for demonstration").Run(); err != nil {
		log.Printf("Failed to inject fake kernel error: %v", err)
	}

	log.Println("Bootloader sabotage complete.")
	return nil
}

func findGrubCfg() string {
	paths := []string{"/boot/grub2/grub.cfg", "/boot/grub/grub.cfg"}
	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}

func removeRootLines(filepath string) error {
	data, err := ioutil.ReadFile(filepath)
	if err != nil {
		return err
	}
	lines := strings.Split(string(data), "\n")
	var newLines []string
	for _, line := range lines {
		if !strings.Contains(line, "root=") {
			newLines = append(newLines, line)
		}
	}
	return ioutil.WriteFile(filepath, []byte(strings.Join(newLines, "\n")), 0644)
}

func logError(msg string) error {
	log.Println(msg)
	return fmt.Errorf(msg)
}

func main() {
	fmt.Println("port is", MonitorPort)
	p, err := strconv.Atoi(MonitorPortStr)
	if err != nil {
		log.Fatalf("invalid MonitorPortStr %q: %v", MonitorPortStr, err)
	}
	MonitorPort = p
	if err := BreakBootLoader(); err != nil {
		log.Fatalf("❌ BreakBootLoader failed: %v", err)
	}
}
