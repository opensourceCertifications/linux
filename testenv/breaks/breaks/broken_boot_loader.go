package breaks

import (
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
	"fmt"
)

func BreakBootLoader() error {
	grubPath := findGrubCfg()
	if grubPath == "" {
		return logError("No grub.cfg found!")
	}

	log.Println("Sabotaging GRUB at:", grubPath)

	if err := os.Remove("/etc/default/grub"); err != nil && !os.IsNotExist(err) {
		log.Printf("Failed to delete /etc/default/grub: %v", err)
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
