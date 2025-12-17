package main

import (
	"fmt"
	"errors"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
	"path/filepath"
	"math/big"
	"crypto/rand"
	"golang.org/x/crypto/ssh"
	"chaos-agent/library"
	"chaos-agent/types"
	"chaos-agent/breaks"
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

func pickRandomFile(dir string) (string, error) {
	files, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		return "", err
	}
	if len(files) == 0 {
		return "", fmt.Errorf("no .go files found in %s", dir)
	}
	///mathrand.Seed(time.Now().UnixNano())
	n := big.NewInt(int64(len(files)))
	r, err := rand.Int(rand.Reader, n)
	if err != nil {
		return "", fmt.Errorf("failed to generate random index: %w", err)
	}
	return files[r.Int64()], nil

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


	p, err := pickRandomFile("breaks")
	if err != nil {
		fmt.Printf("Error picking random test file: %v\n", err)
	}
	fmt.Printf("Picked random test file: %s\n", p)

	// p == "breaks/FileCorruptor.go"
	base := filepath.Base(p)              // "FileCorruptor.go"
	name := strings.TrimSuffix(base, ".go") // "FileCorruptor"

	fn, ok := breaks.Get(name)
	if !ok {
		fmt.Printf("No registered break for %q\n", name)
		return
	}

	if err := fn(); err != nil {
		fmt.Printf("Break %q failed: %v\n", name, err)
	} else {
		fmt.Printf("Break %q completed successfully\n", name)
	}

	out, err := library.RunCommandOverWarmSSH("touch /vagrant/works")
	if err != nil {
		fmt.Printf("SSH command error: %v\nOutput:\n%s\n", err, out)
		return
	}
	fmt.Printf("SSH command succeeded. Output:\n%s\n", out)
}
