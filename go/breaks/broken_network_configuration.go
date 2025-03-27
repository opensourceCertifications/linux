package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
)

var possibleGrubPaths = []string{
	"/boot/grub2/grub.cfg",
	"/boot/grub/grub.cfg",
}

func BreakGrub() {
	_ = os.Remove("/etc/default/grub")
	for _, path := range possibleGrubPaths {
		if fileExists(path) {
			removeLinesContaining(path, "root=")
			fmt.Println("[chaos] grub.cfg sabotaged at:", path)
			break
		}
	}
	exec.Command("logger", "-p", "err", "Simulated kernel error: This is a test error for demonstration").Run()
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

func main() {
	BreakGrub()
}
