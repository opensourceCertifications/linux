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
	"crypto/aes"
	"crypto/cipher" // Import cipher package for AES-GCM
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
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
	"slices"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v2"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"

	datatypes "chaos-agent/shared/types"
)

// ---------- small helpers (exit on error to keep main flat) ----------

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format, args...)
	os.Exit(1)
}

func getTargetIPOrExit() string {
	ip := os.Getenv("TESTENV_ADDRESS")
	if ip == "" {
		exitf("Error: TESTENV_ADDRESS environment variable not set\n")
	}
	return ip
}

func signerFromDefaultKeyOrExit() ssh.Signer {
	home, err := os.UserHomeDir()
	if err != nil {
		exitf("Error: could not resolve home directory\n%s", err)
	}
	sshDir := filepath.Join(home, ".ssh")
	keyPath := filepath.Join(sshDir, "id_ed25519")

	clean := filepath.Clean(keyPath)
	if !strings.HasPrefix(clean, sshDir+string(os.PathSeparator)) {
		exitf("Refusing to read private key outside %s\n", sshDir)
	}

	// #nosec G304 -- path is derived from os.UserHomeDir() and a fixed filename under ~/.ssh
	key, err := os.ReadFile(clean)
	if err != nil {
		exitf("Error reading private key: %v\n", err)
	}
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		exitf("Error parsing private key: %v\n", err)
	}
	return signer
}

func buildSSHConfigOrExit(signer ssh.Signer) *ssh.ClientConfig {
	home, err := os.UserHomeDir()
	if err != nil {
		exitf("Error: could not resolve home directory\n")
	}
	khPath := filepath.Join(home, ".ssh", "known_hosts")
	khChecker, err := knownhosts.New(khPath)
	if err != nil {
		exitf("failed to load known_hosts: %v\n", err)
	}

	return &ssh.ClientConfig{
		User: "root",
		Auth: []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			if err := khChecker(hostname, remote, key); err != nil {
				if ke, ok := err.(*knownhosts.KeyError); ok {
					if len(ke.Want) == 0 {
						return fmt.Errorf("unknown host %s (fp %s). Add its key to %s",
							hostname, ssh.FingerprintSHA256(key), khPath)
					}
					return fmt.Errorf("host key mismatch for %s. Expected one of %v, got %s",
						hostname, ke.Want, ssh.FingerprintSHA256(key))
				}
				return err
			}
			return nil
		},
		HostKeyAlgorithms: []string{
			ssh.KeyAlgoED25519,
			ssh.KeyAlgoRSA,
			ssh.KeyAlgoECDSA256, ssh.KeyAlgoECDSA384, ssh.KeyAlgoECDSA521,
		},
	}
}

