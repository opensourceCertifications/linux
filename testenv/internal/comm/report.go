package comm

import (
	"encoding/json"
	"log"
	"net"
)

var monitorAddress string

func Init(address string) {
	monitorAddress = address
}

func SendMessage(msgType string, payload interface{}) {
	var rawData json.RawMessage

	switch v := payload.(type) {
	case []byte:
		rawData = json.RawMessage(v)
	default:
		data, err := json.Marshal(v)
		if err != nil {
			log.Printf("Failed to marshal payload for type %s: %v", msgType, err)
			return
		}
		rawData = data
	}

	envelope := map[string]interface{}{
		"type": msgType,
		"data": rawData,
	}

	finalData, err := json.Marshal(envelope)
	if err != nil {
		log.Printf("Failed to marshal envelope: %v", err)
		return
	}

	//log.Printf("Sending message to monitor at %s:\n%s\n", monitorAddress, string(finalData))
	conn, err := net.Dial("tcp", monitorAddress)
	if err != nil {
		log.Printf("Failed to connect to monitor at %s: %v", monitorAddress, err)
		return
	}
	defer conn.Close()

	_, err = conn.Write(append(finalData, '\n'))
	if err != nil {
		log.Printf("Failed to send message to monitor: %v", err)
	}
}
