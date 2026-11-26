package handlers

import (
	"encoding/json"
	"errors"
	"jsfraz/shept-server/database"
	"jsfraz/shept-server/models"
	"jsfraz/shept-server/utils"
	"time"
)

// processTyping relays one typing hint to the peer's live devices.
func processTyping(conn *utils.WSConnection, payload json.RawMessage) {
	var request models.WsTypingRequest
	if !decodeEphemeral(conn, payload, &request) {
		return
	}
	allowed, err := conversationExists(conn, request.PeerId)
	if err != nil || !allowed {
		return
	}
	utils.GetSingleton().Hub.SendToUser(request.PeerId, models.NewWsResponse(
		models.WsResponseTypeTyping, models.TypingFrame{V: 1, PeerId: conn.UserId}))
}

// processPresence relays the caller's state to the peers it names. A peer with
// no conversation is skipped silently, so the relay cannot be used to learn
// whether an account exists.
func processPresence(conn *utils.WSConnection, payload json.RawMessage) {
	var request models.WsPresenceRequest
	if !decodeEphemeral(conn, payload, &request) {
		return
	}
	frame := models.PresenceFrame{V: 1, PeerId: conn.UserId, Online: request.Online}
	if !request.Online {
		now := time.Now().UTC()
		frame.LastSeenAt = &now
	}
	for _, peerId := range request.PeerIds {
		allowed, err := conversationExists(conn, peerId)
		if err != nil {
			return
		}
		if !allowed {
			continue
		}
		utils.GetSingleton().Hub.SendToUser(peerId, models.NewWsResponse(
			models.WsResponseTypePresence, frame))
	}
}

// processLiveLocation relays one encrypted position to the peer's live devices.
func processLiveLocation(conn *utils.WSConnection, payload json.RawMessage) {
	var request models.WsLiveLocationRequest
	if !decodeEphemeral(conn, payload, &request) {
		return
	}
	allowed, err := conversationExists(conn, request.PeerId)
	if err != nil || !allowed {
		return
	}
	utils.GetSingleton().Hub.SendToUser(request.PeerId, models.NewWsResponse(
		models.WsResponseTypeLiveLocation, models.LiveLocationFrame{
			V:       1,
			PeerId:  conn.UserId,
			Payload: request.Payload,
		}))
}

// decodeEphemeral parses and validates one relay frame, answering the client
// when it does not fit.
func decodeEphemeral(conn *utils.WSConnection, payload json.RawMessage, out interface{}) bool {
	if err := json.Unmarshal(payload, out); err != nil {
		conn.SendError(errors.New(utils.ErrorInvalidMessage))
		return false
	}
	if err := wsValidator.Struct(out); err != nil {
		conn.SendError(errors.New(utils.ErrorInvalidMessage))
		return false
	}
	return true
}

// conversationExists reports whether the caller may signal that peer, answering
// the client when the check itself fails.
func conversationExists(conn *utils.WSConnection, peerId string) (bool, error) {
	allowed, err := database.HasConversation(conn.UserId, peerId)
	if err != nil {
		conn.SendError(err)
		return false, err
	}
	if !allowed {
		conn.SendError(errors.New(utils.ErrorNoConversation))
		return false, nil
	}
	return true, nil
}
