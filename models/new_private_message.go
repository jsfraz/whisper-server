package models

import (
	"encoding/json"
	"time"
)

type NewPrivateMessageReceive struct {
	ReceiverId uint64          `json:"receiverId" validate:"required"`
	Message    json.RawMessage `json:"message" validate:"required"`
	SentAt     time.Time       `json:"sentAt" validate:"required"`
}
