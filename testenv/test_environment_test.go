package main

import (
	"crypto/sha256"
	"encoding/json"
	"io"
	"os"
	"testing"
	"time"

	"github.com/pquerna/otp/totp"
	"github.com/opensourceCertifications/linux/shared/types"
)

// ---------------- Core Utility Tests ----------------

func TestComputeChecksum(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "example.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	content := []byte("test content")
	if _, err := tmpfile.Write(content); err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()

	expected := sha256.Sum256(content)
	actual, err := ComputeChecksum(tmpfile.Name())
	if err != nil {
		t.Fatalf("Failed to compute checksum: %v", err)
	}

	expHex := fmt.Sprintf("%x", expected)
	if actual != expHex {
		t.Errorf("Expected checksum %s, got %s", expHex, actual)
	}
}

func TestComputeChecksumSelf(t *testing.T) {
	checksum, err := ComputeChecksumSelf()
	if err != nil {
		t.Fatalf("Failed to compute self checksum: %v", err)
	}
	if len(checksum) == 0 {
		t.Error("Checksum should not be empty")
	}
}

// ---------------- Heartbeat Tests ----------------

func TestHeartbeatSerialization(t *testing.T) {
	hb := types.Heartbeat{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Status:    "alive",
		Service:   "test_environment",
		Version:   "1.0.0",
		TOTP:      "123456",
		Checksum:  "abcdef",
		First:     true,
	}

	data, err := json.Marshal(hb)
	if err != nil {
		t.Fatalf("Failed to marshal heartbeat: %v", err)
	}

	var parsed types.Heartbeat
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal heartbeat: %v", err)
	}

	if parsed.Service != hb.Service || parsed.TOTP != hb.TOTP {
		t.Errorf("Unmarshalled heartbeat does not match original")
	}
}

func TestTOTPGeneration(t *testing.T) {
	code, err := totp.GenerateCode(sharedSecret, time.Now())
	if err != nil {
		t.Fatalf("TOTP generation failed: %v", err)
	}
	if len(code) != 6 {
		t.Errorf("Expected TOTP code of length 6, got %d", len(code))
	}
}

// ---------------- Randomness & Chaos ----------------

func TestGetRandomDurationBounds(t *testing.T) {
	for i := 0; i < 10; i++ {
		d := GetRandomDuration()
		if d < 0 || d > time.Minute {
			t.Errorf("Random duration out of expected bounds: %v", d)
		}
	}
}

// Note: This test requires at least one registered break to work
func TestExecuteRandomBreakReturnsName(t *testing.T) {
	// If there are no breaks registered, skip the test
	if len(registry.All()) == 0 {
		t.Skip("No registered chaos functions to test")
	}

	name, err := ExecuteRandomBreak()
	if err != nil {
		t.Fatalf("ExecuteRandomBreak failed: %v", err)
	}
	if name == "" {
		t.Error("Expected non-empty break name")
	}
}
