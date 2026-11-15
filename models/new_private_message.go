package models

import (
	"encoding/json"
	"time"
)

// MaxSentAtSkew bounds how far a client timestamp may be from the server
// clock. The timestamp keys the offline queue and the delivery dedup, so a
// wild value could overwrite another message or pin a chat forever.
const MaxSentAtSkew = 5 * time.Minute

// IsSentAtPlausible reports whether a client timestamp is close enough to now.
//
//	@param sentAt
//	@return bool
func IsSentAtPlausible(sentAt time.Time) bool {
	diff := time.Since(sentAt)
	if diff < 0 {
		diff = -diff
	}
	return diff <= MaxSentAtSkew
}

type NewPrivateMessageReceive struct {
	ReceiverId string          `json:"receiverId" validate:"required,uuid"`
	Content    json.RawMessage `json:"content" validate:"required"`
	Key        json.RawMessage `json:"key" validate:"required"`
	Nonce      json.RawMessage `json:"nonce" validate:"required"`
	Mac        json.RawMessage `json:"mac" validate:"required"`
	SentAt     time.Time       `json:"sentAt" validate:"required"`
}
