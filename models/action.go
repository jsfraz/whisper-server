package models

import "github.com/go-playground/validator/v10"

// Action is the type for supported actions for WebSocket communication
type Action string

// Supported actions
const (
	ActionSubscribe   Action = "subscribe"
	ActionUnsubscribe Action = "unsubscribe"
	ActionPublish     Action = "publish"
)

// ValidateAction is a custom validator for Action
//
//	@param fl
//	@return bool
func ValidateAction(fl validator.FieldLevel) bool {
	action := Action(fl.Field().String())
	switch action {
	case ActionSubscribe, ActionUnsubscribe, ActionPublish:
		return true
	}
	return false
}
