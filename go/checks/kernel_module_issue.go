package main

import (
	"log"
	"os/exec"
	"strings"
)

func CheckKernelModuleFix() bool {
	// Check if e1000 module is loaded
	modOut, err := exec.Command("lsmod").Output()
	if err != nil || strings.Contains(string(modOut), "e1000") {
		log.Println("[check] Kernel module e1000 still loaded → FAIL")
		return false
	}

	// Check if it's blacklisted
	blacklistOut, err := exec.Command("sh", "-c", "grep 'blacklist e1000' /etc/modprobe.d/*.conf").CombinedOutput()
	if err != nil || len(blacklistOut) == 0 {
		log.Println("[check] Kernel module e1000 not blacklisted → FAIL")
		return false
	}

	// Check dmesg logs for module-related issues
	dmesgOut, err := exec.Command("dmesg").CombinedOutput()
	if err != nil || !strings.Contains(strings.ToLower(string(dmesgOut)), "module") {
		log.Println("[check] No module-related entries in dmesg → FAIL")
		return false
	}

	log.Println("[check] Kernel module checks passed → PASS")
	return true
}

func main() {
	CheckKernelModuleFix()
}