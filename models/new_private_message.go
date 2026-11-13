package models

import (
	"encoding/json"
	"time"
)

type NewPrivateMessageReceive struct {
	ReceiverId string          `json:"receiverId" validate:"required,uuid"`
	Content    json.RawMessage `json:"content" validate:"required"`
	Key        json.RawMessage `json:"key" validate:"required"`
	Nonce      json.RawMessage `json:"nonce" validate:"required"`
	Mac        json.RawMessage `json:"mac" validate:"required"`
	SentAt     time.Time       `json:"sentAt" validate:"required"`
}
