package main

import (
    "encoding/json"
    "fmt"
    "log"
    "net"
    "os"
    "time"

    "github.com/pquerna/otp/totp"
)

type Heartbeat struct {
    Timestamp string `json:"timestamp"`
    Status    string `json:"status"`
    Service   string `json:"service"`
    Version   string `json:"version"`
    TOTP      string `json:"totp"`
}

const sharedSecret = "JBSWY3DPEHPK3PXP"

func main() {
    if len(os.Args) != 2 {
        log.Fatalf("Usage: %s ", os.Args[0])
    }

    ip := os.Args[1]
    address := fmt.Sprintf("%s:9000", ip)

    for {
        totpCode, err := totp.GenerateCode(sharedSecret, time.Now())
        if err != nil {
            log.Printf("Failed to generate TOTP: %v", err)
            continue
        }

        heartbeat := Heartbeat{
            Timestamp: time.Now().UTC().Format(time.RFC3339),
            Status:    "alive",
            Service:   "test_environment",
            Version:   "1.0.0",
            TOTP:      totpCode,
        }

        data, err := json.Marshal(heartbeat)
        if err != nil {
            log.Printf("Failed to marshal heartbeat: %v", err)
            continue
        }

        sendMessage(address, data)
        time.Sleep(500 * time.Millisecond)
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
        return
    }

//    fmt.Printf("Sent heartbeat to %s: %s\n", address, string(data))
}
