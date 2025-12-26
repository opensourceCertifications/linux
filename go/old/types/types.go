// Package datatypes defines shared data structures used across the application.
package types

import (
    "time"
)

type Probe struct {
    Addr     string
    User     string
    Timeout  time.Duration
}
