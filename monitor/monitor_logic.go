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
	"shared/types"
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

var (
    lastHeartbeat time.Time
    mu             sync.Mutex
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
			log.Printf("[CHAOS REPORT] %s by %s: %s\n", report.Timestamp, report.Agent, report.Action)
			saveReport(report)

		default:
			log.Printf("Unknown message type: %s", envelope)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Error reading from connection: %v", err)
	}
}

//func handleConnection(conn net.Conn) {
//	defer conn.Close()
//
//	data, err := io.ReadAll(conn)
//	if err != nil {
//		return
//	}
//
//	var envelope types.Envelope
//	if err := json.Unmarshal(data, &envelope); err != nil {
//		fmt.Println("Failed to parse envelope:", err)
//		return
//	}
//
//	switch envelope.Type {
//	case "heartbeat":
//		var hb types.Heartbeat
//		if err := json.Unmarshal(envelope.Data, &hb); err != nil {
//			fmt.Println("Invalid heartbeat:", err)
//			return
//		}
//
//		if !validateTOTP(hb.TOTP) {
//			fmt.Println("Invalid TOTP")
//			return
//		}
//
//		mu.Lock()
//		defer mu.Unlock()
//
//		if !hb.First && !hasReceivedFirst {
//			return
//		}
//
//		if hb.First {
//			expectedChecksum = hb.Checksum
//			hasReceivedFirst = true
//		} else if hb.Checksum != expectedChecksum {
//			// checksum mismatch logic here
//		}
//
//		lastHeartbeat = time.Now()
//
//	case "chaos_report":
//		var report types.ChaosReport
//		if err := json.Unmarshal(envelope.Data, &report); err != nil {
//			fmt.Println("Invalid chaos report:", err)
//			return
//		}
//		//fmt.Printf("[CHAOS REPORT] %s by %s: %s\n", report.Timestamp, report.Agent, report.Action)
//		log.Printf("[CHAOS REPORT] %s by %s: %s\n", report.Timestamp, report.Agent, report.Action)
//
//		saveReport(report)
//
//
//	default:
//		fmt.Println("Unknown message type:", envelope)
//	}
//}


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

