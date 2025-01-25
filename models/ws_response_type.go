package models

type WsResponseType string

const (
	WsResponseTypeError    WsResponseType = "error"
	WsResponseTypeMessages WsResponseType = "messages"
)
