package models

type WsResponseType string

const (
	WsResponseTypeError         WsResponseType = "error"
	WsResponseTypeMessages      WsResponseType = "messages"
	WsResponseTypeSynced        WsResponseType = "synced"
	WsResponseTypeDeleteAccount WsResponseType = "deleteAccount"
	WsResponseTypeAck           WsResponseType = "ack"
)
