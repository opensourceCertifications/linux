package main

import (
	//"encoding/json"
	"fmt"
	"os"

	//"github.com/opensourceCertifications/linux/shared/types"
	"github.com/opensourceCertifications/linux/shared/library"
)

func sendOperationComplete(ip string, port int, token string, encryptionKey string) error {
	// Prepare the JSON payload using the shared ChaosMessage type
	//msg := types.ChaosMessage{
	//	Status:  "operation_complete",
	//	Message: "done",
	//	Token:   token,
	//}
	fmt.Printf("I am about to send a general message")
	if err := library.SendMessage(ip, port, "operation_complete", "this is a test", token, encryptionKey); err != nil {
		fmt.Printf("Error sending message: %v\n", err)
		return err
	}

	// Marshal the message to JSON
	//jsonData, err := json.Marshal(msg)
	//if err != nil {
	//	return fmt.Errorf("failed to marshal JSON: %v", err)
	//}

	// Use the SendRawMessage function from the library to send the encrypted message
	//err = library.SendRawMessage(ip, port, string(jsonData), encryptionKey)
	//if err != nil {
	//	return fmt.Errorf("failed to send message: %v", err)
	//}

	fmt.Println("✅ Operation complete message sent successfully!")
	return nil
}

func main() {
	// Ensure correct number of arguments (IP, Port, Token, Encryption Key)
	if len(os.Args) != 5 {
		fmt.Println("Usage: go run send_operation_complete.go <IP> <Port> <Token> <EncryptionKey>")
		os.Exit(1)
	}

	// Parse IP, port, token, and encryption key from command-line arguments
	ip := os.Args[1]
	port := os.Args[2]
	token := os.Args[3]
	encryptionKey := os.Args[4] // The encryption key passed as a command-line argument

	// Convert port from string to integer
	var portInt int
	_, err := fmt.Sscanf(port, "%d", &portInt)
	if err != nil || portInt == 0 {
		fmt.Println("Error: Invalid port number")
		os.Exit(1)
	}

	// Send the operation complete message with encryption key
	err = sendOperationComplete(ip, portInt, token, encryptionKey)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
