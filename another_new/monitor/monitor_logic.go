package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"bufio"
	"io"
	"net"
	"golang.org/x/crypto/ssh"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"path/filepath"
	"time"
	mathrand "math/rand"
	"strings"
	"os/exec"
	"log"

	"github.com/opensourceCertifications/linux/shared/types"
)

func main() {
	// Get IP from environment variable
	targetIP := os.Getenv("TESTENV_ADDRESS")
	if targetIP == "" {
		fmt.Fprintln(os.Stderr, "Error: TESTENV_ADDRESS environment variable not set")
		os.Exit(1)
	}

	// Read private key
	keyPath := filepath.Join(os.Getenv("HOME"), ".ssh", "id_ed25519")
	key, err := ioutil.ReadFile(keyPath)
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
	scriptPath, err := pickRandomFile("breaks/test")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to pick test file: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("🎯 Selected test script: %s\n", scriptPath)

	err = runRemoteCommandWithListener(targetIP, config, scriptPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✅ Successfully ran 'touch /tmp/hello' on", targetIP)
}

func generateToken(nBytes int) (string, error) {
	b := make([]byte, nBytes)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func pickRandomFile(dir string) (string, error) {
	files, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		return "", err
	}
	if len(files) == 0 {
		return "", fmt.Errorf("no .go files found in %s", dir)
	}
	mathrand.Seed(time.Now().UnixNano())
	return files[mathrand.Intn(len(files))], nil
}

func compileChaosBinary(sourcePath, monitorIP string, port int, token string) (string, error) {
	outputPath := filepath.Join("/tmp", "break_tool")

	ldflags := fmt.Sprintf("-X 'main.MonitorIP=%s' -X 'main.MonitorPort=%d' -X 'main.Token=%s'", monitorIP, port, token)

	cmd := exec.Command("go", "build", "-o", outputPath, "-ldflags", ldflags, sourcePath)
	cmd.Env = append(os.Environ(), "GOOS=linux", "GOARCH=amd64")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Printf("🛠️ Compiling %s → %s...\n", sourcePath, outputPath)
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("failed to compile chaos binary: %v", err)
	}

	fmt.Printf("✅ Compiled binary written to: %s\n", outputPath)
	return outputPath, nil
}

func runRemoteCommandWithListener(ip string, config *ssh.ClientConfig, scriptPath string) error {
	// Step 1: Find random open port
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return fmt.Errorf("Failed to open listener: %v", err)
	}
	defer listener.Close()

	// Extract the actual port chosen
	addr := listener.Addr().(*net.TCPAddr)
	port := addr.Port
	token, err := generateToken(16) // 16 bytes = 32 hex chars
	fmt.Printf("🔑 Generated token: %s\n", token)
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
		handleChaosConnection(conn, token)
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

	compiledPath, err := compileChaosBinary(scriptPath, ip, port, token)
	if err != nil {
		log.Fatalf("Compilation failed: %v", err)
	}
	fmt.Printf("✅ Compiled binary: %s\n", compiledPath)

	// Step 2.5: Copy chaos script to testenv using SFTP-over-SSH
	remotePath := "/tmp/" + filepath.Base(compiledPath)
	content, err := os.ReadFile(compiledPath)
	if err != nil {
		return fmt.Errorf("failed to read local file for scp: %v", err)
	}

	sess, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create SCP session: %v", err)
	}
	defer sess.Close()

	scpCmd := fmt.Sprintf("scp -t /tmp")
	sess.Stdin = strings.NewReader(fmt.Sprintf("C0644 %d %s\n%s\x00", len(content), filepath.Base(compiledPath), content))

	if err := sess.Run(scpCmd); err != nil {
		return fmt.Errorf("scp to remote failed: %v", err)
	}

	fmt.Printf("📦 Copied %s to %s on testenv\n", compiledPath, remotePath)

	check := fmt.Sprintf("for i in {1..5}; do nc -z %s %d && break || sleep 2; done", ip, port)
	run := fmt.Sprintf("chmod +x %s && %s", compiledPath, compiledPath)
	fullCmd := fmt.Sprintf("%s && %s", check, run)

	fmt.Printf("🚀 Running remote command on %s...\n", ip)
	if err := session.Run(fullCmd); err != nil {
		return fmt.Errorf("Command failed: %v", err)
	}
	/*
	fmt.Printf("🚀 Running remote command on %s...\n", ip)
	if err := session.Run(compiledPath); err != nil {
		return fmt.Errorf("Command failed: %v", err)
	}
*/
	// Step 3: Wait for listener to finish
	<-done
	return nil
}

func handleChaosConnection(conn net.Conn, expectedToken string) {
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
		fmt.Println("🔹 Raw input:", line)

		var msg types.ChaosMessage
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			fmt.Fprintf(os.Stderr, "⚠️ Invalid JSON: %s\n", line)
			continue
		}

		// Token check
		if msg.Token == expectedToken {
			fmt.Println("🔐 Token check: ✅ valid")
		} else {
			fmt.Println("🔐 Token check: ❌ invalid")
		}

		fmt.Printf("📨 Status: %-20s  Message: %s\n", msg.Status, msg.Message)

		if msg.Status == "operation_complete" {
			fmt.Println("✅ Operation completed, closing connection.")
			break
		}
	}
}

