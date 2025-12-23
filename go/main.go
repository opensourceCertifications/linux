// Description: This Go program connects to a remote VM via SSH, deploys a chaos testing binary,
// and listens for encrypted messages from the binary to log chaos events and update configuration files.
// It uses AES-GCM for encryption and handles various message types including general logs, chaos reports, variable updates, and operation completion signals.
// It ensures secure communication using a randomly generated token and encryption key for each session.
// It also compiles the chaos binary with embedded configuration parameters and manages its lifecycle on the remote VM.
package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/big"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	//	"golang.org/x/crypto/ssh"
	cryptohelpers "chaos-agent/library/ssh"
	datatypes "chaos-agent/library/types"

	"golang.org/x/crypto/nacl/box"
)

// scp the binary to the remote host using SSH config
func scpUsingSSHConfig(host, localPath, remotePath string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	cfg := filepath.Join(home, ".ssh", "config")

	scpBin, err := exec.LookPath("scp")
	if err != nil {
		return err
	}

	// #nosec G204 -- arguments are not user-controlled; exec.Command does not use a shell
	cmd := exec.Command(scpBin,
		"-F", cfg,
		localPath,
		fmt.Sprintf("%s:%s", host, remotePath),
	)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	return cmd.Run()
}

// run a remote command using SSH config
func runRemote(host, remoteCmd string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	cfg := filepath.Join(home, ".ssh", "config")

	sshBin, err := exec.LookPath("ssh")
	if err != nil {
		return err
	}

	// #nosec G204 -- arguments are not user-controlled; exec.Command does not use a shell
	cmd := exec.Command(sshBin,
		"-F", cfg,
		"-o", "BatchMode=yes",
		host,
		"--",
		remoteCmd,
	)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	return cmd.Run()
}

func pickRandomFile(dir string) (string, error) {
	files, err := filepath.Glob(filepath.Join(dir, "*/*.go"))
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

func setupListener(portHint int) (net.Listener, int, error) {
	// 0.0.0.0 means “listen on all IPv4 interfaces”
	addr := net.JoinHostPort("0.0.0.0", strconv.Itoa(portHint)) // portHint usually 0
	fmt.Println("BIND ADDRESS:", addr)

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to open listener on %s: %w", addr, err)
	}

	port := listener.Addr().(*net.TCPAddr).Port
	return listener, port, nil
}

// func handleConn(c net.Conn) {
//	defer func() {
//		if err := c.Close(); err != nil {
//			log.Printf("error closing connection from %s: %v",
//				c.RemoteAddr(), err)
//		}
//	}()
//
//	from := c.RemoteAddr().String()
//	fmt.Printf("\n--- BEGIN message stream from %s ---\n", from)
//
//	// Copy whatever the client sends straight to stdout until it closes the connection.
//	_, _ = io.Copy(os.Stdout, c)
//
//	fmt.Printf("\n--- END message stream from %s ---\n", from)
//}

func parseKey32(b64 string) (*[32]byte, error) {
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(b64))
	if err != nil {
		return nil, fmt.Errorf("base64 decode key: %w", err)
	}
	if len(raw) != 32 {
		return nil, fmt.Errorf("key must be 32 bytes, got %d", len(raw))
	}
	var k [32]byte
	copy(k[:], raw)
	return &k, nil
}

func handleConnDecrypt(c net.Conn, pub *[32]byte, priv *[32]byte) (string, error) {
	defer func() {
		if err := c.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "error closing connection: %v\n", err)
		}
	}()

	r := bufio.NewReader(c)

	for {
		// 1) Read 4-byte length prefix
		var lenBuf [4]byte
		_, err := io.ReadFull(r, lenBuf[:])
		if err != nil {
			if errors.Is(err, io.EOF) {
				return "", fmt.Errorf("read length error from %s: %v", c.RemoteAddr(), err)
			}
		}

		n := binary.BigEndian.Uint32(lenBuf[:])
		if n == 0 || n > 4*1024*1024 {
			return "", fmt.Errorf("bad frame length %d from %s", n, c.RemoteAddr())
		}

		// 2) Read payload bytes (base64 + "\n")
		payload := make([]byte, n)
		_, err = io.ReadFull(r, payload)
		if err != nil {
			return "", fmt.Errorf("read payload error from %s: %v", c.RemoteAddr(), err)
		}

		payload = bytes.TrimSpace(payload) // removes trailing "\n"

		// 3) Decode base64 -> ciphertext
		ct, err := base64.StdEncoding.DecodeString(string(payload))
		if err != nil {
			log.Printf("bad base64 from %s: %v", c.RemoteAddr(), err)
			continue
		}

		// 4) Decrypt
		pt, ok := box.OpenAnonymous(nil, ct, pub, priv)
		if !ok {
			log.Printf("decrypt failed from %s", c.RemoteAddr())
			continue
		}

		// handleChaosConnection closes conn internally (you already defer there).
		return strings.TrimSpace(string(pt)), nil
	}
}

