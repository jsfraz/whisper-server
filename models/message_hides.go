package models

import "time"

// MessageIdPath addresses one archived message.
type MessageIdPath struct {
	Id string `path:"id" validate:"required,uuid" description:"The message id"`
}

// ClearConversationQuery deletes a whole conversation for the calling account.
// Before is an RFC3339 instant and defaults to now, so a retry with an older
// instant cannot resurrect anything.
type ClearConversationQuery struct {
	PeerId string `query:"peerId" validate:"required,uuid" description:"The other participant of the conversation"`
	Before string `query:"before" description:"RFC3339 instant to delete up to (default: now)"`
}

// ClearConversationResponse tells the client which cut is now in force, so it
// can drop the same messages locally even when its clock is off.
type ClearConversationResponse struct {
	ClearedAt time.Time `json:"clearedAt" validate:"required"`
}
