// Package datatypes defines shared data structures used across the application.
package datatypes

import (
	"os"
	"time"
)

// ChaosMessage represents a message structure used in the chaos agent communication.
type ChaosMessage struct {
	Status     string `json:"status"`
	Message    string `json:"message"`
	Token      string `json:"token"`
	TokenCheck bool   `json:"token_check"`
}

// FileMeta holds metadata about a file necessary for preserving its state.
type FileMeta struct {
	Mode  os.FileMode
	UID   int
	GID   int
	Atime time.Time
	Mtime time.Time
	XAttr map[string][]byte // best-effort
}
