package main

import (
	"bufio"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func parseClassicGrub(grubCfgPath string) bool {
	contents, err := os.ReadFile(grubCfgPath)
	if err != nil {
		log.Println("[check] Failed to read grub.cfg → FAIL")
		return false
	}
	contentsStr := string(contents)

	if !(strings.Contains(contentsStr, "linux") || strings.Contains(contentsStr, "linux16")) ||
		!(strings.Contains(contentsStr, "initrd") || strings.Contains(contentsStr, "initrd16")) {
		log.Println("[check] grub.cfg missing linux/initrd entries → FAIL")
		return false
	}

	scanner := bufio.NewScanner(strings.NewReader(contentsStr))
	pathsToCheck := []string{}
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "linux") || strings.HasPrefix(line, "initrd") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				pathsToCheck = append(pathsToCheck, parts[1])
			}
		}
	}

	for _, path := range pathsToCheck {
		fullPath := filepath.Join("/boot", strings.TrimPrefix(path, "/"))
		info, err := os.Stat(fullPath)
		if err != nil || info.Size() < 1024 {
			log.Println("[check] Missing or empty kernel/initrd file:", fullPath, "→ FAIL")
			return false
		}
	}

	log.Println("[check] Classic GRUB looks good → PASS")
	return true
}

func parseBLS() bool {
	entries, err := filepath.Glob("/boot/loader/entries/*.conf")
	if err != nil || len(entries) == 0 {
		log.Println("[check] No BLS entries found → FAIL")
		return false
	}

	for _, entry := range entries {
		content, err := os.ReadFile(entry)
		if err != nil {
			log.Println("[check] Failed to read BLS entry:", entry)
			return false
		}
		lines := strings.Split(string(content), "\n")
		foundLinux := false
		foundInitrd := false
		for _, line := range lines {
			if strings.HasPrefix(line, "linux") || strings.HasPrefix(line, "initrd") {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					fullPath := filepath.Join("/boot", strings.TrimPrefix(fields[1], "/"))
					info, err := os.Stat(fullPath)
					if err != nil || info.Size() < 1024 {
						log.Println("[check] Missing or empty file:", fullPath, "→ FAIL")
						return false
					}
				}
				if strings.HasPrefix(line, "linux") {
					foundLinux = true
				}
				if strings.HasPrefix(line, "initrd") {
					foundInitrd = true
				}
			}
		}
		if !foundLinux || !foundInitrd {
			log.Println("[check] BLS entry missing linux/initrd in:", entry, "→ FAIL")
			return false
		}
	}

	log.Println("[check] BLS entries look good → PASS")
	return true
}

func CheckGrubSafeToReboot() bool {
	// 1. Check /etc/default/grub exists
	if !fileExists("/etc/default/grub") {
		log.Println("[check] /etc/default/grub is missing → FAIL")
		return false
	}

	// 2. Check if grub.cfg exists
	grubCfgPath := "/boot/grub2/grub.cfg"
	if !fileExists(grubCfgPath) {
		log.Println("[check] grub.cfg not found at", grubCfgPath, "→ FAIL")
		return false
	}

	contents, _ := os.ReadFile(grubCfgPath)
	if strings.Contains(string(contents), "blscfg") {
		log.Println("[check] Detected BLS mode")
		if !parseBLS() {
			return false
		}
	} else {
		log.Println("[check] Detected Classic GRUB mode")
		if !parseClassicGrub(grubCfgPath) {
			return false
		}
	}

	// 3. Attempt to regenerate GRUB config
	cmd := exec.Command("grub2-mkconfig", "-o", "/tmp/test-grub.cfg")
	err := cmd.Run()
	if err != nil {
		log.Println("[check] grub2-mkconfig failed → FAIL")
		return false
	}
	_ = os.Remove("/tmp/test-grub.cfg")

	log.Println("[check] GRUB appears safe to reboot → PASS")
	return true
}

func main() {
	CheckGrubSafeToReboot()
}
