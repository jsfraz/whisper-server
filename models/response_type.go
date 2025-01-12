package models

type ResponseType string

const (
	ResponseTypeError   ResponseType = "error"
	ResponseTypeMessage ResponseType = "message"
)
