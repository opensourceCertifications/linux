package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

var possibleGrubPaths = []string{
	"/boot/grub2/grub.cfg",
	"/boot/grub/grub.cfg",
}

func CheckGrubFix() bool {
	if _, err := os.Stat("/etc/default/grub"); os.IsNotExist(err) {
		log.Println("[check] /etc/default/grub is missing → FAIL")
		return false
	}
	found := false
	for _, path := range possibleGrubPaths {
		if fileExists(path) {
			contents, err := os.ReadFile(path)
			if err == nil && strings.Contains(string(contents), "root=") {
				found = true
				break
			}
		}
	}
	if !found {
		log.Println("[check] grub.cfg missing or lacks root= → FAIL")
		return false
	}
	statusOut, err := exec.Command("systemctl", "is-system-running").Output()
	if err != nil || strings.TrimSpace(string(statusOut)) != "running" {
		log.Println("[check] System not in running state → FAIL")
		return false
	}
	errLog, err := exec.Command("journalctl", "-b", "-p", "err").Output()
	if err == nil && len(errLog) > 0 {
		log.Println("[check] Kernel boot errors detected → FAIL")
		return false
	}
	return true
}

func BreakNetwork() {
	exec.Command("sh", "-c", "ip route del default || true").Run()
	exec.Command("sh", "-c", "iptables -I OUTPUT -p icmp --dst 8.8.8.8 -j DROP || true").Run()
	exec.Command("sh", "-c", "iptables -I OUTPUT -p udp --dport 53 -j DROP || true").Run()
	exec.Command("sh", "-c", "iptables -I OUTPUT -p tcp --dport 53 -j DROP || true").Run()

	unit := `[Unit]
Description=Persistently sabotage network (firewall rules)
After=network-online.target
Wants=network-online.target

[Service]
Type=oneshot
ExecStart=/usr/sbin/ip route del default
ExecStart=/usr/sbin/iptables -I OUTPUT -p icmp --dst 8.8.8.8 -j DROP
ExecStart=/usr/sbin/iptables -I OUTPUT -p udp --dport 53 -j DROP
ExecStart=/usr/sbin/iptables -I OUTPUT -p tcp --dport 53 -j DROP
RemainAfterExit=true

[Install]
WantedBy=multi-user.target`

	err := os.WriteFile("/etc/systemd/system/sabotage-firewall.service", []byte(unit), 0644)
	if err != nil {
		log.Println("[chaos] failed to write firewall sabotage unit:", err)
	}

	exec.Command("systemctl", "enable", "sabotage-firewall.service").Run()
	exec.Command("sh", "-c", "mv /usr/sbin/iptables /usr/sbin/iptables_bak || true").Run()
}

func CheckNetworkFix() bool {
	routeOut, err := exec.Command("sh", "-c", "ip route show | grep default").Output()
	if err != nil || len(routeOut) == 0 {
		log.Println("[check] No default gateway found → FAIL")
		return false
	}
	pingOut, err := exec.Command("ping", "-c", "1", "8.8.8.8").CombinedOutput()
	if err != nil {
		log.Println("[check] Ping to 8.8.8.8 failed → FAIL")
		return false
	}
	dnsOut, err := exec.Command("ping", "-c", "1", "google.com").CombinedOutput()
	if err != nil {
		log.Println("[check] DNS resolution failed → FAIL")
		return false
	}
	return true
}

func BreakExecutables() {
	dirs := []string{"/usr/local/sbin", "/usr/local/bin", "/usr/sbin", "/usr/bin", "/sbin", "/bin", "/usr/local/games"}
	excludes := []string{"sh", "ssh", "sshd", "sudo", "rsync"}
	var executables []string

	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.Type().IsRegular() && !contains(excludes, entry.Name()) {
				executables = append(executables, filepath.Join(dir, entry.Name()))
			}
		}
	}

	rand.Seed(time.Now().UnixNano())
	n := rand.Intn(11) + 20 // 20–30
	rand.Shuffle(len(executables), func(i, j int) {
		executables[i], executables[j] = executables[j], executables[i]
	})
	chosen := executables[:min(n, len(executables))]

	_ = os.MkdirAll("/tmp/corrupted", 0755)

	for _, path := range chosen {
		dest := filepath.Join("/tmp/corrupted", filepath.Base(path))
		exec.Command("cp", path, dest).Run()
		exec.Command("dd", "if=/dev/urandom", fmt.Sprintf("of=%s", dest), "bs=1", "count=1024", "conv=notrunc").Run()
		exec.Command("cp", dest, path).Run()
	}
}

func CheckExecutablesFix() bool {
	dirs := []string{"/usr/local/sbin", "/usr/local/bin", "/usr/sbin", "/usr/bin", "/sbin", "/bin", "/usr/local/games"}
	excludes := []string{"sh", "ssh", "sshd", "sudo", "rsync"}

	// Look for obviously corrupted binaries
	// (This check is a stub — could be improved with hash compare, etc.)
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			name := entry.Name()
			if entry.Type().IsRegular() && !contains(excludes, name) {
				path := filepath.Join(dir, name)
				out, err := exec.Command("file", path).CombinedOutput()
				if err != nil || !strings.Contains(string(out), "ELF") {
					log.Printf("[check] Executable %s appears corrupted → FAIL\n", path)
					return false
				}
			}
		}
	}
	return true
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func removeLinesContaining(path, substr string) {
	contents, err := os.ReadFile(path)
	if err != nil {
		log.Println("[chaos] failed to read", path)
		return
	}
	lines := strings.Split(string(contents), "\n")
	var filtered []string
	for _, line := range lines {
		if !strings.Contains(line, substr) {
			filtered = append(filtered, line)
		}
	}
	err = os.WriteFile(path, []byte(strings.Join(filtered, "\n")), 0644)
	if err != nil {
		log.Println("[chaos] failed to write", path)
	}
}

func contains(slice []string, val string) bool {
	for _, s := range slice {
		if s == val {
			return true
		}
	}
	return false
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func main() {
	CheckGrubFix()
}
