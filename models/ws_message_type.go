package models

import "github.com/go-playground/validator/v10"

// WsMessageType is the type for supported topics for WebSocket communication
type WsMessageType string

// Supported topics
const (
	WsMessageTopicMessage WsMessageType = "message"
	WsMessageTopicSync    WsMessageType = "sync"
	// The ephemeral topics are relayed and never archived.
	WsMessageTopicTyping       WsMessageType = "typing"
	WsMessageTopicPresence     WsMessageType = "presence"
	WsMessageTopicLiveLocation WsMessageType = "liveLocation"
)

// ValidateWsMessageType is a custom validator for WsMessageTopic
//
//	@param fl
//	@return bool
func ValidateWsMessageType(fl validator.FieldLevel) bool {
	topic := WsMessageType(fl.Field().String())
	switch topic {
	case WsMessageTopicMessage, WsMessageTopicSync,
		WsMessageTopicTyping, WsMessageTopicPresence, WsMessageTopicLiveLocation:
		return true
	}
	return false
}
