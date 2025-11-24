package library

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// GetAuditScope probes Linux kernel auditing via `auditctl`.
// It returns a JSON string describing enabled/immutable state, pid, backlog, lost events, failure mode,
// rate/backlog limits, and the current rule list. Partial results are returned on errors.
func GetAuditScope() string {
	scope := probeAudit()
	b, _ := json.Marshal(scope)
	return string(b)
}

type AuditScope struct {
	// High-level
	Enabled       bool   `json:"enabled"`         // kernel audit enabled (enabled=1 or immutable=2)
	Immutable     bool   `json:"immutable"`       // enabled==2 (cannot change rules until reboot)
	FailureMode   string `json:"failure_mode"`    // 0=silent,1=printk,2=panic (as reported)
	PID           int    `json:"pid"`             // kernel-audit owner PID (usually auditd)
	RateLimit     int    `json:"rate_limit"`      // messages/sec (0 = no limit)
	Backlog       int    `json:"backlog"`         // current queue depth
	BacklogLimit  int    `json:"backlog_limit"`   // max queue
	LostEvents    int    `json:"lost"`            // lost messages (kernel reported)
	FeaturesHex   string `json:"features_hex"`    // raw features bitmap if available
	BacklogWaitMS int    `json:"backlog_wait_ms"` // wait time if available (older kernels may omit)

	// Rules as raw strings (exactly what `auditctl -l` prints)
	Rules []string `json:"rules"`

	// Diagnostics
	HasAuditctl  bool   `json:"has_auditctl"`
	AuditctlPath string `json:"auditctl_path,omitempty"`
	Error        string `json:"error,omitempty"` // aggregated non-fatal errors
}

func probeAudit() AuditScope {
	var scope AuditScope
	var errs []string

	// Try to find auditctl in common places + PATH.
	auditctl, ok := findBinary("auditctl",
		"/usr/sbin/auditctl",
		"/sbin/auditctl",
		"/bin/auditctl",
		"/usr/bin/auditctl",
	)
	scope.HasAuditctl = ok
	if ok {
		scope.AuditctlPath = auditctl
	}

	// Always use a short timeout—this should never hang your chaos run.
	ctx, cancel := context.WithTimeout(context.Background(), 800*time.Millisecond)
	defer cancel()

	// 1) Status (auditctl -s)
	if ok {
		out, err := run(ctx, auditctl, "-s")
		if err != nil {
			errs = append(errs, fmt.Sprintf("auditctl -s: %v", err))
		} else if perr := parseAuditctlStatus(strings.TrimSpace(out), &scope); perr != nil {
			errs = append(errs, fmt.Sprintf("parse status: %v", perr))
		}
	} else {
		errs = append(errs, "auditctl not found; status unavailable")
	}

	// 2) Rules (auditctl -l)
	if ok {
		out, err := run(ctx, auditctl, "-l")
		if err != nil {
			// Not fatal; immutable rules may still list. Keep error.
			errs = append(errs, fmt.Sprintf("auditctl -l: %v", err))
		} else {
			scope.Rules = parseAuditctlRules(out)
		}
	}

	// 3) Synthesize booleans even if we only partially parsed.
	//    If "enabled" is missing, leave defaults (false).
	//    If flag==2, mark immutable.
	// (Already handled in parse; nothing extra needed here.)

	if len(errs) > 0 {
		scope.Error = strings.Join(errs, "; ")
	}
	return scope
}

func run(ctx context.Context, bin string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, bin, args...)
	// Do not inherit stdin; capture only stdout/stderr.
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	// Some distros put admin binaries in /usr/sbin not on non-root PATH.
	// Preserve PATH but also append common sbin dirs for child.
	cmd.Env = append(os.Environ(),
		"PATH="+augmentPATH(os.Getenv("PATH")),
	)
	err := cmd.Run()
	if ctx.Err() == context.DeadlineExceeded {
		return "", errors.New("timeout")
	}
	if err != nil {
		se := strings.TrimSpace(stderr.String())
		if se != "" {
			return "", fmt.Errorf("%w: %s", err, se)
		}
		return "", err
	}
	return stdout.String(), nil
}

