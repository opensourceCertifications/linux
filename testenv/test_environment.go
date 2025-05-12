package main

import (
	"crypto/sha256"
	"encoding/json"
	"math/rand"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"time"

	"github.com/pquerna/otp/totp"
	"testenv/breaks/breaks"
)

type Heartbeat struct {
	Timestamp string `json:"timestamp"`
	Status    string `json:"status"`
	Service   string `json:"service"`
	Version   string `json:"version"`
	TOTP      string `json:"totp"`
	Checksum  string `json:"checksum"`
	First     bool   `json:"first"`
}

const (
	sharedSecret   = "JBSWY3DPEHPK3PXP"
	heartbeatDelay = 500 * time.Millisecond
	monitorPort    = 9000
)

func chaosInjector(address string, breakHistory *[]string) {
	for {
		delay := GetRandomDuration()
		log.Printf("Chaos injector sleeping for %v before next break...", delay)
		time.Sleep(delay)

		breakFile, err := ExecuteRandomBreak()
		if err != nil {
			log.Printf("Failed to select a random break: %v", err)
			continue
		}

		log.Printf("Executing chaos break: %s", breakFile)

		err = SendBreakNameToMonitor(address, breakFile)
		if err != nil {
			log.Printf("Failed to send break name to monitor: %v", err)
			continue
		}

		*breakHistory = append(*breakHistory, breakFile)
	}
}

func main() {
	if len(os.Args) != 2 {
		log.Fatalf("Usage: %s <monitor_ip>", os.Args[0])
	}

	monitorIP := os.Args[1]
	address := fmt.Sprintf("%s:%d", monitorIP, monitorPort)

	waitForMonitor(address)
	log.Println("Connected to monitor. Starting heartbeat...")

	// === New chaos injection ===
	breakHistory := []string{}
	go chaosInjector(address, &breakHistory)

	startHeartbeat(address)
}

func waitForMonitor(address string) {
	log.Println("Waiting for monitor service to become available...")

	for {
		conn, err := net.DialTimeout("tcp", address, 1*time.Second)
		if err != nil {
			log.Println("Monitor not ready, retrying...")
			time.Sleep(1 * time.Second)
			continue
		}
		conn.Close()
		break
	}
}

func startHeartbeat(address string) {
	first := true

	for {
		totpCode, err := totp.GenerateCode(sharedSecret, time.Now())
		if err != nil {
			log.Printf("Failed to generate TOTP: %v", err)
			continue
		}

		// Compute checksum of the running binary (os.Args[0])
		checksum, err := ComputeChecksum(os.Args[0])
		if err != nil {
			log.Printf("Failed to compute checksum: %v", err)
			continue
		}

		heartbeat := Heartbeat{
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Status:    "alive",
			Service:   "test_environment",
			Version:   "1.0.0",
			TOTP:      totpCode,
			Checksum:  checksum,
			First:     first,
		}

		first = false

		data, err := json.Marshal(heartbeat)
		if err != nil {
			log.Printf("Failed to marshal heartbeat: %v", err)
			continue
		}

		sendMessage(address, data)
		time.Sleep(heartbeatDelay)
	}
}

func sendMessage(address string, data []byte) {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		log.Printf("Failed to connect to %s: %v", address, err)
		return
	}
	defer conn.Close()

	_, err = conn.Write(data)
	if err != nil {
		log.Printf("Failed to send message: %v", err)
	}
}

func ComputeChecksum(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

func GetRandomDuration() time.Duration {
	rand.Seed(time.Now().UnixNano())
	min := 5 * time.Minute
	max := 7 * time.Minute
	return min + time.Duration(rand.Int63n(int64(max-min)))
}

var chaosFunctions = []struct {
	Name string
	Fn   func() error
}{
	{"BreakBootLoader", breaks.BreakBootLoader},
	// add more here as needed
}

func ExecuteRandomBreak() (string, error) {
	rand.Seed(time.Now().UnixNano())
	selected := chaosFunctions[rand.Intn(len(chaosFunctions))]
	log.Printf("Selected chaos function: %s", selected.Name)
	err := selected.Fn()
	return selected.Name, err
}

func SendBreakNameToMonitor(address, breakName string) error {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return fmt.Errorf("failed to connect to monitor at %s: %w", address, err)
	}
	defer conn.Close()

	_, err = conn.Write([]byte(breakName))
	if err != nil {
		return fmt.Errorf("failed to send break name to monitor: %w", err)
	}

	return nil
}

