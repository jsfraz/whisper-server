package models

import (
	"encoding/json"
	"time"
)

type PrivateMessage struct {
	SenderId        uint64          `json:"senderId" validate:"required"`
	Message         json.RawMessage `json:"message" validate:"required"`
	SentAt          time.Time       `json:"sentAt" validate:"required"`
	RecipientOnline bool            `json:"recipientOnline" validate:"required"`
}

// Return new PrivateMessage.
//
//	@param senderId
//	@param message
//	@return PrivateMessage
func NewPrivateMessage(senderId uint64, message json.RawMessage, sentAt time.Time, recipientOnline bool) PrivateMessage {
	return PrivateMessage{
		SenderId:        senderId,
		Message:         message,
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
