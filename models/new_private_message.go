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
	// TTLSeconds is a disappearing message: the sender asks for a life shorter
	// than the account retention. Absent or 0 means the retention alone applies.
	TTLSeconds int `json:"ttlSeconds" validate:"omitempty,gte=1,lte=604800"`
}

// MaxMessageTTLSeconds is the upper bound of a disappearing message: a week.
const MaxMessageTTLSeconds = 7 * 24 * 60 * 60

// EffectiveExpiry returns when a message leaves the archive: the shorter of the
// sender's requested lifetime and the account retention wins.
//
//	@param sentAt
//	@param retentionDays
//	@param ttlSeconds
//	@return time.Time
func EffectiveExpiry(sentAt time.Time, retentionDays int, ttlSeconds int) time.Time {
	expiresAt := sentAt.AddDate(0, 0, retentionDays)
	if ttlSeconds > 0 {
		if ttl := sentAt.Add(time.Duration(ttlSeconds) * time.Second); ttl.Before(expiresAt) {
			return ttl
		}
	}
	return expiresAt
}
