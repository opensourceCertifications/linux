// Package datatypes defines shared data structures used across the application.
package datatypes

// ChaosMessage represents a message structure used in the chaos agent communication.
type ChaosMessage struct {
	Status     string `json:"status"`
	Message    string `json:"message"`
	Token      string `json:"token"`
	TokenCheck bool   `json:"token_check"`
}
