package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"bufio"
	"io"
	"net"

	"golang.org/x/crypto/ssh"
)

func main() {
	// Get IP from environment variable
	targetIP := os.Getenv("TESTENV_ADDRESS")
	if targetIP == "" {
		fmt.Fprintln(os.Stderr, "Error: TESTENV_ADDRESS environment variable not set")
		os.Exit(1)
	}

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
	err = runRemoteCommandWithListener(targetIP, config, "touch /tmp/hello")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✅ Successfully ran 'touch /tmp/hello' on", targetIP)
}

type ChaosMessage struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

func runRemoteCommandWithListener(ip string, config *ssh.ClientConfig, command string) error {
	// Step 1: Find random open port
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return fmt.Errorf("Failed to open listener: %v", err)
	}
	defer listener.Close()

	// Extract the actual port chosen
	addr := listener.Addr().(*net.TCPAddr)
	port := addr.Port
	fmt.Printf("📡 Listening on port %d\n", port)

	// Start listener goroutine
	done := make(chan struct{})
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to accept connection: %v\n", err)
			close(done)
			return
		}
		defer conn.Close()

		reader := bufio.NewReader(conn)
		for {
			line, err := reader.ReadString('\n')
			if err == io.EOF {
				break
			}
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error reading from connection: %v\n", err)
				break
			}

			line = line[:len(line)-1] // Strip newline
			fmt.Println("🔹", line)

			if line == "operation_complete" {
				fmt.Println("✅ Operation completed, closing connection.")
				break
			}
		}
		close(done)
	}()

	// Step 2: SSH and run the command
	addrStr := ip + ":22"
	client, err := ssh.Dial("tcp", addrStr, config)
	if err != nil {
		return fmt.Errorf("SSH dial error to %s: %v", addrStr, err)
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("Failed to create session: %v", err)
	}
	defer session.Close()

	fmt.Printf("🚀 Running remote command on %s...\n", ip)
	if err := session.Run(command); err != nil {
		return fmt.Errorf("Command failed: %v", err)
	}

	// Step 3: Wait for listener to finish
	<-done
	return nil
}

/*
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
*/