func acceptAndDecrypt(listener net.Listener, pubB64, privB64 string) error {
	pub, err := parseKey32(pubB64)
	if err != nil {
		return err

	}
	priv, err := parseKey32(privB64)
	if err != nil {
		return err
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return nil
			}
			log.Printf("accept error: %v", err)
			continue
		}
		// go handleConnDecrypt(conn, pub, priv)
		//STARTHERE: you need to put the complete into a counter
		// tokens need to be recoreded, add for each new one subtract for each complete
		// when zero, exit listener
		decryptedConn, err := handleConnDecrypt(conn, pub, priv)
		if err != nil {
			log.Printf("handleConnDecrypt error: %v", err)
			continue
		}
		op := handleChaosConnection(decryptedConn)
		if op == "complete" {
			fmt.Println("✅ Operation completed successfully, exiting listener.")
			return nil
		}
	}
}

func compileChaosBinary(sourcePath, monitorIP string, port int, encryptionKey string) (string, error) {
	outputPath := filepath.Join("/tmp", "break_tool")
	ldflags := fmt.Sprintf(
		"-X=main.MonitorIP=%s -X=main.MonitorPortStr=%s -X=main.EncryptionKey=%s",
		monitorIP, strconv.Itoa(port), encryptionKey,
	)

	// Guardrail 1: only build files under ./breaks and with .go extension
	absSrc, err := filepath.Abs(sourcePath)
	if err != nil {
		return "", fmt.Errorf("abs source: %w", err)
	}
	breaksDir, err := filepath.Abs("breaks")
	if err != nil {
		return "", fmt.Errorf("abs breaks: %w", err)
	}
	if filepath.Ext(absSrc) != ".go" || !strings.HasPrefix(absSrc, breaksDir+string(os.PathSeparator)) {
		return "", fmt.Errorf("refusing to build untrusted source path: %q", absSrc)
	}

	// Guardrail 2: resolve the go tool explicitly
	goBin, err := exec.LookPath("go")
	if err != nil {
		return "", fmt.Errorf("go tool not found: %w", err)
	}

	// Optional: timeout so builds can’t hang this ephemeral service
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// #nosec G204 -- argv validated (source restricted to ./breaks/*.go); explicit tool path; no shell used
	cmd := exec.CommandContext(ctx, goBin, "build", "-o", outputPath, "-ldflags", ldflags, absSrc)

	// Guardrail 3: explicit env (avoid inherited GOFLAGS/-toolexec/etc.)
	cmd.Env = []string{
		"GOOS=linux",
		"GOARCH=amd64",
		"CGO_ENABLED=0",
		"HOME=/tmp",
		// clear potentially dangerous vars
		"GOFLAGS=",
		"GOTOOLCHAIN=local",
	}
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to compile chaos binary: %w", err)
	}
	return outputPath, nil
}

// nolint:cyclop // TODO: split into small handlers (general, report, variable, opComplete)
func handleChaosConnection(plaintext string) string {
	// Step 1: Decode JSON
	var msg datatypes.ChaosMessage
	if err := json.Unmarshal([]byte(plaintext), &msg); err != nil {
		fmt.Printf("⚠️ Invalid JSON after decryption: %s\n", plaintext)
		return "" // can't proceed safely
	}

	// Step 3: Handle based on status
	switch msg.Status {
	case "init":
		fmt.Printf("🚀 Init message received: %s\n", msg.Message)
		return ""
	case "operation_complete":
		if msg.TokenCheck {
			fmt.Println("✅ Operation completed, continuing to listen for new messages.")
			return "complete"
		}
		fmt.Println("❌ Operation_complete received but token check failed")
		return ""

	case "general":
		fmt.Printf("📢 General: %s\n", msg.Message)
		return ""

	case "chaos_report", "error":
		fmt.Printf("🐛 Chaos Report: %s\n", msg.Message)
		logPath := "/tmp/chaos_reports.log"
		f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			fmt.Printf("log open error: %v\n", err)
			return ""
		}
		if _, err := f.Write(append([]byte(plaintext), '\n')); err != nil {
			fmt.Printf("log write error: %v\n", err)
		}
		_ = f.Close()
		return ""

	case "variable":
		// ... your existing variable handling ...
		// make sure every `break` becomes `return ""` in this case
		// (because you're not inside a loop anymore)
		return ""

	default:
		fmt.Printf("⚠️ Unknown message type: %s\n", msg.Status)
		return ""
	}
}

