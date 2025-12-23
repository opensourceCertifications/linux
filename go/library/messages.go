package library

import (
	datatypes "chaos-agent/library/types"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/url"
	"strings"

	"golang.org/x/crypto/nacl/box"
)

// EncryptMessage encrypts the message using the receiver's NaCl box PUBLIC key (base64).
// NOTE: encryptionKey is now expected to be base64(pubKey[32]).
// The receiver must decrypt with box.OpenAnonymous using its keypair.
func EncryptMessage(message string, encryptionKey string) ([]byte, error) {
	pubKeyB64 := encryptionKey

	if strings.Contains(pubKeyB64, "%") {
		if s, err := url.QueryUnescape(pubKeyB64); err == nil {
			pubKeyB64 = s
		}
	}

	pubBytes, err := base64.StdEncoding.DecodeString(pubKeyB64)
	if err != nil {
		return nil, fmt.Errorf("base64 decode public key: %w", err)
	}
	if len(pubBytes) != 32 {
		return nil, fmt.Errorf("public key must be 32 bytes, got %d", len(pubBytes))
	}

	var pubKey [32]byte
	copy(pubKey[:], pubBytes)

	ciphertext, err := box.SealAnonymous(nil, []byte(message), &pubKey, rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("seal anonymous: %w", err)
	}

	line := base64.StdEncoding.EncodeToString(ciphertext) + "\n"
	return []byte(line), nil
}

// SendRawMessage sends an encrypted message to the specified IP and port using a TCP connection
func SendRawMessage(ip string, port int, message string, encryptionKey string) error {
	encryptedMessage, err := EncryptMessage(message, encryptionKey)
	if err != nil {
		return fmt.Errorf("failed to encrypt message: %v", err)
	}

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

	// 4-byte length prefix
	// #nosec G115
	msgLen := uint32(len(encryptedMessage))
	lenBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(lenBuf, msgLen)

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

// GenerateToken creates a random hexadecimal token of specified byte length
func GenerateToken(nBytes int) (string, error) {
	b := make([]byte, nBytes)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// SendMessage prepares and sends a structured message to the monitoring server
func SendMessage(ip string, port int, status string, message string, token string, encryptionKey string) {
	msg := datatypes.ChaosMessage{
		Status:  status,
		Message: message,
		Token:   token,
	}

	jsonData, err := json.Marshal(msg)
	if err != nil {
		fmt.Printf("❌ Failed to marshal JSON: %v\n", err)
		return
	}

	err = SendRawMessage(ip, port, string(jsonData), encryptionKey)
	if err != nil {
		fmt.Printf("❌ Failed to send message: %v\n", err)
		return
	}
}
