package types

type ChaosMessage struct {
	Status     string `json:"status"`
	Message    string `json:"message"`
	Token      string `json:"token"`
	TokenCheck bool   `json:"token_check"`
}