// func handleChaosConnection(plaintext string) string {
//	// Step 1: Decode JSON
//	var msg datatypes.ChaosMessage
//	if err := json.Unmarshal([]byte(plaintext), &msg); err != nil {
//		fmt.Printf(os.Stderr, "⚠️ Invalid JSON after decryption: %s\n", plaintext)
//	}
//
//	// Step 3: Handle based on status
//	switch msg.Status {
//	case "operation_complete":
//		if msg.TokenCheck {
//			fmt.Println("✅ Operation completed, continuing to listen for new messages.")
//			return "complete"
//		}
//		fmt.Println("❌ Operation_complete received but token check failed")
//
//	case "general":
//		// just print the message
//		fmt.Printf("📢 General: %s", msg.Message)
//
//	case "chaos_report", "error":
//		fmt.Printf("🐛 Chaos Report: %s", msg.Message)
//		logPath := "/tmp/chaos_reports.log"
//		f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
//		if err != nil {
//			fmt.Printf(os.Stderr, "log open error: %v\n", err)
//			break
//		}
//		// write the full JSON we received (plus newline)
//		if _, err := f.Write(append([]byte(plaintext), '\n')); err != nil {
//			fmt.Printf(os.Stderr, "log write error: %v\n", err)
//		}
//		_ = f.Close()
//
//	case "variable":
//		parts := strings.SplitN(msg.Message, ",", 2)
//		if len(parts) != 2 {
//			fmt.Printf("⚠️ Invalid variable message format: %s\n", msg.Message)
//			break
//		}
//
//		key := parts[0]
//		value := parts[1]
//		filePath := os.Getenv("ANSIBLE_VARS_PATH")
//
//		// Step 1: Load existing YAML if present
//		vars := make(map[string][]string)
//		// #nosec G304 -- filePath is fixed under ../ansible/; not user-controlled
//		if data, err := os.ReadFile(filePath); err == nil {
//			if len(data) > 0 {
//				if err := yaml.Unmarshal(data, &vars); err != nil {
//					fmt.Printf(os.Stderr, "YAML unmarshal error: %v\n", err)
//					break
//				}
//			}
//		}
//
//		// Step 2: Append value (avoid duplicates if you like)
//		list := vars[key]
//		// Optional: deduplicate
//		alreadyExists := slices.Contains(list, value)
//		if !alreadyExists {
//			vars[key] = append(list, value)
//		}
//
//		// Step 3: Write back full YAML
//		out, err := yaml.Marshal(vars)
//		if err != nil {
//			fmt.Printf(os.Stderr, "YAML marshal error: %v\n", err)
//			break
//		}
//
//		if err := os.WriteFile(filePath, out, 0600); err != nil {
//			fmt.Printf(os.Stderr, "variable file write error: %v\n", err)
//		} else {
//			fmt.Printf("✅ Updated %s with %s -> %s\n", filePath, key, value)
//		}
//
//	default:
//		return fmt.Printf("⚠️ Unknown message type: %s", msg.Status)
//	}
//}

func main() {
	scriptPath, err := pickRandomFile("breaks")
	if err != nil {
		log.Printf("Failed to pick test file: %v", err)
		return
	}
	fmt.Printf("🎯 Selected test script: %s\n", scriptPath)

	// privatKey, publicKey, err := cryptohelpers.GenerateEd25519KeyPair()
	publicKey, privatKey, err := cryptohelpers.GenerateKeys()
	if err != nil {
		log.Printf("Failed to generate keys: %s", err)
		return
	}
	fmt.Println("PRIVATE KEY:\n", string(privatKey))
	fmt.Println("PUBLIC KEY:\n", string(publicKey))

	monitorAddr := os.Getenv("MONITOR_ADDRESS")
	if monitorAddr == "" {
		log.Printf("MONITOR_ADDRESS not set")
		return
	}

	listener, port, err := setupListener(0)
	if err != nil {
		log.Printf("Failed to setup listender: %s", err)
		return
	}
	defer func() {
		if err := listener.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "error closing connection: %v\n", err)
		}
	}()

	go func() {
		if err := acceptAndDecrypt(listener, publicKey, privatKey); err != nil {
			log.Printf("accept loop stopped: %v", err)
		}
	}()

	fmt.Println("LISTENING ON PORT:", port)
	fmt.Println("listner:", listener)

	localBin, err := compileChaosBinary(scriptPath, monitorAddr, port, publicKey)
	if err != nil {
		log.Printf("error when compling binary: %s", err)
		return
	}
	fmt.Println("COMPILED BINARY AT:", localBin)

	const remoteBin = "/tmp/break_tool"

	if err := scpUsingSSHConfig("testenv", localBin, remoteBin); err != nil {
		log.Printf("scp failed: %v", err)
		return
	}

	start := time.Now()
	if err := runRemote("testenv", remoteBin); err != nil {
		log.Printf("remote run failed: %v", err)
		return
	}
	dur := time.Since(start)
	log.Printf("remote run finished in %s", dur)
}
