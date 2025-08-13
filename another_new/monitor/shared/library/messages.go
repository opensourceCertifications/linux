package library

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"encoding/binary"
)

// EncryptMessage encrypts the message using AES-GCM with the provided encryption key
func EncryptMessage(message string, encryptionKey string) ([]byte, error) {
	// Convert the hex-encoded encryption key to bytes
	key, err := hex.DecodeString(encryptionKey)
	fmt.Printf("🔑 Encryption key (hex): %s\n", encryptionKey)
	fmt.Printf("🔒 Message to encrypt: %s\n", message)
	fmt.Printf("key: %s\n", key)
	if err != nil {
		return nil, fmt.Errorf("failed to decode encryption key: %v", err)
	}

	// Ensure the key is the correct length for AES (32 bytes for AES-256)
	if len(key) != 32 {
		return nil, fmt.Errorf("invalid encryption key length: %d bytes (expected 32 bytes)", len(key))
	}

	// Create a new AES cipher block from the key
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %v", err)
	}

	// Create a nonce (12 random bytes) for AES-GCM
	nonce := make([]byte, 12)
	_, err = rand.Read(nonce)
	fmt.Printf("🔑 Nonce (before random generation): %x\n", nonce)
	//fmt.Printf("🔑 Generated nonce key of length %d bytes\n", keyLength)
	if err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %v", err)
	}

	// Create an AES-GCM cipher instance
	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES-GCM cipher: %v", err)
	}

	// Encrypt the message using AES-GCM
	ciphertext := aesgcm.Seal(nil, nonce, []byte(message), nil)

	// Prepend the nonce to the ciphertext (nonce is required for decryption)
	encryptedMessage := append(nonce, ciphertext...)
	fmt.Printf("🔒 Encrypted message (hex): %s\n", hex.EncodeToString(encryptedMessage))


	// Log the nonce and ciphertext for debugging
	fmt.Printf("🔑 Nonce: %x\n", nonce)
	fmt.Printf("🔒 Ciphertext: %x\n", ciphertext)

	return encryptedMessage, nil
}

func SendRawMessage(ip string, port int, message string, encryptionKey string) error {
	// Encrypt the message before sending it
	fmt.Printf("🔒 Encrypting message: %s\n", message)
	encryptedMessage, err := EncryptMessage(message, encryptionKey)
	if err != nil {
		return fmt.Errorf("failed to encrypt message: %v", err)
	}

	// Convert encrypted message to a string (hex) for easy transmission
	//encryptedMessageHex := hex.EncodeToString(encryptedMessage)

	// Prepare the address
	addr := fmt.Sprintf("%s:%d", ip, port)
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %v", addr, err)
	}
	defer conn.Close()

	// Send the encrypted message (hex-encoded)
	//_, err = fmt.Fprintf(conn, "%s\n", encryptedMessageHex)
	// 👇 Add 4-byte length prefix
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
