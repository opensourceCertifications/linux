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
		log.Fatal("No grub.cfg or BLS entries found!")
	} else if strings.HasPrefix(grubPath, "/boot/loader/entries/") {
		log.Println("✅ System uses BLS entries instead of grub.cfg.")
	} else {
		log.Println("Found grub.cfg at:", grubPath)
	}

	// Check if /etc/default/grub exists
	if _, err := os.Stat("/etc/default/grub"); os.IsNotExist(err) {
		log.Println("❌ /etc/default/grub does not exist!")
	} else {
		log.Println("✅ /etc/default/grub exists.")
	}

	// Check grub.cfg file itself
	if _, err := os.Stat(grubPath); os.IsNotExist(err) {
		log.Println("❌ grub.cfg does not exist!")
	} else {
		log.Println("✅ grub.cfg exists.")
	}

	// Check for 'root=' lines
	rootFound, err := hasRootLines(grubPath)
	if err != nil {
		log.Fatalf("Error reading grub.cfg: %v", err)
	}
	if rootFound {
		log.Println("✅ grub.cfg contains root= entries.")
	} else {
		log.Println("❌ grub.cfg missing root= entries!")
	}

	// Check system running status
	cmd := exec.Command("systemctl", "is-system-running")
	output, err := cmd.Output()
	if err != nil {
		log.Printf("Failed to check systemctl status: %v", err)
	} else if strings.TrimSpace(string(output)) == "running" {
		log.Println("✅ System is running normally.")
	} else {
		log.Println("❌ System not in running state:", string(output))
	}
}

func findGrubCfg() string {
	possiblePaths := []string{
		"/boot/grub2/grub.cfg",
		"/boot/grub/grub.cfg",
		"/boot/efi/EFI/centos/grub.cfg",
		"/boot/efi/EFI/almalinux/grub.cfg",
	}

	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	// Fallback for BLS
	files, err := os.ReadDir("/boot/loader/entries/")
	if err == nil && len(files) > 0 {
		log.Println("✅ Found BLS entry files in /boot/loader/entries/")
		return "/boot/loader/entries/" // special case
	}

	return ""
}


func hasRootLines(filepath string) (bool, error) {
	data, err := ioutil.ReadFile(filepath)
	if err != nil {
		return false, err
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.Contains(line, "root=") {
			return true, nil
		}
	}
	return false, nil
}