func augmentPATH(p string) string {
	parts := []string{p, "/usr/sbin", "/sbin"}
	seen := map[string]bool{}
	var out []string
	for _, part := range parts {
		if part == "" {
			continue
		}
		if !seen[part] {
			out = append(out, part)
			seen[part] = true
		}
	}
	return strings.Join(out, string(os.PathListSeparator))
}

func findBinary(name string, extra ...string) (string, bool) {
	// Try PATH first
	if p, err := exec.LookPath(name); err == nil {
		return p, true
	}
	// Then common absolute locations
	for _, e := range extra {
		if e == "" {
			continue
		}
		if st, err := os.Stat(e); err == nil && !st.IsDir() {
			return e, true
		}
	}
	return "", false
}

// Example auditctl -s (varies slightly by kernel):
// "enabled 1\nfailure 1\npid 1234\nrate_limit 0\nbacklog_limit 8192\nlost 0\nbacklog 0\nbacklog_wait_time 60000\nfeatures 0x0000000f\n"
// Or new-style single-line: "AUDIT_STATUS: enabled=1 ..."
// Handle both.
func parseAuditctlStatus(s string, scope *AuditScope) error {
	// Normalize: split lines, gather key=value pairs.
	// Two formats:
	//  1) multiline "key value"
	//  2) one-line "AUDIT_STATUS: key=val key=val ..."
	s = strings.TrimSpace(s)
	if s == "" {
		return errors.New("empty status")
	}

	// New-style one-line?
	if strings.Contains(s, "AUDIT_STATUS:") || strings.Contains(s, "enabled=") {
		pairs := kvPairsFromOneLine(s)
		applyKVsToScope(pairs, scope)
		return nil
	}

	// Old-style multiline "key value"
	lines := strings.Split(s, "\n")
	pairs := map[string]string{}
	for _, ln := range lines {
		ln = strings.TrimSpace(ln)
		if ln == "" {
			continue
		}
		fields := strings.Fields(ln)
		if len(fields) >= 2 {
			k := strings.ToLower(fields[0])
			v := fields[1]
			pairs[k] = v
		}
	}
	applyKVsToScope(pairs, scope)
	return nil
}

var kvRe = regexp.MustCompile(`([a-zA-Z_]+)=([^\s]+)`)

func kvPairsFromOneLine(s string) map[string]string {
	pairs := map[string]string{}
	for _, m := range kvRe.FindAllStringSubmatch(s, -1) {
		if len(m) == 3 {
			pairs[strings.ToLower(m[1])] = m[2]
		}
	}
	return pairs
}

func atoiDefault(s string, def int) int {
	if s == "" {
		return def
	}
	var n int
	_, err := fmt.Sscanf(s, "%d", &n)
	if err != nil {
		return def
	}
	return n
}

func applyKVsToScope(kv map[string]string, scope *AuditScope) {
	// enabled: 0=disabled,1=enabled,2=immutable
	if v, ok := kv["enabled"]; ok {
		ev := atoiDefault(v, 0)
		scope.Enabled = ev == 1 || ev == 2
		scope.Immutable = ev == 2
	}
	if v, ok := kv["failure"]; ok {
		scope.FailureMode = v // keep raw 0/1/2; caller can map to meaning
	}
	if v, ok := kv["pid"]; ok {
		scope.PID = atoiDefault(v, 0)
	}
	if v, ok := kv["rate_limit"]; ok {
		scope.RateLimit = atoiDefault(v, 0)
	}
	if v, ok := kv["backlog_limit"]; ok {
		scope.BacklogLimit = atoiDefault(v, 0)
	}
	if v, ok := kv["backlog"]; ok {
		scope.Backlog = atoiDefault(v, 0)
	}
	if v, ok := kv["lost"]; ok {
		scope.LostEvents = atoiDefault(v, 0)
	}
	if v, ok := kv["backlog_wait_time"]; ok {
		scope.BacklogWaitMS = atoiDefault(v, 0)
	}
	if v, ok := kv["features"]; ok {
		scope.FeaturesHex = v
	}
}

func parseAuditctlRules(out string) []string {
	var rules []string
	for _, ln := range strings.Split(out, "\n") {
		ln = strings.TrimSpace(ln)
		if ln == "" {
			continue
		}
		// `auditctl -l` prints the canonical rule lines already (e.g., -a,always,exit -F arch=b64 -S execve ...).
		rules = append(rules, ln)
	}
	return rules
}
