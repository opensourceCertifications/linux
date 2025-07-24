package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"golang.org/x/crypto/ssh"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <target_ip>\n", os.Args[0])
		os.Exit(1)
	}
	targetIP := os.Args[1]

	// Read private key
	key, err := ioutil.ReadFile("/home/vagrant/.ssh/id_ed25519")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading private key: %v\n", err)
		os.Exit(1)
	}

	// Parse key
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing private key: %v\n", err)
		os.Exit(1)
	}

	// SSH config
	config := &ssh.ClientConfig{
		User: "vagrant",
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // ⚠️ ok for testing only
	}

	// Run remote command
	err = runRemoteCommand(targetIP, config, "touch /tmp/hello")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✅ Successfully ran 'touch /tmp/hello' on", targetIP)
}

func runRemoteCommand(ip string, config *ssh.ClientConfig, command string) error {
	addr := ip + ":22"
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return fmt.Errorf("SSH dial error to %s: %v", addr, err)
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("Failed to create session: %v", err)
	}
	defer session.Close()

	if err := session.Run(command); err != nil {
		return fmt.Errorf("Command failed: %v", err)
	}

	return nil
}

