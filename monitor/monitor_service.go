package main

import (
    "fmt"
    "log"
    "net"
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
