package models

import "encoding/json"

type PrivateMessageReceive struct {
	ReceiverId uint64          `json:"receiverId" validate:"required"`
	Message    json.RawMessage `json:"message" validate:"required"`
}
