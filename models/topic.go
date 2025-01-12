package models

import "github.com/go-playground/validator/v10"

// Topic is the type for supported topics for WebSocket communication
type Topic string

// Supported topics
const (
	TopicMessage Topic = "message"
)

// ValidateTopic is a custom validator for Topic
//
//	@param fl
//	@return bool
func ValidateTopic(fl validator.FieldLevel) bool {
	topic := Topic(fl.Field().String())
	switch topic {
	case TopicMessage:
		return true
	}
	return false
}
