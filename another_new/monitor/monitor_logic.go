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
	"crypto/aes"
	"crypto/cipher"  // Import cipher package for AES-GCM
	"encoding/json"
	"path/filepath"
	"time"
	mathrand "math/rand"
	"os/exec"
	"encoding/binary"

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

    fmt.Println("✅ Successfully ran 'touch /tmp/hello' on", targetIP)

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

	ldflags := fmt.Sprintf("-X 'main.MonitorIP=%s' -X 'main.MonitorPort=%d' -X 'main.Token=%s'", monitorIP, port, token)

	cmd := exec.Command("go", "build", "-o", outputPath, "-ldflags", ldflags, sourcePath)
	cmd.Env = append(os.Environ(), "GOOS=linux", "GOARCH=amd64")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	//fmt.Printf("🛠️ Compiling %s → %s...\n", sourcePath, outputPath)
	//err := cmd.Run()
	//if err != nil {
	//	return "", fmt.Errorf("failed to compile chaos binary: %v", err)
	//}

	fmt.Printf("✅ Compiled binary written to: %s\n", outputPath)
	return outputPath, nil
}

func runRemoteCommandWithListener(ip string, config *ssh.ClientConfig, scriptPath string) error {
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

	// Step 2: Keep accepting connections in a loop
	for {
		conn, err := listener.Accept() // Accept an incoming connection
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to accept connection: %v\n", err)
			continue // Continue waiting for new connections even if one fails
		}
		// Handle the connection in a separate goroutine so it doesn't block other connections
		go handleChaosConnection(conn, token, encryptionKey)
	}

	// This point is never reached due to the infinite loop above.
	return nil
}

func handleChaosConnection(conn net.Conn, expectedToken string, encryptionKey string) {
	defer conn.Close()

	reader := bufio.NewReader(conn)

	for {
		// Step 1: Read the 4-byte length prefix
		lengthBuf := make([]byte, 4)
		_, err := io.ReadFull(reader, lengthBuf)
		if err == io.EOF {
			// client disconnected
			fmt.Println("📴 Client disconnected.")
			return
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Error reading message length: %v\n", err)
			return
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
			return
		}

		fmt.Printf("🔹 Raw input (encrypted %d bytes): %x\n", msgLen, encryptedData)

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

		fmt.Println("🔓 Decrypted JSON message:", string(plaintext))

		// Step 5: Token check
		if msg.Token == expectedToken {
			fmt.Println("🔐 Token check: ✅ valid")
		} else {
			fmt.Println("🔐 Token check: ❌ invalid")
		}

		// Step 6: Process message
		fmt.Printf("📨 Status: %-20s  Message: %s\n", msg.Status, msg.Message)

		if msg.Status == "operation_complete" {
			fmt.Println("✅ Operation completed, continuing to listen for new messages.")
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

	fmt.Printf("🔑 Generated encryption key of length %d bytes\n", keyLength)
	fmt.Printf("🔑 Key (hex): %s\n", hex.EncodeToString(key))
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

    tagCopy := ciphertext[len(ciphertext)-16:]
    ciphertextCopy := ciphertext[:len(ciphertext)-16] // Remove the tag from ciphertext

    // Log the nonce and ciphertext for debugging
    fmt.Printf("🔑 Nonce: %x\n", nonce)
    fmt.Printf("🔒 Ciphertext: %x\n", ciphertextCopy)
    fmt.Printf("🔑 encryptedData: %s\n", encryptedData[:12])
    fmt.Printf("🔒 Tag: %x\n", tagCopy)
    fmt.Printf("🔑 Nonce (%d bytes): %x\n", len(nonce), nonce)
    fmt.Printf("🔒 Ciphertext (%d bytes): %x\n", len(ciphertextCopy), ciphertextCopy)
    fmt.Printf("🔐 Tag (%d bytes): %x\n", len(tagCopy), tagCopy)

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


