// library/ssh_warm.go
package library

import (
	"fmt"
	"sync"
	"time"
	"net"

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
func SetupWarmSSH(probe types.Probe, auth []ssh.AuthMethod, hostKeyCallback ssh.HostKeyCallback) error {
    // Wrap the host key callback so that we effectively say "yes" even when the
    // host is unknown or the callback returns an error.
    //
    // - If hostKeyCallback == nil, we use ssh.InsecureIgnoreHostKey().
    // - If it's non-nil, we call it and ignore any error (like StrictHostKeyChecking=no).
    var cb ssh.HostKeyCallback
    if hostKeyCallback == nil {
        cb = ssh.InsecureIgnoreHostKey()
    } else {
        cb = func(hostname string, remote net.Addr, key ssh.PublicKey) error {
            if err := hostKeyCallback(hostname, remote, key); err != nil {
                // TODO: optionally log err here if you want visibility
                // "Auto-yes": ignore the error and accept the host key anyway.
                return nil
            }
            return nil
        }
    }

    cfg := &ssh.ClientConfig{
        User:            probe.User,
        Auth:            auth,
        HostKeyCallback: cb,
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
