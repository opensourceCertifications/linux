package main

import (
    "encoding/json"
    "os"
    "path/filepath"
    "testing"
    "time"

    "github.com/pquerna/otp/totp"
)

// ---------------- Basic Functionality Tests ----------------

func TestComputeChecksum(t *testing.T) {
    tmpfile, err := os.CreateTemp("", "example.txt")
    if err != nil {
        t.Fatal(err)
    }
    defer os.Remove(tmpfile.Name())

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

func TestGenerateTOTP(t *testing.T) {
    code, err := totp.GenerateCode(sharedSecret, time.Now())
    if err != nil {
        t.Fatalf("Failed to generate TOTP code: %v", err)
    }

    if len(code) != 6 {
        t.Errorf("Expected TOTP code length of 6, got %d", len(code))
    }
}

// ---------------- Chaos Logic Tests ----------------

func TestGetRandomDuration(t *testing.T) {
    duration := GetRandomDuration()
    if duration < 5*time.Minute || duration > 7*time.Minute {
        t.Errorf("Duration out of range: got %v", duration)
    }
}

func TestDirectoriesExist(t *testing.T) {
    dirs := []string{"./breaks/breaks/", "./breaks/checks/"}

    for _, dir := range dirs {
        info, err := os.Stat(dir)
        if os.IsNotExist(err) {
            t.Errorf("Directory does not exist: %s", dir)
        } else if err != nil {
            t.Errorf("Error checking directory %s: %v", dir, err)
        } else if !info.IsDir() {
            t.Errorf("Path exists but is not a directory: %s", dir)
        }
    }
}

func TestMatchingBreakAndCheckFiles(t *testing.T) {
    breakFiles, err := os.ReadDir("./breaks/breaks/")
    if err != nil {
        t.Fatalf("Failed reading breaks directory: %v", err)
    }

    checkFiles, err := os.ReadDir("./breaks/checks/")
    if err != nil {
        t.Fatalf("Failed reading checks directory: %v", err)
    }

    checkSet := make(map[string]bool)
    for _, file := range checkFiles {
        if filepath.Ext(file.Name()) == ".go" {
            checkSet[file.Name()] = true
        }
    }

    for _, file := range breakFiles {
        if filepath.Ext(file.Name()) == ".go" {
            if !checkSet[file.Name()] {
                t.Errorf("No matching check file for break file: %s", file.Name())
            }
        }
    }
}

func TestExecuteRandomBreak(t *testing.T) {
    name, err := ExecuteRandomBreak("./breaks/breaks/")
    if err != nil {
        t.Fatalf("Failed to execute random break: %v", err)
    }
    if name == "" {
        t.Error("Break name should not be empty")
    }
}

func TestMonitorReceivesBreak(t *testing.T) {
    sentBreak := "example_break.go"
    receivedBreak := simulateMonitorReception(sentBreak)

    if receivedBreak != sentBreak {
        t.Errorf("Monitor did not receive correct break script name: got %s, want %s", receivedBreak, sentBreak)
    }
}
