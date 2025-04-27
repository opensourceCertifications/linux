package main

import (
    "encoding/json"
    "testing"
    "time"

    "github.com/pquerna/otp/totp"
)

// TestValidateTOTP_Valid tests that validateTOTP accepts correct codes
func TestValidateTOTP_Valid(t *testing.T) {
    code, err := totp.GenerateCode(sharedSecret, time.Now())
    if err != nil {
        t.Fatalf("Failed to generate TOTP: %v", err)
    }

    if !validateTOTP(code) {
        t.Errorf("validateTOTP rejected a valid TOTP code")
    }
}

// TestValidateTOTP_Invalid tests that validateTOTP rejects wrong codes
func TestValidateTOTP_Invalid(t *testing.T) {
    fakeCode := "000000"

    if validateTOTP(fakeCode) {
        t.Errorf("validateTOTP accepted an invalid TOTP code")
    }
}

// TestHeartbeatMarshalling checks that Heartbeat struct can be serialized/deserialized
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

// TestCheckHeartbeat simulates missed heartbeat detection
func TestCheckHeartbeat(t *testing.T) {
    lastHeartbeat = time.Now().Add(-2 * time.Second) // Simulate 2 seconds ago

    triggered := false

    // Simulate a short check loop
    ticker := time.NewTicker(100 * time.Millisecond)
    defer ticker.Stop()

    done := make(chan bool)

    go func() {
        for range ticker.C {
            mu.Lock()
            since := time.Since(lastHeartbeat)
            mu.Unlock()

            if since > 1*time.Second {
                triggered = true
                done <- true
                return
            }
        }
    }()

    select {
    case <-done:
        if !triggered {
            t.Errorf("Expected missed heartbeat detection")
        }
    case <-time.After(500 * time.Millisecond):
        t.Errorf("Heartbeat check did not trigger in expected time")
    }
}
