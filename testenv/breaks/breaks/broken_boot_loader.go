package breaks

import (
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
	"fmt"
	"time"
	"testenv/internal/comm"
	"testenv/internal/registry"
	"github.com/opensourceCertifications/linux/shared/types"
)

func BreakBootLoader() error {
	grubPath := findGrubCfg()
	comm.SendMessage("general", types.General{Message: "this is a test message from BreakBootLoader"})
	if grubPath == "" {
		report := types.ChaosReport{
			Timestamp: time.Now().Format(time.RFC3339),
			Action:    fmt.Sprintf("No grub.cfg found!"),
			Agent:     "test_environment",
		}
		comm.SendMessage("chaos_report", report)
		return logError("No grub.cfg found!")
	}

	log.Println("Sabotaging GRUB at:", grubPath)
	report := types.ChaosReport{
		Timestamp: time.Now().Format(time.RFC3339),
		Action:    fmt.Sprintf("Deleted /etc/default/grub"),
		Agent:     "test_environment",
	}
	comm.SendMessage("chaos_report", report)

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

// Auto-register on package init
func init() {
    registry.Register("BreakBootLoader", BreakBootLoader)
}
