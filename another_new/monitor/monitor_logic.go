package main

import (
	"bufio"
	"crypto/aes"
	"crypto/cipher" // Import cipher package for AES-GCM
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	mathrand "math/rand"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"
	"errors"

	"golang.org/x/crypto/ssh"

	"github.com/opensourceCertifications/linux/shared/types"
)

func main() {
    // Get IP from environment variable
    targetIP := os.Getenv("TESTENV_ADDRESS")
    if targetIP == "" {
        fmt.Fprintln(os.Stderr, "Error: TESTENV_ADDRESS environment variable not set")
        os.Exit(1) // This line is okay for startup, but subsequent errors won't exit the service
    }

    // Read private key (handles SSH connection)
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

    // SSH config (for remote interaction)
    config := &ssh.ClientConfig{
        User: "vagrant",
        Auth: []ssh.AuthMethod{
            ssh.PublicKeys(signer),
        },
        HostKeyCallback: ssh.InsecureIgnoreHostKey(), // ⚠️ ok for testing only
    }

    // Run remote command (this will pick a test file from "breaks/breaks")
    scriptPath, err := pickRandomFile("breaks/breaks")
    if err != nil {
        fmt.Fprintf(os.Stderr, "Failed to pick test file: %v\n", err)
        os.Exit(1)
    }
    fmt.Printf("🎯 Selected test script: %s\n", scriptPath)

    // Start the listener and wait for connections
    err = runRemoteCommandWithListener(targetIP, config, scriptPath)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        os.Exit(1)  // This will exit only on critical failure to start service
    }

    // The service should now keep running and processing incoming connections
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

func compileChaosBinary(sourcePath, monitorIP string, port int, token string, encryptionKey string) (string, error) {
	outputPath := filepath.Join("/tmp", "break_tool")

	ldflags := fmt.Sprintf("-X=main.MonitorIP=%s -X=main.MonitorPortStr=%s -X=main.Token=%s -X=main.EncryptionKey=%s", monitorIP, strconv.Itoa(port), token, encryptionKey)

	cmd := exec.Command("go", "build", "-o", outputPath, "-ldflags", ldflags, sourcePath)
	cmd.Env = append(os.Environ(), "GOOS=linux", "GOARCH=amd64")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("failed to compile chaos binary: %v", err)
	}

	return outputPath, nil
}

// copy the compiled binary to the remote VM using scp
func scpToRemote(ip, localPath, remotePath string) error {
    keyPath := filepath.Join(os.Getenv("HOME"), ".ssh", "id_ed25519")
    // -o StrictHostKeyChecking=no is fine for your local test setup
    cmd := exec.Command(
        "scp",
        "-i", keyPath,
        "-o", "StrictHostKeyChecking=no",
        localPath,
        fmt.Sprintf("vagrant@%s:%s", ip, remotePath),
    )
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
    return cmd.Run()
}

// run the binary remotely via SSH (background)
func runRemoteBinary(ip string, config *ssh.ClientConfig, remotePath string) error {
    client, err := ssh.Dial("tcp", ip+":22", config)
    if err != nil {
        return fmt.Errorf("ssh dial failed: %w", err)
    }
    defer client.Close()

    session, err := client.NewSession()
    if err != nil {
        return fmt.Errorf("new ssh session failed: %w", err)
    }
    defer session.Close()

    // chmod, then start in background; log to /tmp/break_tool.log
    cmd := fmt.Sprintf("chmod +x %s && nohup sudo %s >/tmp/break_tool.log 2>&1 &", remotePath, remotePath)
    session.Stdout = os.Stdout
    session.Stderr = os.Stderr
    return session.Run(cmd)
}


func runRemoteCommandWithListener(ip string, config *ssh.ClientConfig, scriptPath string) error {
	for {
		// Step 1: Find random open port
		listener, err := net.Listen("tcp", ":0") // Open a TCP port and start listening
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

		// Generate the encryption key
		encryptionKey, err := GenerateEncryptionKey(32) // 32 bytes = 64 hex chars
		fmt.Printf("🔑 Generated encryption key: %s\n", encryptionKey)
		if err != nil {
			return fmt.Errorf("failed to generate encryption key: %v", err)
		}
		monitorIP := os.Getenv("MONITOR_ADDRESS")
		localBin, err := compileChaosBinary(scriptPath, monitorIP, port, token, encryptionKey)
		if err != nil {
			return err
		}

		// Ship it to testenv and start it
		const remoteBin = "/tmp/break_tool"
		if err := scpToRemote(ip, localBin, remoteBin); err != nil {
			return fmt.Errorf("scp failed: %w", err)
		}
		if err := runRemoteBinary(ip, config, remoteBin); err != nil {
			return fmt.Errorf("remote start failed: %w", err)
		}

		// Step 2: Keep accepting connections in a loop
		for {
			conn, err := listener.Accept() // Accept an incoming connection
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to accept connection: %v\n", err)
				continue // Continue waiting for new connections even if one fails
			}
			// Handle the connection in a separate goroutine so it doesn't block other connections
			//go handleChaosConnection(conn, token, encryptionKey)
			opperation := handleChaosConnection(conn, token, encryptionKey)
			if opperation == "complete" {
				fmt.Println("✅ Operation completed successfully, exiting listener.")
				listener.Close()
				return nil // Exit the listener loop if operation is complete
			}
		}

		// This point is never reached due to the infinite loop above.
		defer listener.Close()
	}
}

