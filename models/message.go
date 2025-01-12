package models

import "encoding/json"

type Message struct {
	Topic   Topic       `json:"topic"`
	Payload interface{} `json:"payload"`
}

// Marshals the message to a JSON binary message.
//
//	@param message
//	@return []byte
//	@return error
func MarshalMessage(message Message) ([]byte, error) {
	json, err := json.Marshal(message)
	if err != nil {
		return nil, err
	}
	return json, nil
}
