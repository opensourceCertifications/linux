package main

import (
    "fmt"
    "log"
    "net"
    "os"
    "time"
)

func main() {
    port := os.Getenv("MONITOR_PORT")
    if port == "" {
        port = "9000" // Default
    }
    listener, err := net.Listen("tcp", ":"+port)
//    listener, err := net.Listen("tcp", ":9000")
    if err != nil {
        log.Fatalf("Failed to start listener: %v", err)
    }
    defer listener.Close()
    fmt.Println("Monitor listening on port %v...", port)

//    go checkHeartbeat()
    go func() {
        for {
            defer func() {
                if r := recover(); r != nil {
                    log.Printf("checkHeartbeat panicked: %v", r)
                }
            }()
            checkHeartbeat()
            time.Sleep(1 * time.Second)
        }
    }()

    for {
        conn, err := listener.Accept()
        if err != nil {
            log.Printf("Failed to accept connection: %v", err)
            continue
        }
        go handleConnection(conn)
    }
}
