// Package monitor implements functions to set up nftables rules
package monitor

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

const ntpNatTable = "ntp_nat"

func runNft(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "nft", args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	return out.String(), err
}

func isIgnorableNftError(output string) bool {
	o := strings.ToLower(output)
	return strings.Contains(o, "file exists") ||
		strings.Contains(o, "exists") ||
		strings.Contains(o, "no such file") ||
		strings.Contains(o, "not found")
}

func runNftOK(ctx context.Context, errMsg string, args ...string) error {
	out, err := runNft(ctx, args...)
	if err != nil && !isIgnorableNftError(out) {
		return fmt.Errorf("%s: %v: %s", errMsg, err, out)
	}
	return nil
}

func ensureTable(ctx context.Context) error {
	return runNftOK(ctx, "nft add table",
		"add", "table", "ip", ntpNatTable,
	)
}

func ensurePreroutingChain(ctx context.Context) error {
	// Try the split-token version first (some people prefer it)
	out, err := runNft(ctx, "add", "chain", "ip", ntpNatTable, "prerouting",
		"{", "type", "nat", "hook", "prerouting", "priority", "-100", ";", "}")
	if err == nil || isIgnorableNftError(out) {
		return nil
	}

	// Fallback: single-string braces (some nft builds parse this better)
	out2, err2 := runNft(ctx, "add", "chain", "ip", ntpNatTable, "prerouting",
		"{ type nat hook prerouting priority -100; }")
	if err2 != nil && !isIgnorableNftError(out2) {
		return fmt.Errorf("nft add chain prerouting: %v: %s", err2, out2)
	}
	return nil
}

func ensurePostroutingChain(ctx context.Context) error {
	return runNftOK(ctx, "nft add chain postrouting",
		"add", "chain", "ip", ntpNatTable, "postrouting",
		"{ type nat hook postrouting priority srcnat; }",
	)
}

func addPreroutingRule(ctx context.Context, monitorIP, clientIP string, fakePort int) error {
	args := []string{"add", "rule", "ip", ntpNatTable, "prerouting"}
	if clientIP != "" {
		args = append(args, "ip", "saddr", clientIP)
	}
	args = append(args,
		"ip", "daddr", monitorIP,
		"udp", "dport", "123",
		"redirect", "to", fmt.Sprintf(":%d", fakePort),
	)
	return runNftOK(ctx, "nft add prerouting rule", args...)
}

func addPostroutingRule(ctx context.Context, monitorIP, clientIP string, fakePort int) error {
	args := []string{"add", "rule", "ip", ntpNatTable, "postrouting",
		"ip", "saddr", monitorIP,
	}
	if clientIP != "" {
		args = append(args, "ip", "daddr", clientIP)
	}
	args = append(args,
		"udp", "sport", fmt.Sprintf("%d", fakePort),
		"snat", "to", fmt.Sprintf("%s:123", monitorIP),
	)
	return runNftOK(ctx, "nft add postrouting rule", args...)
}

// SetupNTPRedirect installs nft rules on the monitor VM so that NTP queries to :123/udp
// are redirected to fakePort (e.g. 9123), and replies are SNAT'd back to source port 123.
//
// - monitorIP: the monitor VM IP that clients query (e.g. "192.168.56.10")
// - clientIP: the testenv IP to target (e.g. "192.168.56.11") (use "" to target any client)
// - fakePort: the port your fake server listens on (e.g. 9123)
func SetupNTPRedirect(ctx context.Context, monitorIP string, clientIP string, fakePort int) error {
	if err := ensureTable(ctx); err != nil {
		return err
	}
	if err := ensurePreroutingChain(ctx); err != nil {
		return err
	}
	if err := ensurePostroutingChain(ctx); err != nil {
		return err
	}
	if err := addPreroutingRule(ctx, monitorIP, clientIP, fakePort); err != nil {
		return err
	}
	if err := addPostroutingRule(ctx, monitorIP, clientIP, fakePort); err != nil {
		return err
	}
	return nil
}

// TeardownNTPRedirect removes the nftables table used for NTP redirection.
func TeardownNTPRedirect(ctx context.Context) error {
	return runNftOK(ctx, "nft delete table",
		"delete", "table", "ip", ntpNatTable,
	)
}
