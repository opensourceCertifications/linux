/* monitor_service.go */

package main

import (
    "encoding/json"
    "fmt"
    "io"
    "log"
    "net"
    "sync"
    "time"

    "github.com/pquerna/otp/totp"
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

func main() {
    listener, err := net.Listen("tcp", ":9000")
    if err != nil {
        log.Fatalf("Failed to start listener: %v", err)
    }
    defer listener.Close()
    fmt.Println("Monitor listening on port 9000...")

    go checkHeartbeat()

    for {
        conn, err := listener.Accept()
        if err != nil {
            log.Printf("Failed to accept connection: %v", err)
            continue
        }
        go handleConnection(conn)
    }
}

func handleConnection(conn net.Conn) {
    defer conn.Close()

    data, err := io.ReadAll(conn)
    if err != nil {
        log.Printf("Failed to read data: %v", err)
        return
    }

    var hb Heartbeat
    if err := json.Unmarshal(data, &hb); err != nil {
        log.Printf("Invalid heartbeat format: %v", err)
        return
    }

    if !validateTOTP(hb.TOTP) {
        log.Printf("ERROR: Invalid TOTP received: %s", hb.TOTP)
        return
    } else {
        log.Printf("Valid TOTP received: %s", hb.TOTP)
    }

    mu.Lock()
    defer mu.Unlock()

    // Check for first heartbeat and validate checksum
    if !hb.First && !hasReceivedFirst {
        log.Printf("ERROR: Received non-initial heartbeat before first was received")
        return
    }

    if hb.First {
        expectedChecksum = hb.Checksum
        hasReceivedFirst = true
        log.Printf("Initial checksum received: %s", expectedChecksum)
    } else if hb.Checksum != expectedChecksum {
        log.Printf("ERROR: Checksum mismatch. Received: %s, Expected: %s", hb.Checksum, expectedChecksum)
    } else {
        log.Printf("Valid heartbeat with matching checksum: %s", hb.Checksum)
    }

    lastHeartbeat = time.Now()
}


func validateTOTP(code string) bool {
    return totp.Validate(code, sharedSecret)
}

func checkHeartbeat() {
    ticker := time.NewTicker(1 * time.Second)
    defer ticker.Stop()

    for range ticker.C {
        mu.Lock()
        since := time.Since(lastHeartbeat)
        mu.Unlock()

        if since > 1*time.Second {
            log.Println("ERROR: Missed heartbeat")
        }
    }
}
