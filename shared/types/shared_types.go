package types

import "encoding/json"

type Envelope struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

type Heartbeat struct {
	Timestamp string `json:"timestamp"`
	Status    string `json:"status"`
	Service   string `json:"service"`
	Version   string `json:"version"`
	TOTP      string `json:"totp"`
	Checksum  string `json:"checksum"`
	First     bool   `json:"first"`
}

type ChaosReport struct {
	Timestamp string `json:"timestamp"`
	Action    string `json:"action"`
	Agent     string `json:"agent"`
}

type General struct {
	Message string `json:"message"`
}
