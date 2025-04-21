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
)

type Heartbeat struct {
    Timestamp string `json:"timestamp"`
    Status    string `json:"status"`
    Service   string `json:"service"`
    Version   string `json:"version"`
}

var (
    lastHeartbeat time.Time
    mu             sync.Mutex
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

    mu.Lock()
    lastHeartbeat = time.Now()
    mu.Unlock()
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

