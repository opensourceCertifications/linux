package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/opensourceCertifications/linux/shared/types"
	"github.com/opensourceCertifications/linux/shared/library"
)

func sendOperationComplete(ip string, port int, token string) error {
	// Prepare the JSON payload using the shared ChaosMessage type
	msg := types.ChaosMessage{
		Status:  "operation_complete",
		Message: "done",
		Token:   token,
	}

	// Marshal the message to JSON
	jsonData, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %v", err)
	}

	// Use the SendRawMessage function from the library to send the message
	err = library.SendRawMessage(ip, port, string(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send message: %v", err)
	}

	fmt.Println("✅ Operation complete message sent successfully!")
	return nil
}

func main() {
	// Ensure correct number of arguments (IP, Port, Token)
	if len(os.Args) != 4 {
		fmt.Println("Usage: go run send_operation_complete.go <IP> <Port> <Token>")
		os.Exit(1)
	}

	// Parse IP, port, and token from command-line arguments
	ip := os.Args[1]
	port := os.Args[2]
	token := os.Args[3]

	// Convert port from string to integer
	var portInt int
	_, err := fmt.Sscanf(port, "%d", &portInt)
	if err != nil || portInt == 0 {
		fmt.Println("Error: Invalid port number")
		os.Exit(1)
	}

	// Send the operation complete message
	err = sendOperationComplete(ip, portInt, token)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

