package main

import (
	"fmt"

	"github.com/opensourceCertifications/linux/shared/library"
)

func main() {
	ip := "192.168.56.10"
	port := 42057
	message := `{"status":"foo","message":"done","token":"57bad567f8f9c476f7a77186eadfcd2f"}`

	err := library.SendRawMessage(ip, port, message)
	if err != nil {
		fmt.Println("❌ Failed to send message:", err)
	} else {
		fmt.Println("✅ Message sent successfully")
	}
}