func handleChaosConnection(conn net.Conn, expectedToken string, encryptionKey string) string {
	defer conn.Close()

	reader := bufio.NewReader(conn)

	for {
		// Step 1: Read the 4-byte length prefix
		lengthBuf := make([]byte, 4)
		_, err := io.ReadFull(reader, lengthBuf)
		if err == io.EOF {
			// client disconnected
			fmt.Println("📴 Client disconnected.")
			return "disconnected"
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Error reading message length: %v\n", err)
			return "error"
		}

		msgLen := binary.BigEndian.Uint32(lengthBuf)
		if msgLen == 0 {
			fmt.Fprintf(os.Stderr, "⚠️ Received zero-length message\n")
			continue
		}

		// Step 2: Read the encrypted message
		encryptedData := make([]byte, msgLen)
		_, err = io.ReadFull(reader, encryptedData)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Error reading encrypted message: %v\n", err)
			return "error"
		}

		// Step 3: Decrypt the message
		plaintext, err := DecryptMessage(encryptedData, encryptionKey)
		if err != nil {
			fmt.Fprintf(os.Stderr, "⚠️ Failed to decrypt message: %v\n", err)
			continue
		}

		// Step 4: Decode JSON
		var msg types.ChaosMessage
		if err := json.Unmarshal(plaintext, &msg); err != nil {
			fmt.Fprintf(os.Stderr, "⚠️ Invalid JSON after decryption: %s\n", plaintext)
			continue
		}

		// Step 5: Token check
		if msg.Token == expectedToken {
			fmt.Println("🔐 Token check: ✅ valid")
			msg.TokenCheck = true
		} else {
			fmt.Println("🔐 Token check: ❌ invalid")
			msg.TokenCheck = false
		}

		// Step 6: Process message
		if err := AppendChaosToReport(msg); err != nil {
		    fmt.Fprintf(os.Stderr, "failed to append to report: %v\n", err)
		}

		switch msg.Status {
			case "operation_complete":
				if msg.TokenCheck {
					fmt.Println("✅ Operation completed, continuing to listen for new messages.")
					return "complete"
				} else {
					fmt.Println("❌ Operation_complete received but token check failed")
				}

			case "general":
				// just print the message
				fmt.Printf("📢 General: %s", msg.Message)

			case "chaos_report":
				fmt.Printf("🐛 Chaos Report: %s", msg.Message)
				logPath := "/tmp/chaos_reports.log"
				f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
				if err != nil {
					fmt.Fprintf(os.Stderr, "log open error: %v\n", err)
					break
				}
				// write the full JSON we received (plus newline)
				if _, err := f.Write(append(plaintext, '\n')); err != nil {
					fmt.Fprintf(os.Stderr, "log write error: %v\n", err)
				}
				_ = f.Close()

			default:
				fmt.Printf("⚠️ Unknown message type: %s", msg.Status)
		}
	}
}

// GenerateEncryptionKey generates a secure random encryption key of the specified length (in bytes)
func GenerateEncryptionKey(keyLength int) (string, error) {
	// Generate a random byte slice of the specified length
	key := make([]byte, keyLength)
	_, err := rand.Read(key)
	if err != nil {
		return "", fmt.Errorf("failed to generate encryption key: %v", err)
	}

	// Convert the key to a hex string for easy storage and transfer
	return hex.EncodeToString(key), nil
}

