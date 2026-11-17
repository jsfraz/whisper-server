package models

import "github.com/go-playground/validator/v10"

// WsMessageType is the type for supported topics for WebSocket communication
type WsMessageType string

// Supported topics
const (
	WsMessageTopicMessage WsMessageType = "message"
	WsMessageTopicSync    WsMessageType = "sync"
)

// ValidateWsMessageType is a custom validator for WsMessageTopic
//
//	@param fl
//	@return bool
func ValidateWsMessageType(fl validator.FieldLevel) bool {
	topic := WsMessageType(fl.Field().String())
	switch topic {
	case WsMessageTopicMessage, WsMessageTopicSync:
		return true
	}
	return false
}
