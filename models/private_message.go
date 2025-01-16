package models

import "encoding/json"

type PrivateMessage struct {
	SenderId uint64          `json:"senderId" validate:"required"`
	Message  json.RawMessage `json:"message" validate:"required"`
}

// Return new PrivateMessage.
//
//	@param senderId
//	@param message
//	@return PrivateMessage
func NewPrivateMessage(senderId uint64, message json.RawMessage) PrivateMessage {
	return PrivateMessage{
		SenderId: senderId,
		Message:  message,
	}
}