func pickRandomBreakScriptOrExit(dir string) string {
	p, err := pickRandomFile(dir)
	if err != nil {
		exitf("Failed to pick test file: %v\n", err)
	}
	return p
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

// imports: context, fmt, os, os/exec, path/filepath, strconv, strings, time

func compileChaosBinary(sourcePath, monitorIP string, port int, token, encryptionKey string) (string, error) {
	outputPath := filepath.Join("/tmp", "break_tool")
	ldflags := fmt.Sprintf(
		"-X=main.MonitorIP=%s -X=main.MonitorPortStr=%s -X=main.Token=%s -X=main.EncryptionKey=%s",
		monitorIP, strconv.Itoa(port), token, encryptionKey,
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

// imports you'll need:
// "context", "fmt", "net", "os", "os/exec", "path/filepath", "strings", "time"

func scpToRemote(ip, localPath, remotePath string) error {
	// --- Validate inputs (prove argv aren't attacker-controlled) ---
	if net.ParseIP(ip) == nil {
		return fmt.Errorf("invalid IP: %q", ip)
	}
	// Only allow copying to /tmp on the remote (tighten as you like)
	if !strings.HasPrefix(remotePath, "/tmp/") {
		return fmt.Errorf("refusing remote path outside /tmp: %q", remotePath)
	}

	absLocal, err := filepath.Abs(localPath)
	if err != nil {
		return fmt.Errorf("abs local: %w", err)
	}
	fi, err := os.Stat(absLocal)
	if err != nil {
		return fmt.Errorf("stat local: %w", err)
	}
	if !fi.Mode().IsRegular() {
		return fmt.Errorf("local path is not a regular file: %q", absLocal)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("resolve home: %w", err)
	}
	sshDir := filepath.Join(home, ".ssh")
	keyPath := filepath.Join(sshDir, "id_ed25519")
	khPath := filepath.Join(sshDir, "known_hosts")

	// Resolve scp and set a timeout so we can’t hang forever
	scpBin, err := exec.LookPath("scp")
	if err != nil {
		return fmt.Errorf("scp not found in PATH: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// #nosec G204 -- argv validated (IP, remotePath restricted to /tmp, local is regular file);
	// no shell used; host key verification enforced via UserKnownHostsFile
	cmd := exec.CommandContext(ctx, scpBin,
		"-i", keyPath,
		"-o", "IdentitiesOnly=yes",
		"-o", "BatchMode=yes",
		"-o", "PasswordAuthentication=no",
		"-o", "KbdInteractiveAuthentication=no",
		"-o", "StrictHostKeyChecking=yes",
		"-o", "UserKnownHostsFile="+khPath,
		absLocal,
		fmt.Sprintf("root@%s:%s", ip, remotePath),
	)
	// Minimal, explicit env to avoid hostile GOFLAGS/etc. leaking in (optional here)
	cmd.Env = []string{
		"PATH=/usr/bin:/bin", // enough to resolve ld/rt deps used by scp
		"HOME=" + home,       // some libs consult HOME; harmless and predictable here
		"LANG=C", "LC_ALL=C", // stable parsing/messages
		// Note: we intentionally do NOT pass LD_PRELOAD/LD_LIBRARY_PATH/DYLD_*,
		// SSH_AUTH_SOCK, SSH_ASKPASS, etc.
	}
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr

	return cmd.Run()
}

// run the binary remotely via SSH (background)
func runRemoteBinary(ip string, config *ssh.ClientConfig, remotePath string) error {
	client, err := ssh.Dial("tcp", ip+":22", config)
	if err != nil {
		return fmt.Errorf("ssh dial failed: %w", err)
	}
	// defer client.Close()
	defer func() {
		if err := client.Close(); err != nil && !errors.Is(err, io.EOF) {
			log.Printf("client close ssh client: %v", err)
		}
	}()

	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("new ssh session failed: %w", err)
	}
	defer func() {
		if err := session.Close(); err != nil && !errors.Is(err, io.EOF) {
			log.Printf("close ssh session: %v", err)
		}
	}()

	// chmod, then start in background; log to /tmp/break_tool.log
	cmd := fmt.Sprintf("chmod +x %s && nohup %s >/tmp/break_tool.log 2>&1 &", remotePath, remotePath)
	session.Stdout = os.Stdout
	session.Stderr = os.Stderr
	return session.Run(cmd)
}

// closeQuietly is just to satisfy errcheck on Close().
func closeQuietly(c io.Closer, label string) {
	if err := c.Close(); err != nil && !errors.Is(err, io.EOF) {
		log.Printf("close %s: %v", label, err)
	}
}

// setupListenerAndSecrets opens a random TCP port and creates the token+encKey.
func setupListenerAndSecrets() (ln net.Listener, port int, token, encKey string, err error) {
	mon := os.Getenv("MONITOR_ADDRESS")
	if mon == "" {
		return nil, 0, "", "", fmt.Errorf("MONITOR_ADDRESS not set")
	}

	// Use JoinHostPort so IPv6 like "fd00::1" becomes "[fd00::1]:0"
	addr := net.JoinHostPort(mon, "0")

	// Bind only to this local IP, random port
	// #nosec G102 -- binding to a specific interface via MONITOR_ADDRESS (not 0.0.0.0)
	ln, err = net.Listen("tcp", addr)
	if err != nil {
		return nil, 0, "", "", fmt.Errorf("failed to open listener on %s: %w", addr, err)
	}

	port = ln.Addr().(*net.TCPAddr).Port

	token, err = generateToken(16)
	if err != nil {
		closeQuietly(ln, "listener")
		return nil, 0, "", "", fmt.Errorf("token: %w", err)
	}

	encKey, err = GenerateEncryptionKey(32)
	if err != nil {
		closeQuietly(ln, "listener")
		return nil, 0, "", "", fmt.Errorf("enc key: %w", err)
	}

	// SUGGESTION: remove sensitive logs in production
	fmt.Printf("🔑 Generated token: %s\n", token)
	fmt.Printf("📡 Listening on %s:%d\n", mon, port)
	fmt.Printf("🔑 Generated encryption key: %s\n", encKey)
	return ln, port, token, encKey, nil
}

// deployRemote compiles the chaos binary, scp’s it, and starts it via SSH.
func deployRemote(ip string, cfg *ssh.ClientConfig, srcPath, monitorIP string, port int, token, encKey string) error {
	localBin, err := compileChaosBinary(srcPath, monitorIP, port, token, encKey)
	if err != nil {
		return err
	}
	//SUGGESTION: make random remote srcPath
	// delete after execution.
	const remoteBin = "/tmp/break_tool"
	if err := scpToRemote(ip, localBin, remoteBin); err != nil {
		return fmt.Errorf("scp failed: %w", err)
	}
	if err := runRemoteBinary(ip, cfg, remoteBin); err != nil {
		return fmt.Errorf("remote start failed: %w", err)
	}
	return nil
}

// acceptLoop serves connections until the operation completes.
func acceptLoop(ln net.Listener, token, encKey string) (done bool, err error) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to accept connection: %v\n", err)
			continue
		}
		// handleChaosConnection closes conn internally (you already defer there).
		op := handleChaosConnection(conn, token, encKey)
		if op == "complete" {
			fmt.Println("✅ Operation completed successfully, exiting listener.")
			return true, nil
		}
	}
}

// Orchestrator: now short and readable.
func runRemoteCommandWithListener(ip string, cfg *ssh.ClientConfig, scriptPath string) error {
	for {
		ln, port, token, encKey, err := setupListenerAndSecrets()
		if err != nil {
			return err
		}

		monitorIP := os.Getenv("MONITOR_ADDRESS")
		if err := deployRemote(ip, cfg, scriptPath, monitorIP, port, token, encKey); err != nil {
			closeQuietly(ln, "listener")
			return err
		}

		done, err := acceptLoop(ln, token, encKey)
		closeQuietly(ln, "listener")
		if err != nil {
			return err
		}
		if done {
			return nil
		}
	}
}

// nolint:cyclop // TODO: split into small handlers (general, report, variable, opComplete)
func handleChaosConnection(conn net.Conn, expectedToken string, encryptionKey string) string {
	// defer conn.Close()
	defer func() {
		if err := conn.Close(); err != nil {
			log.Printf("conn close ssh client: %v", err)
		}
	}()

	// SUGGESTION: this can only handle one message at a time; if you want
	// multiple concurrent messages, spawn goroutines here.
	// For simplicity, we handle one message at a time in sequence.
	// bufio.Reader for easier reading
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
		// SUGGESTION: set a max length to avoid DoS
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
		var msg datatypes.ChaosMessage
		if err := json.Unmarshal(plaintext, &msg); err != nil {
			fmt.Fprintf(os.Stderr, "⚠️ Invalid JSON after decryption: %s\n", plaintext)
			continue
		}

		// Step 5: Token check
		if msg.Token == expectedToken {
			fmt.Println("🔐 Token check: ✅ valid")
			msg.TokenCheck = true

			// Step 6: Process message
			if err := AppendChaosToReport(msg); err != nil {
				fmt.Fprintf(os.Stderr, "failed to append to report: %v\n", err)
			}
		} else {
			fmt.Println("🔐 Token check: ❌ invalid")
			msg.TokenCheck = false
		}

		// Step 7: Handle based on status
		switch msg.Status {
		case "operation_complete":
			if msg.TokenCheck {
				fmt.Println("✅ Operation completed, continuing to listen for new messages.")
				return "complete"
			}
			fmt.Println("❌ Operation_complete received but token check failed")

		case "general":
			// just print the message
			fmt.Printf("📢 General: %s", msg.Message)

		case "chaos_report":
			fmt.Printf("🐛 Chaos Report: %s", msg.Message)
			logPath := "/tmp/chaos_reports.log"
			f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
			if err != nil {
				fmt.Fprintf(os.Stderr, "log open error: %v\n", err)
				break
			}
			// write the full JSON we received (plus newline)
			if _, err := f.Write(append(plaintext, '\n')); err != nil {
				fmt.Fprintf(os.Stderr, "log write error: %v\n", err)
			}
			_ = f.Close()

		case "variable":
			parts := strings.SplitN(msg.Message, ",", 2)
			if len(parts) != 2 {
				fmt.Printf("⚠️ Invalid variable message format: %s\n", msg.Message)
				break
			}

			key := parts[0]
			value := parts[1]
			filePath := os.Getenv("ANSIBLE_VARS_PATH")

			// Step 1: Load existing YAML if present
			vars := make(map[string][]string)
			// #nosec G304 -- filePath is fixed under ../ansible/; not user-controlled
			if data, err := os.ReadFile(filePath); err == nil {
				if len(data) > 0 {
					if err := yaml.Unmarshal(data, &vars); err != nil {
						fmt.Fprintf(os.Stderr, "YAML unmarshal error: %v\n", err)
						break
					}
				}
			}

			// Step 2: Append value (avoid duplicates if you like)
			list := vars[key]
			// Optional: deduplicate
			alreadyExists := slices.Contains(list, value)
			if !alreadyExists {
				vars[key] = append(list, value)
			}

			// Step 3: Write back full YAML
			out, err := yaml.Marshal(vars)
			if err != nil {
				fmt.Fprintf(os.Stderr, "YAML marshal error: %v\n", err)
				break
			}

			if err := os.WriteFile(filePath, out, 0600); err != nil {
				fmt.Fprintf(os.Stderr, "variable file write error: %v\n", err)
			} else {
				fmt.Printf("✅ Updated %s with %s -> %s\n", filePath, key, value)
			}

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
	//// I'll probably remove it after we release version 1					 ////
	////////////////////////////////////////////////////////////////////////////////
	// tagCopy := ciphertext[len(ciphertext)-16:]
	// ciphertextCopy := ciphertext[:len(ciphertext)-16] // Remove the tag from ciphertext

	// Log the nonce and ciphertext for debugging
	// fmt.Printf("🔑 Nonce: %x\n", nonce)
	// fmt.Printf("🔒 Ciphertext: %x\n", ciphertextCopy)
	// fmt.Printf("🔑 encryptedData: %s\n", encryptedData[:12])
	// fmt.Printf("🔒 Tag: %x\n", tagCopy)
	// fmt.Printf("🔑 Nonce (%d bytes): %x\n", len(nonce), nonce)
	// fmt.Printf("🔒 Ciphertext (%d bytes): %x\n", len(ciphertextCopy), ciphertextCopy)
	// fmt.Printf("🔐 Tag (%d bytes): %x\n", len(tagCopy), tagCopy)
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
func AppendChaosToReport(msg datatypes.ChaosMessage) error {
	// the file size could grow large over time, but for simplicity
	// we keep it simple and read the whole file each time.
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
	var msg datatypes.ChaosMessage
	if err := json.Unmarshal(line, &msg); err != nil {
		return fmt.Errorf("parse chaos line: %w", err)
	}
	if msg.Token == "" {
		return errors.New("missing token in chaos line")
	}
	return AppendChaosToReport(msg)
}

func ensureJSONFile(path string) error {
	_, statErr := os.Stat(path)
	if os.IsNotExist(statErr) {
		// Make sure parent dir exists (usually HOME, but safe)
		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			return fmt.Errorf("mkdir parents: %w", err)
		}
		// Initialize as {}
		if err := os.WriteFile(path, []byte("{}"), 0o600); err != nil {
			return fmt.Errorf("init report file: %w", err)
		}
		return nil
	}
	return statErr
}

func readReport(path string) (map[string][]datatypes.ChaosMessage, error) {
	// #nosec G304 -- path is fixed under $HOME; not user-controlled
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	// Treat empty as {}
	if len(bytesTrimSpace(data)) == 0 {
		return map[string][]datatypes.ChaosMessage{}, nil
	}
	var logs map[string][]datatypes.ChaosMessage
	if err := json.Unmarshal(data, &logs); err != nil {
		// If the file is corrupt, back it up and start fresh
		_ = os.WriteFile(path+".corrupt.bak", data, 0o600)
		logs = map[string][]datatypes.ChaosMessage{}
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
	if err := os.WriteFile(tmp, out, 0o600); err != nil {
		return err
	}
	// Atomic replace
	return os.Rename(tmp, path)
}

// bytesTrimSpace avoids importing strings just to trim
func bytesTrimSpace(b []byte) []byte {
	return bytes.TrimSpace(b)
}

// ------------------------------ main ------------------------------

func main() {
	targetIP := getTargetIPOrExit()
	signer := signerFromDefaultKeyOrExit()
	config := buildSSHConfigOrExit(signer)

	scriptPath := pickRandomBreakScriptOrExit("breaks")
	fmt.Printf("🎯 Selected test script: %s\n", scriptPath)

	if err := runRemoteCommandWithListener(targetIP, config, scriptPath); err != nil {
		exitf("Error: %v\n", err)
	}
}
