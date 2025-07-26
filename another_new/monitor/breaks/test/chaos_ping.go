package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"time"
	"github.com/opensourceCertifications/linux/shared/types"
)

// These will be injected via -ldflags
var MonitorIP string
var MonitorPort string
var Token string

/*
type ChaosMessage struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Token   string `json:"token"`
}
*/

func main() {
	if MonitorIP == "" || MonitorPort == "" || Token == "" {
		fmt.Fprintln(os.Stderr, "Error: MonitorIP, MonitorPort, or Token is not set (use -ldflags)")
		os.Exit(1)
	}

	addr := fmt.Sprintf("%s:%s", MonitorIP, MonitorPort)
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to connect to monitor at %s: %v\n", addr, err)
		os.Exit(1)
	}
	defer conn.Close()

	sendLog(conn, "info", "chaos started")
	time.Sleep(1 * time.Second)
	sendLog(conn, "operation_complete", "done")
}

func sendLog(conn net.Conn, status, message string) {
	msg := types.ChaosMessage{
		Status:  status,
		Message: message,
		Token:   Token,
	}
	data, err := json.Marshal(msg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to encode JSON: %v\n", err)
		return
	}
	fmt.Fprintf(conn, "%s\n", data)
}
