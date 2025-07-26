package library

import (
	"fmt"
	"net"
)

func SendRawMessage(ip string, port int, message string) error {
	addr := fmt.Sprintf("%s:%d", ip, port)
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %v", addr, err)
	}
	defer conn.Close()

	_, err = fmt.Fprintf(conn, "%s\n", message)
	if err != nil {
		return fmt.Errorf("failed to send message: %v", err)
	}

	return nil
}
