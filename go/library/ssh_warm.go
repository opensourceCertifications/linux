// library/ssh_warm.go
package library

import (
	"fmt"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"

	"chaos-agent/types"
)

// global warm client + mutex to protect it
var (
	warmClientMu sync.RWMutex
	warmClient   *ssh.Client
)

// SetupWarmSSH establishes a long-lived SSH connection to the testenv
// described by probe, and starts a keepalive loop in the background.
//
// - auth: SSH auth methods (e.g. public key, password)
// - hostKeyCallback: host key verification callback
func SetupWarmSSH(probe types.Probe, auth []ssh.AuthMethod, hostKeyCallback ssh.HostKeyCallback) error {
	cfg := &ssh.ClientConfig{
		User:            probe.User,
		Auth:            auth,
		HostKeyCallback: hostKeyCallback,
		Timeout:         probe.Timeout,
	}

	// Dial the SSH server
	client, err := ssh.Dial("tcp", probe.Addr, cfg)
	if err != nil {
		return fmt.Errorf("failed to dial SSH: %w", err)
	}

	// Store as the warm client
	warmClientMu.Lock()
	warmClient = client
	warmClientMu.Unlock()

	// Start keepalive goroutine
	go keepAliveLoop(client)

	return nil
}

// keepAliveLoop sends periodic keepalive requests while the client is alive.
func keepAliveLoop(client *ssh.Client) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		// If client is closed, SendRequest will error and we can stop.
		_, _, err := client.SendRequest("keepalive@openssh.com", true, nil)
		if err != nil {
			// Connection is likely dead; stop the loop.
			return
		}
	}
}

// GetWarmClient returns the current warm SSH client, or nil if not set.
func GetWarmClient() *ssh.Client {
	warmClientMu.RLock()
	defer warmClientMu.RUnlock()
	return warmClient
}
