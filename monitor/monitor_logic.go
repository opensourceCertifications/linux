package main

import (
	"bufio"
	"encoding/json"
	"log"
	"net"
	"sync"
	"time"
	"os"

	"github.com/pquerna/otp/totp"
	"github.com/opensourceCertifications/linux/shared/types"
)

var (
	lastHeartbeat time.Time
	mu			 sync.Mutex
	sharedSecret   = "JBSWY3DPEHPK3PXP" // Predefined TOTP secret for testing
	expectedChecksum string
	hasReceivedFirst bool
)

func validateTOTP(code string) bool {
	return totp.Validate(code, sharedSecret)
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	scanner := bufio.NewScanner(conn)

	for scanner.Scan() {
		line := scanner.Bytes()

		var envelope types.Envelope
		if err := json.Unmarshal(line, &envelope); err != nil {
			log.Printf("Failed to parse envelope: %v\nPayload: %s\n", err, string(line))
			continue
		}

		switch envelope.Type {
		case "heartbeat":
			var hb types.Heartbeat
			if err := json.Unmarshal(envelope.Data, &hb); err != nil {
				log.Printf("Invalid heartbeat payload: %v\n", err)
				continue
			}

			if !validateTOTP(hb.TOTP) {
				log.Println("Invalid TOTP")
				continue
			}

			mu.Lock()
			if !hb.First && !hasReceivedFirst {
				mu.Unlock()
				continue
			}

			if hb.First {
				expectedChecksum = hb.Checksum
				hasReceivedFirst = true
			} else if hb.Checksum != expectedChecksum {
				log.Println("Checksum mismatch detected!")
			}

			lastHeartbeat = time.Now()
			mu.Unlock()

		case "chaos_report":
			var report types.ChaosReport
			if err := json.Unmarshal(envelope.Data, &report); err != nil {
				log.Printf("Invalid chaos report payload: %v\n", err)
				continue
			}

			if report.Timestamp == "" || report.Agent == "" || report.Action == "" {
				log.Println("Invalid chaos report: missing required fields")
				continue
			}

			if report.Timestamp == "" || report.Agent == "" || report.Action == "" {
			log.Printf("[CHAOS REPORT] %s by %s: %s\n", report.Timestamp, report.Agent, report.Action)
			saveReport(report)
		}

		default:
			if json.Valid(envelope.Data) {
				compact, err := json.Marshal(envelope)
				if err != nil {
					log.Printf("Failed to encode envelope for type %q: %v", envelope.Type, err)
					break
				}
				log.Printf("%s", compact)
			} else {
				log.Printf("Unknown or malformed message type %q:\n%s",
					envelope.Type, string(envelope.Data))
			}
		}
	}
}

func checkHeartbeat() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		mu.Lock()
		since := time.Since(lastHeartbeat)
		mu.Unlock()

		if since > 1*time.Second {
			println("No heartbeat received in the last second")
		}
	}
}

func saveReport(report types.ChaosReport) {
	path := os.Getenv("CHAOS_REPORT_LOG")
	if path == "" {
		path = "/tmp/chaos_reports.log"
	}

	f, err := os.OpenFile("chaos_reports.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("Failed to open chaos report log: %v", err)
		return
	}
	defer f.Close()

	data, err := json.Marshal(report)
	if err != nil {
		log.Printf("Failed to marshal chaos report: %v", err)
		return
	}

	f.WriteString(string(data) + "\n")
}

