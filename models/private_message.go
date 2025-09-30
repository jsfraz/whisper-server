package models

import (
	"encoding/json"
	"time"
)

type PrivateMessage struct {
	SenderId        uint64          `json:"senderId" validate:"required"`
	Content         json.RawMessage `json:"content" validate:"required"`
	Key             json.RawMessage `json:"key" validate:"required"`
	Nonce           json.RawMessage `json:"nonce" validate:"required"`
	Mac             json.RawMessage `json:"mac" validate:"required"`
	SentAt          time.Time       `json:"sentAt" validate:"required"`
	RecipientOnline bool            `json:"recipientOnline" validate:"required"`
}

// Return new PrivateMessage.
//
//	@param senderId
//	@param content
//	@return PrivateMessage
func NewPrivateMessage(senderId uint64, content json.RawMessage, key json.RawMessage, nonce json.RawMessage, mac json.RawMessage, sentAt time.Time, recipientOnline bool) PrivateMessage {
	return PrivateMessage{
		SenderId:        senderId,
		Content:         content,
		Key:             key,
		Nonce:           nonce,
		Mac:             mac,
		SentAt:          sentAt,
		RecipientOnline: recipientOnline,
	}
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
