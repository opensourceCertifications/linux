package main

import (
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"os"
	"time"

	"github.com/pquerna/otp/totp"
	"testenv/breaks/breaks"
	"testenv/internal/comm"
	"github.com/opensourceCertifications/linux/shared/types"
)

const (
	sharedSecret   = "JBSWY3DPEHPK3PXP"
	heartbeatDelay = 500 * time.Millisecond
	monitorPort    = 9000
)

func chaosInjector(address string, breakHistory *[]string) {
	comm.SendMessage("general", "this is a test")
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

		report := types.ChaosReport{
			Timestamp: time.Now().Format(time.RFC3339),
			Action:    fmt.Sprintf("Executed chaos break: %s", breakFile),
			Agent:     "test_environment",
		}
		comm.SendMessage("chaos_report", report)
		*breakHistory = append(*breakHistory, breakFile)
	}
}

func main() {
	if len(os.Args) != 2 {
		log.Fatalf("Usage: %s <monitor_ip>", os.Args[0])
	}

	monitorIP := os.Args[1]
	address := fmt.Sprintf("%s:%d", monitorIP, monitorPort)

	comm.Init(address)

	waitForMonitor(address)
	log.Println("Connected to monitor. Starting heartbeat...")

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

		checksum, err := ComputeChecksum(os.Args[0])
		if err != nil {
			log.Printf("Failed to compute checksum: %v", err)
			continue
		}

		heartbeat := types.Heartbeat{
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Status:    "alive",
			Service:   "test_environment",
			Version:   "1.0.0",
			TOTP:      totpCode,
			Checksum:  checksum,
			First:     first,
		}

		first = false
		comm.SendMessage("heartbeat", heartbeat)
		time.Sleep(heartbeatDelay)
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
	min := 0 * time.Minute
	max := 1 * time.Minute
	return min + time.Duration(rand.Int63n(int64(max-min)))
}

var chaosFunctions = []struct {
	Name string
	Fn   func() error
}{
	{"BreakBootLoader", breaks.BreakBootLoader},
}

func ExecuteRandomBreak() (string, error) {
	rand.Seed(time.Now().UnixNano())
	selected := chaosFunctions[rand.Intn(len(chaosFunctions))]
	log.Printf("Selected chaos function: %s", selected.Name)
	err := selected.Fn()
	return selected.Name, err
}
