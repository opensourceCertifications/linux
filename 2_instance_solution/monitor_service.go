/* monitor_service.go */

package main

import (
    "fmt"
    "io"
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
    fmt.Printf("%s\n", string(data))
}
