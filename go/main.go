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
	"sync"
	"time"

	//	"golang.org/x/crypto/ssh"
	cryptohelpers "chaos-agent/library/ssh"
	datatypes "chaos-agent/library/types"

	"golang.org/x/crypto/nacl/box"
)

var (
	activeTokens = make(map[string]bool)
	tokenMutex   sync.Mutex
)

func manageToken(token, action string) error {
	tokenMutex.Lock()
	defer tokenMutex.Unlock()

	switch action {
	case "add":
		if activeTokens[token] {
			return fmt.Errorf("token %s already exists", token)
		}
		activeTokens[token] = true
	case "subtract":
		if !activeTokens[token] {
			return fmt.Errorf("token %s does not exist", token)
		}
		delete(activeTokens, token)
	default:
		return fmt.Errorf("unknown action: %s", action)
	}
	return nil
}

func getActiveTokenCount() int {
	tokenMutex.Lock()
	defer tokenMutex.Unlock()
	return len(activeTokens)
}

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

func readAndDecryptMessage(c net.Conn, pub *[32]byte, priv *[32]byte) (string, error) {
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

		// handleChaosMessage closes conn internally (you already defer there).
		return strings.TrimSpace(string(pt)), nil
	}
}

func acceptLoop(listener net.Listener, pubB64, privB64 string) error {
	pub, err := parseKey32(pubB64)
	if err != nil {
		return err
	}
	priv, err := parseKey32(privB64)
	if err != nil {
		return err
	}
	fmt.Println("Waiting for incoming connections...")

	timeout := 30 * time.Second
	connAccepted := make(chan struct{})
	exit := make(chan struct{})

	go startAcceptLoopTimer(listener, timeout, connAccepted, exit)

	for {
		conn, err := listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return nil
			}
			log.Printf("accept error: %v", err)
			continue
		}
		connAccepted <- struct{}{}
		shouldExit := handleConnection(conn, pub, priv)
		if shouldExit {
			fmt.Println("✅ Operation completed successfully, exiting listener.")
			close(exit)
			return nil
		}
	}
}

func startAcceptLoopTimer(listener net.Listener, timeout time.Duration, connAccepted <-chan struct{}, exit <-chan struct{}) {
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	for {
		select {
		case <-timer.C:
			fmt.Println("⏰ No messages received in 30 seconds, closing listener.")
			if err := listener.Close(); err != nil {
				fmt.Fprintf(os.Stderr, "error closing connection: %v\n", err)
			}
			return
		case <-connAccepted:
			if !timer.Stop() {
				<-timer.C
			}
			timer.Reset(timeout)
		case <-exit:
			return
		}
	}
}

func handleConnection(conn net.Conn, pub, priv *[32]byte) bool {
	decryptedConn, err := readAndDecryptMessage(conn, pub, priv)
	if err != nil {
		log.Printf("readAndDecryptMessage error: %v", err)
		return false
	}
	return handleChaosMessage(decryptedConn)
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
func handleChaosMessage(plaintext string) bool {
	// Step 1: Decode JSON
	var msg datatypes.ChaosMessage
	if err := json.Unmarshal([]byte(plaintext), &msg); err != nil {
		fmt.Printf("⚠️ Invalid JSON after decryption: %s\n", plaintext)
		return false // can't proceed safely
	}

	// Step 3: Handle based on status
	switch msg.Status {
	case "init":
		fmt.Printf("🚀 Init message received: %s\n", msg.Message)
		if err := manageToken(msg.Token, "add"); err != nil {
			fmt.Printf("Error checking token: %v\n", err)
		}
		return false
	case "operation_complete":
		fmt.Printf("Operation_complete: %s\n", msg.Token)
		if err := manageToken(msg.Token, "subtract"); err != nil {
			fmt.Printf("Error checking token: %v\n", err)
		}
		// If no more active tokens, tell listener to exit
		if getActiveTokenCount() == 0 {
			return true
		}
		return false

	case "chaos_report", "error":
		fmt.Printf("🐛 Chaos Report: %s\n", msg.Message)
		logPath := "/tmp/chaos_reports.log"
		f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			fmt.Printf("log open error: %v\n", err)
			return false
		}
		if _, err := f.Write(append([]byte(plaintext), '\n')); err != nil {
			fmt.Printf("log write error: %v\n", err)
		}
		_ = f.Close()
		return false

	case "general":
		fmt.Printf("📢 General: %s\n", msg.Message)
		return false

	case "variable":
		// ... your existing variable handling ...
		// make sure every `break` becomes `return ""` in this case
		// (because you're not inside a loop anymore)
		return false

	default:
		fmt.Printf("⚠️ Unknown message type: %s\n", msg.Status)
		return false
	}
}

func runChaosCycle(breaksDir string) {
	scriptPath, err := pickRandomFile(breaksDir)
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
	// This ensures the listener is closed when runChaosCycle returns,
	// which will unblock acceptLoop if it's still running.
	defer func() {
		if err := listener.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "error closing connection: %v\n", err)
		}
	}()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := acceptLoop(listener, publicKey, privatKey); err != nil {
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

	// Wait for the accept loop to finish (triggered by operation_complete)
	// If runRemote fails above, we return, listener closes, acceptLoop exits, wg is Done.
	wg.Wait()
}

func main() {
	// Pick initial random long interval (5–7 minutes)
	longInterval, err := rand.Int(rand.Reader, big.NewInt(121)) // 0..120
	if err != nil {
		log.Printf("failed to generate long interval: %v", err)
		longInterval = big.NewInt(0)
	}
	longIntervalSecs := longInterval.Int64() + 300 // 300–420 seconds

	counter := int64(0)

	for {
		runChaosCycle("./breaks/cheap")

		// Random sleep for short interval (60–120s)
		n, err := rand.Int(rand.Reader, big.NewInt(61)) // 0..60
		if err != nil {
			log.Printf("failed to generate short sleep: %v", err)
			n = big.NewInt(0)
		}
		shortSleepSecs := n.Int64() + 60
		fmt.Printf("✅ Long interval %s", time.Duration(longIntervalSecs)*time.Second)
		fmt.Printf("Sleeping for %d seconds...\n", shortSleepSecs)
		time.Sleep(time.Duration(shortSleepSecs) * time.Second)

		// Increment counter by short interval sleep
		counter += shortSleepSecs

		// Check if long interval has been reached
		if counter >= longIntervalSecs {
			fmt.Println("✅ Long interval reached, running additional chaos cycle")
			runChaosCycle("./breaks/expensive")

			// Reset counter and pick a new random long interval
			counter = 0
			n, err := rand.Int(rand.Reader, big.NewInt(121)) // 0..120
			if err != nil {
				log.Printf("failed to generate long interval: %v", err)
				n = big.NewInt(0)
			}
			longIntervalSecs = n.Int64() + 300
			fmt.Printf("Next long interval set to %d seconds\n", longIntervalSecs)
		}
	}
}
