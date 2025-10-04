package library

import (
	datatypes "chaos-agent/shared/types"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net"
)

// EncryptMessage encrypts the message using AES-GCM with the provided encryption key
func EncryptMessage(message string, encryptionKey string) ([]byte, error) {
	// Convert the hex-encoded encryption key to bytes
	key, err := hex.DecodeString(encryptionKey)
	fmt.Printf("🔑 Encryption key (hex): %s\n", encryptionKey)
	fmt.Printf("🔒 Message to encrypt: %s\n", message)
	fmt.Printf("key: %s\n", key)
	if err != nil {
		return nil, err
	}

	// Ensure the key is the correct length for AES (32 bytes for AES-256)
	if len(key) != 32 {
		return nil, err
	}

	// Create a new AES cipher block from the key
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	// Create a nonce (12 random bytes) for AES-GCM
	nonce := make([]byte, 12)
	_, err = rand.Read(nonce)
	fmt.Printf("🔑 Nonce (before random generation): %x\n", nonce)
	if err != nil {
		return nil, err
	}

	// Create an AES-GCM cipher instance
	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// Encrypt the message using AES-GCM
	ciphertext := aesgcm.Seal(nil, nonce, []byte(message), nil)

	// Prepend the nonce to the ciphertext into a new buffer
	encryptedMessage := make([]byte, 0, len(nonce)+len(ciphertext))
	encryptedMessage = append(encryptedMessage, nonce...)
	encryptedMessage = append(encryptedMessage, ciphertext...)

	fmt.Printf("🔒 Encrypted message (hex): %s\n", hex.EncodeToString(encryptedMessage))

	// Log the nonce and ciphertext for debugging
	fmt.Printf("🔑 Nonce: %x\n", nonce)
	fmt.Printf("🔒 Ciphertext: %x\n", ciphertext)

	return encryptedMessage, nil
}

// SendRawMessage sends an encrypted message to the specified IP and port using a TCP connection
func SendRawMessage(ip string, port int, message string, encryptionKey string) error {
	// Encrypt the message before sending it
	fmt.Printf("🔒 Encrypting message: %s\n", message)
	encryptedMessage, err := EncryptMessage(message, encryptionKey)
	if err != nil {
		return fmt.Errorf("failed to encrypt message: %v", err)
	}

	// Prepare the address
	addr := fmt.Sprintf("%s:%d", ip, port)
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %v", addr, err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			log.Printf("conn close ssh client: %v", err)
		}
	}()

	// 👇 Add 4-byte length prefix
	// #nosec G115 // No point limiting message size here, will be limited by server anyway
	msgLen := uint32(len(encryptedMessage))
	lenBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(lenBuf, msgLen)

	// 👇 Write length first, then the message
	_, err = conn.Write(lenBuf)
	if err != nil {
		return fmt.Errorf("failed to send length prefix: %v", err)
	}

	_, err = conn.Write(encryptedMessage)
	if err != nil {
		return fmt.Errorf("failed to send message: %v", err)
	}

	return nil
}

// SendMessage prepares and sends a structured message to the monitoring server
func SendMessage(ip string, port int, status string, message string, token string, encryptionKey string) {
	fmt.Printf("🔒 Preparing to send message to %s:%d\n", ip, port)
	// Prepare the JSON payload using the shared ChaosMessage type
	msg := datatypes.ChaosMessage{
		Status:  status,
		Message: message,
		Token:   token,
	}

	// Marshal the message to JSON
	jsonData, err := json.Marshal(msg)
	if err != nil {
		fmt.Printf("❌ Failed to marshal JSON: %v\n", err)
		return
	}

	// Use the SendRawMessage function from the library to send the encrypted message
	fmt.Printf("🔒 Sending message")
	err = SendRawMessage(ip, port, string(jsonData), encryptionKey)
	if err != nil {
		fmt.Printf("❌ Failed to send message: %v\n", err)
		return
	}
}
