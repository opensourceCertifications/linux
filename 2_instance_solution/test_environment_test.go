package main

import (
    "encoding/json"
    "os"
    "testing"
    "time"

    "github.com/pquerna/otp/totp"
)

// TestComputeChecksum checks that ComputeChecksum returns consistent results
func TestComputeChecksum(t *testing.T) {
    // Create a temporary file
    tmpfile, err := os.CreateTemp("", "example.txt")
    if err != nil {
        t.Fatal(err)
    }
    defer os.Remove(tmpfile.Name()) // clean up

    content := []byte("This is some test data.\n")
    if _, err := tmpfile.Write(content); err != nil {
        t.Fatal(err)
    }
    tmpfile.Close()

    checksum1, err := ComputeChecksum(tmpfile.Name())
    if err != nil {
        t.Fatalf("Failed to compute checksum: %v", err)
    }

    checksum2, err := ComputeChecksum(tmpfile.Name())
    if err != nil {
        t.Fatalf("Failed to compute checksum: %v", err)
    }

    if checksum1 != checksum2 {
        t.Errorf("Checksums should match: got %s and %s", checksum1, checksum2)
    }
}

// TestHeartbeatMarshalling checks that Heartbeat can be marshaled and unmarshaled correctly
func TestHeartbeatMarshalling(t *testing.T) {
    heartbeat := Heartbeat{
        Timestamp: time.Now().UTC().Format(time.RFC3339),
        Status:    "alive",
        Service:   "test_environment",
        Version:   "1.0.0",
        TOTP:      "123456",
        Checksum:  "abcdef",
        First:     true,
    }

    data, err := json.Marshal(heartbeat)
    if err != nil {
        t.Fatalf("Failed to marshal heartbeat: %v", err)
    }

    var hb Heartbeat
    if err := json.Unmarshal(data, &hb); err != nil {
        t.Fatalf("Failed to unmarshal heartbeat: %v", err)
    }

    if hb.Service != "test_environment" {
        t.Errorf("Unexpected service field: got %s, want %s", hb.Service, "test_environment")
    }
}

// TestGenerateTOTP checks that TOTP codes can be generated
func TestGenerateTOTP(t *testing.T) {
    code, err := totp.GenerateCode(sharedSecret, time.Now())
    if err != nil {
        t.Fatalf("Failed to generate TOTP code: %v", err)
    }

    if len(code) != 6 {
        t.Errorf("Expected TOTP code length of 6, got %d", len(code))
    }
}
