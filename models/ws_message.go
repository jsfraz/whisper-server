package models

import "encoding/json"

type WsMessage struct {
	Action Action `json:"action" validate:"required,action"`
	Topic  Topic  `json:"topic" validate:"required,topic"`
	// Required if Topic is Publish
	Payload json.RawMessage `json:"payload" validate:"required_if=Action publish,omitempty"`
}
