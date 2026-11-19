package models

import (
	"encoding/json"
	"strings"
	"time"
)

// Kinds of a shared message mutation. Each one is also the name of the
// WebSocket frame that carries it live, so a client hears the same vocabulary
// from sync and from the socket.
const (
	MessageEventEdit     = "edit"
	MessageEventDelete   = "delete"
	MessageEventReaction = "reaction"
	MessageEventPin      = "pin"
)

// MessageEvent is one entry of the mutation log. It follows the archive
// layout: an edit and a reaction carry ciphertext plus the key material the
// peer needs to open it, so the server still never sees the plaintext.
type MessageEvent struct {
	// V versions the frame; the pre-1.0 frames carry none, everything from here
	// on does.
	V         int    `json:"v" validate:"required" gorm:"-"`
	EventSeq  int64  `json:"eventSeq" validate:"required" gorm:"primarykey"`
	MessageId string `json:"messageId" validate:"required"`
	// MessageSeq lets sync hold back an event until the device has the message.
	MessageSeq   int64           `json:"-" validate:"required"`
	Conversation string          `json:"-" validate:"required"`
	PeerId       string          `json:"peerId" validate:"required" gorm:"-"`
	SentAt       time.Time       `json:"sentAt" validate:"required"`
	Kind         string          `json:"kind" validate:"required"`
	ActorId      string          `json:"actorId" validate:"required"`
	Content      json.RawMessage `json:"content"`
	Key          json.RawMessage `json:"key" gorm:"column:wrapped_key"`
	Nonce        json.RawMessage `json:"nonce"`
	Mac          json.RawMessage `json:"mac"`
	// Pinned is set on a pin event only: true pins, false unpins.
	Pinned    *bool     `json:"pinned,omitempty"`
	CreatedAt time.Time `json:"createdAt" validate:"required"`
}

// ToWire returns the event as one participant sees it.
//
//	@param viewerId
//	@return MessageEvent
func (e MessageEvent) ToWire(viewerId string) MessageEvent {
	e.V = 1
	e.PeerId = PeerOfConversation(e.Conversation, viewerId)
	return e
}

// PeerOfConversation returns the other participant of a conversation key.
//
//	@param conversation
//	@param viewerId
//	@return string
func PeerOfConversation(conversation string, viewerId string) string {
	parts := strings.SplitN(conversation, "|", 2)
	if len(parts) != 2 {
		return ""
	}
	if parts[0] == viewerId {
		return parts[1]
	}
	return parts[0]
}

// ParticipantsOfConversation returns both sides of a conversation key.
//
//	@param conversation
//	@return []string
func ParticipantsOfConversation(conversation string) []string {
	parts := strings.SplitN(conversation, "|", 2)
	if len(parts) != 2 {
		return nil
	}
	return parts
}

// ConversationCleared is one whole-chat cut of the calling account, replayed on
// sync so a device that was offline while the chat was cleared empties it too.
type ConversationCleared struct {
	PeerId    string    `json:"peerId" validate:"required"`
	ClearedAt time.Time `json:"clearedAt" validate:"required"`
}

// MessageCiphertext is new ciphertext for an existing message: an edit body or
// a reaction body.
type MessageCiphertext struct {
	Content json.RawMessage `json:"content" validate:"required"`
	Key     json.RawMessage `json:"key" validate:"required"`
	Nonce   json.RawMessage `json:"nonce" validate:"required"`
	Mac     json.RawMessage `json:"mac" validate:"required"`
}

// EditMessageRequest addresses one own message and carries its new ciphertext.
type EditMessageRequest struct {
	Id      string          `path:"id" validate:"required,uuid" description:"The message id"`
	Content json.RawMessage `json:"content" validate:"required"`
	Key     json.RawMessage `json:"key" validate:"required"`
	Nonce   json.RawMessage `json:"nonce" validate:"required"`
	Mac     json.RawMessage `json:"mac" validate:"required"`
}

// ReactionRequest addresses one message and carries the reaction ciphertext.
type ReactionRequest struct {
	Id      string          `path:"id" validate:"required,uuid" description:"The message id"`
	Content json.RawMessage `json:"content" validate:"required"`
	Key     json.RawMessage `json:"key" validate:"required"`
	Nonce   json.RawMessage `json:"nonce" validate:"required"`
	Mac     json.RawMessage `json:"mac" validate:"required"`
}

// EditMessageResponse tells the client which instant the edit is stamped with,
// so it can show "edited" without trusting its own clock.
type EditMessageResponse struct {
	EditedAt time.Time `json:"editedAt" validate:"required"`
}

// PinnedMessage is one pinned message of a chat, newest pin first.
type PinnedMessage struct {
	MessageId string    `json:"messageId" validate:"required"`
	PinnedBy  string    `json:"pinnedBy" validate:"required"`
	PinnedAt  time.Time `json:"pinnedAt" validate:"required"`
}
