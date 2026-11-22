package models

type WsResponseType string

const (
	WsResponseTypeError         WsResponseType = "error"
	WsResponseTypeMessages      WsResponseType = "messages"
	WsResponseTypeSynced        WsResponseType = "synced"
	WsResponseTypeDeleteAccount WsResponseType = "deleteAccount"
	WsResponseTypeAck           WsResponseType = "ack"
	WsResponseTypeEdit          WsResponseType = "edit"
	WsResponseTypeDelete        WsResponseType = "delete"
	WsResponseTypeReaction      WsResponseType = "reaction"
	WsResponseTypePin           WsResponseType = "pin"
	WsResponseTypeOpen          WsResponseType = "open"
	WsResponseTypeRead          WsResponseType = "read"
	// The ephemeral frames are relayed only, never stored or replayed.
	WsResponseTypeTyping       WsResponseType = "typing"
	WsResponseTypePresence     WsResponseType = "presence"
	WsResponseTypeLiveLocation WsResponseType = "liveLocation"
)

// ResponseTypeForKind maps a mutation log kind to the frame that carries it.
// The two vocabularies are deliberately the same words.
//
//	@param kind
//	@return WsResponseType
func ResponseTypeForKind(kind string) WsResponseType {
	switch kind {
	case MessageEventEdit:
		return WsResponseTypeEdit
	case MessageEventDelete:
		return WsResponseTypeDelete
	case MessageEventReaction:
		return WsResponseTypeReaction
	case MessageEventPin:
		return WsResponseTypePin
	case MessageEventOpen:
		return WsResponseTypeOpen
	}
	return WsResponseTypeError
}
