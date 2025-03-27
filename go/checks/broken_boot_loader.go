package main

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func CheckGrubFix() bool {
	// Step 1: Discover grub.cfg under /boot matching grub*
	entries, err := os.ReadDir("/boot")
	if err != nil {
		log.Println("[check] Failed to read /boot directory")
		return false
	}

	var grubDir string
	for _, e := range entries {
		if e.IsDir() && strings.HasPrefix(e.Name(), "grub") {
			grubDir = filepath.Join("/boot", e.Name())
			break
		}
	}

	if grubDir == "" {
		log.Println("[check] No GRUB directory found → FAIL")
		return false
	}

	// Step 2: Find grub.cfg in the discovered grub directory
	var grubCfgPath string
	err = filepath.Walk(grubDir, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() && info.Name() == "grub.cfg" {
			grubCfgPath = path
			return filepath.SkipDir
		}
		return nil
	})

	if grubCfgPath == "" {
		log.Println("[check] grub.cfg not found → FAIL")
		return false
	}
	log.Println("[check] Found GRUB config at:", grubCfgPath)

	// Step 3: Check /etc/default/grub exists
	if _, err := os.Stat("/etc/default/grub"); os.IsNotExist(err) {
		log.Println("[check] /etc/default/grub missing → FAIL")
		return false
	}

	// Step 4: Check grub.cfg exists
	if _, err := os.Stat(grubCfgPath); os.IsNotExist(err) {
		log.Println("[check] grub.cfg file missing → FAIL")
		return false
	}

	// Step 5: Check grub.cfg contains 'root='
	contents, err := os.ReadFile(grubCfgPath)
	if err != nil || !strings.Contains(string(contents), "root=") {
		log.Println("[check] grub.cfg missing root= line → FAIL")
		return false
	}

	// Step 6: Check system status
	statusOut, err := exec.Command("systemctl", "is-system-running").Output()
	if err != nil || strings.TrimSpace(string(statusOut)) != "running" {
		log.Println("[check] System not in running state → FAIL")
		return false
	}

	// Step 7: Check journalctl for boot errors
	errLog, err := exec.Command("journalctl", "-b", "-p", "err").Output()
	if err == nil && len(errLog) > 0 {
		log.Println("[check] Kernel boot errors detected → FAIL")
		return false
	}

	log.Println("[check] All GRUB integrity checks passed → PASS")
	return true
}

func main() {
	CheckGrubFix()
}
