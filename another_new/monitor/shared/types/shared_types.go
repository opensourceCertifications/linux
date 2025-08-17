package types

//import "encoding/json"
//
//type Envelope struct {
//	Type string          `json:"type"`
//	Data json.RawMessage `json:"data"`
//}

type ChaosMessage struct {
	Status     string `json:"status"`
	Message    string `json:"message"`
	Token      string `json:"token"`
	TokenCheck bool   `json:"token_check"`
}

//type ChaosReport struct {
//	Timestamp string `json:"timestamp"`
//	Action    string `json:"action"`
//	Agent     string `json:"agent"`
//}
//
//type General struct {
//	Message string `json:"message"`
//}
