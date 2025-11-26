package main

import (
	"fmt"
	"errors"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
	"golang.org/x/crypto/ssh"
	"chaos-agent/library"
	"chaos-agent/types"
)

func ProbeFromEnv() (types.Probe, error) {
	ip := strings.TrimSpace(os.Getenv("TESTENV_ADDRESS"))
	if ip == "" {
		return types.Probe{}, errors.New("TESTENV not set (expected IP of test environment)")
	}
	if net.ParseIP(ip) == nil {
		return types.Probe{}, errors.New("TESTENV must be a valid IP address")
	}

	// Port (default 22)
	port := 22

	// User (default root)
	user := "root"

	// Timeout (default 5s)
	timeout := 5 * time.Second

	return types.Probe{
		Addr:    net.JoinHostPort(ip, strconv.Itoa(port)),
		User:    user,
		Timeout: timeout,
	}, nil
}

func main() {
	// Call into the library; returns a JSON string.
	//audit_scope := library.GetAuditScope()
	//fmt.Println(audit_scope)
	probe, err := ProbeFromEnv()
	if err != nil {
		fmt.Printf("Error creating probe: %v\n", err)
		return
	}
	fmt.Printf("Probe created: %+v\n", probe)

	// Build SSH auth from your private key, e.g. ~/.ssh/id_rsa
	keyPath := os.Getenv("HOME") + "/.ssh/id_ed25519"
	key, err := os.ReadFile(keyPath)
	if err != nil {
		fmt.Printf("Error reading private key %s: %v\n", keyPath, err)
		return
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		fmt.Printf("Error parsing private key: %v\n", err)
		return
	}

	authMethods := []ssh.AuthMethod{
		ssh.PublicKeys(signer),
	}

	// For now, don't verify host key strictly (OK for your controlled env).
	// Later you can replace this with a proper callback using known_hosts.
	hostKeyCallback := ssh.InsecureIgnoreHostKey()

	if err := library.SetupWarmSSH(probe, authMethods, hostKeyCallback); err != nil {
		fmt.Printf("Error setting up warm SSH: %v\n", err)
		return
	}


	out, err := library.RunCommandOverWarmSSH("touch /vagrant/works")
	if err != nil {
		fmt.Printf("SSH command error: %v\nOutput:\n%s\n", err, out)
		return
	}
	fmt.Printf("SSH command succeeded. Output:\n%s\n", out)
}
