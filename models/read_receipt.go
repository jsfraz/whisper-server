package models

import "time"

// ReadReceipt is one account's read cursor in one conversation, as its peer
// sees it. The server stores a single instant, never which message was read.
type ReadReceipt struct {
	// V versions the frame; everything from this release on is version 1.
	V int `json:"v" validate:"required" gorm:"-"`
	// PeerId is the account the cursor belongs to, seen from the recipient.
	PeerId string    `json:"peerId" validate:"required" gorm:"column:peer_id"`
	ReadAt time.Time `json:"readAt" validate:"required"`
}

// SetReadRequest moves the caller's read cursor of one conversation.
type SetReadRequest struct {
	PeerId string    `path:"peerId" validate:"required,uuid" description:"The other participant of the conversation"`
	ReadAt time.Time `json:"readAt" validate:"required" description:"Instant up to which the chat was read"`
}
