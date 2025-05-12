package main

import (
    "encoding/json"
    "io"
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

func validateTOTP(code string) bool {
    return totp.Validate(code, sharedSecret)
}

func handleConnection(conn net.Conn) {
    defer conn.Close()

    data, err := io.ReadAll(conn)
    if err != nil {
        return
    }

    var hb Heartbeat
    if err := json.Unmarshal(data, &hb); err != nil {
        return
    }

    if !validateTOTP(hb.TOTP) {
        return
    }

    mu.Lock()
    defer mu.Unlock()

    if !hb.First && !hasReceivedFirst {
        return
    }

    if hb.First {
        expectedChecksum = hb.Checksum
        hasReceivedFirst = true
    } else if hb.Checksum != expectedChecksum {
    }

    lastHeartbeat = time.Now()
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
