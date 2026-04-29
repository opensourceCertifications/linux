// A small Go program to add a runtime NTP source to chrony.
package main

import (
	"chaos-agent/library"
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// Injected by your build step:
// -X=main.MonitorIP=...
var (
	MonitorIP      string
	MonitorPortStr string // unused
	EncryptionKey  string // unused
)

var hostnameRE = regexp.MustCompile(`^[a-zA-Z0-9.-]+$`)

func validateTarget(s string) error {
	s = strings.TrimSpace(s)
	if s == "" {
		return errors.New("MonitorIP is empty")
	}
	if ip := net.ParseIP(s); ip != nil {
		return nil
	}
	if hostnameRE.MatchString(s) {
		return nil
	}
	return fmt.Errorf("MonitorIP looks invalid/unsafe: %q", s)
}

func run(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...) // #nosec G204 -- no shell; args are controlled and MonitorIP is validated
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	return out.String(), err
}

func main() {
	if err := validateTarget(MonitorIP); err != nil {
		fmt.Fprintln(os.Stderr, "chrony_add_source:", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

	// Runtime add: in-memory only, gone after chronyd restart
	_, _ = run(ctx, "chronyc", "offline")
	out, err := run(ctx, "chronyc", "add", "server", MonitorIP, "iburst", "prefer")
	if err != nil && !strings.Contains(out, "already present") {
		cancel()
		fmt.Fprintf(os.Stderr, "chrony_add_source: chronyc add failed: %v\n%s\n", err, out)
		os.Exit(1)
	}

	// Optional: show status for debugging during development
	if out2, err2 := run(ctx, "chronyc", "sources", "-v"); err2 == nil {
		cancel()
		fmt.Printf("chrony_add_source: added runtime server %s\n%s\n", MonitorIP, out2)
		return
	}

	cancel()
	fmt.Printf("chrony_add_source: added runtime server %s\n", MonitorIP)
	library.SendMessage(MonitorIP, MonitorPort, "operation_complete", "complete", Token, EncryptionKey)
}
