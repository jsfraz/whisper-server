package models

import (
	"encoding/json"
	"fmt"
	"time"
)

// ConversationKey is the participant pair of a chat in one stable order, so a
// conversation is a single index range regardless of who sends.
//
//	@param firstId
//	@param secondId
//	@return string
func ConversationKey(firstId string, secondId string) string {
	if firstId < secondId {
		return fmt.Sprintf("%s|%s", firstId, secondId)
	}
	return fmt.Sprintf("%s|%s", secondId, firstId)
}

// PrivateMessage is one archived ciphertext message. The server never holds the
// plaintext; Key, Nonce and Mac are what the recipient needs to open Content.
type PrivateMessage struct {
	Seq      int64  `json:"seq" validate:"required" gorm:"->;autoIncrement"`
	Id       string `json:"id" validate:"required" gorm:"primarykey"`
	SenderId string `json:"senderId" validate:"required"`
	// PeerId is the other participant, filled per viewer by ToWire.
	PeerId       string          `json:"peerId" validate:"required" gorm:"-"`
	Conversation string          `json:"-" validate:"required"`
	ReceiverId   string          `json:"-" validate:"required"`
	Content      json.RawMessage `json:"content" validate:"required"`
	Key          json.RawMessage `json:"key" validate:"required" gorm:"column:wrapped_key"`
	Nonce        json.RawMessage `json:"nonce" validate:"required"`
	Mac          json.RawMessage `json:"mac" validate:"required"`
	SentAt       time.Time       `json:"sentAt" validate:"required"`
	ExpiresAt    time.Time       `json:"-"`
	// TTLSeconds is the sender's requested lifetime. It is never stored:
	// InsertMessage folds it into ExpiresAt.
	TTLSeconds int        `json:"-" gorm:"-"`
	DeletedAt  *time.Time `json:"deletedAt"`
	EditedAt   *time.Time `json:"editedAt"`
	// OpenedAt is when the receiver opened a view-once attachment. The flag
	// itself is inside the ciphertext, so the server only records the claim.
	OpenedAt        *time.Time `json:"openedAt"`
	RecipientOnline bool       `json:"recipientOnline" validate:"required" gorm:"-"`
}

// ToWire returns the message as one participant sees it.
//
//	@param viewerId
//	@param recipientOnline
//	@return PrivateMessage
func (p PrivateMessage) ToWire(viewerId string, recipientOnline bool) PrivateMessage {
	p.RecipientOnline = recipientOnline
	if p.SenderId == viewerId {
		p.PeerId = p.ReceiverId
	} else {
		p.PeerId = p.SenderId
	}
	return p
}

// Marshall PrivateMessage to binary.
//
//	@return []byte
//	@return error
func (p PrivateMessage) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

// Unmarshall PrivateMessage.
//
//	@param jsonBytes
//	@return *PrivateMessage
//	@return error
func PrivateMessageFromJson(jsonBytes []byte) (*PrivateMessage, error) {
	var p PrivateMessage
	err := json.Unmarshal(jsonBytes, &p)
	if err != nil {
		return nil, err
	}
	return &p, nil
}
