/* test_environment.go */

package main

import (
    "fmt"
    "log"
    "net"
    "os"
)

func main() {
    if len(os.Args) != 3 {
        log.Fatalf("Usage: %s <ip> <message>", os.Args[0])
    }

    ip := os.Args[1]
    message := os.Args[2]
    address := fmt.Sprintf("%s:9000", ip)

    conn, err := net.Dial("tcp", address)
    if err != nil {
        log.Fatalf("Failed to connect to %s: %v", address, err)
    }
    defer conn.Close()

    _, err = conn.Write([]byte(message))
    if err != nil {
        log.Fatalf("Failed to send message: %v", err)
    }

    fmt.Printf("Sent message to %s: %s\n", ip, message)
}