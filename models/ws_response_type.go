package models

type WsResponseType string

const (
	WsResponseTypeError   WsResponseType = "error"
	WsResponseTypeMessage WsResponseType = "message"
)
