package models

import "encoding/json"

type WsMessage struct {
	Type    WsMessageType   `json:"type" validate:"required,type"`
	Payload json.RawMessage `json:"payload" validate:"required"`
}
