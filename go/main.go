// Description: This Go program connects to a remote VM via SSH, deploys a chaos testing binary,
// and listens for encrypted messages from the binary to log chaos events and update configuration files.
// It uses AES-GCM for encryption and handles various message types including general logs, chaos reports, variable updates, and operation completion signals.
// It ensures secure communication using a randomly generated token and encryption key for each session.
// It also compiles the chaos binary with embedded configuration parameters and manages its lifecycle on the remote VM.
package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"math/big"
	"crypto/rand"
	"context"
	"os/exec"
	"strconv"
	"strings"
	"time"
	"chaos-agent/library/ssh"
)

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

func setupListener(monitorAddr string) (net.Listener, int, error) {

	addr := net.JoinHostPort(monitorAddr, "0")
	fmt.Println("ADDRESS:", addr)

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to open listener on %s: %w", addr, err)
	}

	port := listener.Addr().(*net.TCPAddr).Port

	return listener, port, nil
}

func compileChaosBinary(sourcePath, monitorIP string, port int, encryptionKey string) (string, error) {
	outputPath := filepath.Join("/tmp", "break_tool")
	ldflags := fmt.Sprintf(
		"-X=main.MonitorIP=%s -X=main.MonitorPortStr=%s -X=main.Token=%s -X=main.EncryptionKey=%s",
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

func main() {
	scriptPath, err := pickRandomFile("breaks")
	if err != nil {
		log.Fatal("Failed to pick test file: %v\n", err)
	}
	fmt.Printf("🎯 Selected test script: %s\n", scriptPath)

	//privatKey, publicKey, err := cryptohelpers.GenerateEd25519KeyPair()
	privatKey, publicKey, err := cryptohelpers.GenerateKeys()
	if err != nil {
		log.Fatal(err)
	}

	monitorAddr := os.Getenv("MONITOR_ADDRESS")
	if monitorAddr == "" {
		log.Fatal("MONITOR_ADDRESS not set")
	}

	listener, port, err := setupListener(monitorAddr)
	if err != nil {
		log.Fatal(err)
	}

	localBin, err := compileChaosBinary(scriptPath, monitorAddr, port, publicKey)
	if err != nil {
		fmt.Errorf("error when compling binary: %s", err)
	}

	fmt.Println("PRIVATE KEY:\n", string(privatKey))
	fmt.Println("PUBLIC KEY:\n", string(publicKey))
	fmt.Println("LISTENING ON PORT:", port)
	fmt.Println("listner:", listener)
	fmt.Println("COMPILED BINARY AT:", localBin)
}