// DecryptMessage decrypts an encrypted message using AES-GCM with the provided encryption key
func DecryptMessage(encryptedData []byte, encryptionKey string) ([]byte, error) {
    // Convert the hex-encoded encryption key to bytes
    key, err := hex.DecodeString(encryptionKey)
    if err != nil {
        return nil, fmt.Errorf("failed to decode encryption key: %v", err)
    }

    // Ensure the key is the correct length for AES (32 bytes for AES-256)
    if len(key) != 32 {
        return nil, fmt.Errorf("invalid encryption key length: %d bytes (expected 32 bytes)", len(key))
    }

    // Separate the nonce (first 12 bytes) and ciphertext (rest of the data)
    nonce, ciphertext := encryptedData[:12], encryptedData[12:]
    // Separate the last 16 bytes (authentication tag)

////////////////////////////////////////////////////////////////////////////////
//// keeping this code here and commented to make debugging easier as we go ////
//// I'll probably remove it after we release version 1                     ////
////////////////////////////////////////////////////////////////////////////////
    //tagCopy := ciphertext[len(ciphertext)-16:]
    //ciphertextCopy := ciphertext[:len(ciphertext)-16] // Remove the tag from ciphertext

    // Log the nonce and ciphertext for debugging
    //fmt.Printf("🔑 Nonce: %x\n", nonce)
    //fmt.Printf("🔒 Ciphertext: %x\n", ciphertextCopy)
    //fmt.Printf("🔑 encryptedData: %s\n", encryptedData[:12])
    //fmt.Printf("🔒 Tag: %x\n", tagCopy)
    //fmt.Printf("🔑 Nonce (%d bytes): %x\n", len(nonce), nonce)
    //fmt.Printf("🔒 Ciphertext (%d bytes): %x\n", len(ciphertextCopy), ciphertextCopy)
    //fmt.Printf("🔐 Tag (%d bytes): %x\n", len(tagCopy), tagCopy)
////////////////////////////////////////////////////////////////////////////////

    // Initialize AES-GCM cipher block
    block, err := aes.NewCipher(key)
    if err != nil {
        return nil, fmt.Errorf("failed to create AES cipher: %v", err)
    }

    // Create an AES-GCM cipher instance
    aesgcm, err := cipher.NewGCM(block)
    if err != nil {
        return nil, fmt.Errorf("failed to create AES-GCM cipher: %v", err)
    }

    // Decrypt the ciphertext using the AES-GCM cipher
    plaintext, err := aesgcm.Open(nil, nonce, ciphertext, nil)
    if err != nil {
        return nil, fmt.Errorf("failed to decrypt message: %v", err)
    }

    fmt.Printf("🔓 Decrypted plaintext: %s\n", plaintext)
    return plaintext, nil
}


// AppendChaosToReport appends a ChaosMessage under its token key
// into $HOME/report.log.json. The JSON file is a map[string][]ChaosMessage.
func AppendChaosToReport(msg types.ChaosMessage) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("resolve home dir: %w", err)
	}
	path := filepath.Join(home, "report.log.json")

	// Ensure file exists with an empty JSON object
	if err := ensureJSONFile(path); err != nil {
		return err
	}

	// Load existing map[token][]ChaosMessage
	logs, err := readReport(path)
	if err != nil {
		return fmt.Errorf("read report: %w", err)
	}

	// Append the entry
	logs[msg.Token] = append(logs[msg.Token], msg)

	// Write back atomically
	if err := writeJSONAtomic(path, logs); err != nil {
		return fmt.Errorf("write report: %w", err)
	}

	return nil
}

// AppendChaosLine parses a single JSON line and appends it using AppendChaosToReport.
func AppendChaosLine(line []byte) error {
	var msg types.ChaosMessage
	if err := json.Unmarshal(line, &msg); err != nil {
		return fmt.Errorf("parse chaos line: %w", err)
	}
	if msg.Token == "" {
		return errors.New("missing token in chaos line")
	}
	return AppendChaosToReport(msg)
}

// --- helpers ---

func ensureJSONFile(path string) error {
	_, statErr := os.Stat(path)
	if os.IsNotExist(statErr) {
		// Make sure parent dir exists (usually HOME, but safe)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return fmt.Errorf("mkdir parents: %w", err)
		}
		// Initialize as {}
		if err := os.WriteFile(path, []byte("{}"), 0o644); err != nil {
			return fmt.Errorf("init report file: %w", err)
		}
		return nil
	}
	return statErr
}

func readReport(path string) (map[string][]types.ChaosMessage, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	// Treat empty as {}
	if len(bytesTrimSpace(data)) == 0 {
		return map[string][]types.ChaosMessage{}, nil
	}
	var logs map[string][]types.ChaosMessage
	if err := json.Unmarshal(data, &logs); err != nil {
		// If the file is corrupt, back it up and start fresh
		_ = os.WriteFile(path+".corrupt.bak", data, 0o644)
		logs = map[string][]types.ChaosMessage{}
	}
	return logs, nil
}

func writeJSONAtomic(path string, v any) error {
	tmp := path + ".tmp"
	out, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	// Write temp file first
	if err := os.WriteFile(tmp, out, 0o644); err != nil {
		return err
	}
	// Atomic replace
	return os.Rename(tmp, path)
}

// bytesTrimSpace avoids importing strings just to trim
func bytesTrimSpace(b []byte) []byte {
	start, end := 0, len(b)
	for start < end && (b[start] == ' ' || b[start] == '\n' || b[start] == '\r' || b[start] == '\t') {
		start++
	}
	for end > start && (b[end-1] == ' ' || b[end-1] == '\n' || b[end-1] == '\r' || b[end-1] == '\t') {
		end--
	}
	return b[start:end]
}
