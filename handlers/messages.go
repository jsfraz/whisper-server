package handlers

import (
	"encoding/json"
	"errors"
	"jsfraz/shept-server/database"
	"jsfraz/shept-server/models"
	"jsfraz/shept-server/push"
	"jsfraz/shept-server/utils"
	"log/slog"

	"github.com/go-playground/validator/v10"
)

// syncPageSize bounds one sync answer; a device that was away for a long time
// walks the delta in pages.
const syncPageSize = 200

// wsValidator validates WebSocket payloads. A validator is safe for concurrent
// use, so one instance is shared instead of rebuilt per message.
var wsValidator = func() *validator.Validate {
	v := validator.New()
	_ = v.RegisterValidation("type", models.ValidateWsMessageType)
	return v
}()

// ProcessWsMessage dispatches one frame from an authenticated socket.
//
//	@param conn
//	@param message
func ProcessWsMessage(conn *utils.WSConnection, message models.WsMessage) {
	switch message.Type {
	case models.WsMessageTopicMessage:
		processPrivateMessage(conn, message.Payload)
	case models.WsMessageTopicSync:
		processSync(conn, message.Payload)
	case models.WsMessageTopicTyping:
		processTyping(conn, message.Payload)
	case models.WsMessageTopicPresence:
		processPresence(conn, message.Payload)
	case models.WsMessageTopicLiveLocation:
		processLiveLocation(conn, message.Payload)
	default:
		conn.SendError(errors.New(utils.ErrorUnsupportedMessageType))
	}
}

// processPrivateMessage validates, archives and delivers one private message.
func processPrivateMessage(conn *utils.WSConnection, payload json.RawMessage) {
	var privateMessage models.NewPrivateMessageReceive
	if err := json.Unmarshal(payload, &privateMessage); err != nil {
		conn.SendError(errors.New(utils.ErrorInvalidMessage))
		return
	}
	if err := wsValidator.Struct(privateMessage); err != nil {
		conn.SendError(errors.New(utils.ErrorInvalidMessage))
		return
	}
	// The timestamp keys the archive row and the retry dedup, so it has to be
	// close to now.
	if !models.IsSentAtPlausible(privateMessage.SentAt) {
		conn.SendError(errors.New(utils.ErrorImplausibleTimestamp))
		return
	}
	// The socket was authenticated once; the account may be gone since.
	active, err := userIsActive(conn.UserId)
	if err != nil {
		conn.SendError(err)
		return
	}
	if !active {
		conn.SendError(errors.New(utils.ErrorAccountInactive))
		conn.Close()
		return
	}
	// Check whether user is sending message to self (in production)
	if conn.UserId == privateMessage.ReceiverId && utils.GetSingleton().Config.GinMode == "release" {
		conn.SendError(errors.New(utils.ErrorCannotSendToSelf))
		return
	}
	// Check if message is being sent to user that will be deleted
	toDelete, err := database.WillUserBeDeleted(privateMessage.ReceiverId)
	if err != nil {
		conn.SendError(err)
		return
	}
	if toDelete {
		conn.SendError(errors.New(utils.ErrorCannotSendToUser))
		return
	}
	exists, err := database.UserExistsById(privateMessage.ReceiverId)
	if err != nil {
		conn.SendError(err)
		return
	}
	if !exists {
		conn.SendError(errors.New(utils.ErrorUserNotFound))
		return
	}

	// The archive is the source of truth: history, delta sync and the offline
	// buffer are all this one row.
	retention, err := database.GetRetentionDaysOf(conn.UserId, privateMessage.ReceiverId)
	if err != nil {
		conn.SendError(err)
		return
	}
	message := models.PrivateMessage{
		SenderId:   conn.UserId,
		ReceiverId: privateMessage.ReceiverId,
		Content:    privateMessage.Content,
		Key:        privateMessage.Key,
		Nonce:      privateMessage.Nonce,
		Mac:        privateMessage.Mac,
		SentAt:     privateMessage.SentAt,
		TTLSeconds: privateMessage.TTLSeconds,
	}
	if err := database.InsertMessage(&message, retention); err != nil {
		conn.SendError(err)
		return
	}
	// Hand it to every live device of the recipient. A device that is offline
	// picks it up from the archive on its next sync.
	hub := utils.GetSingleton().Hub
	hub.SendToUser(privateMessage.ReceiverId, models.NewWsResponse(
		models.WsResponseTypeMessages,
		[]models.PrivateMessage{message.ToWire(privateMessage.ReceiverId, true)}))
	if !hub.Online(privateMessage.ReceiverId) {
		go func() {
			if err := push.Send(privateMessage.ReceiverId, "New message", "You have a new message"); err != nil {
				slog.Warn("failed to send push notification",
					"user_id", utils.SanitizeLogValue(privateMessage.ReceiverId),
					"error", utils.SanitizeLogValue(err.Error()))
			}
		}()
	}
	// Only now the sender may show the message as sent
	conn.Send(models.NewWsResponse(models.WsResponseTypeAck, models.DeliveryAck{
		Id:         message.Id,
		Seq:        message.Seq,
		ReceiverId: message.ReceiverId,
		SentAt:     message.SentAt,
	}))
}

// processSync answers a delta request: every archived message newer than the
// client's cursor, plus the accounts it mentions.
func processSync(conn *utils.WSConnection, payload json.RawMessage) {
	var request models.WsSyncRequest
	if err := json.Unmarshal(payload, &request); err != nil {
		conn.SendError(errors.New(utils.ErrorInvalidMessage))
		return
	}
	if err := wsValidator.Struct(request); err != nil {
		conn.SendError(errors.New(utils.ErrorInvalidMessage))
		return
	}
	messages, err := database.SyncSince(conn.UserId, request.SinceSeq, syncPageSize)
	if err != nil {
		conn.SendError(err)
		return
	}
	peers, err := database.GetUsersByIds(database.ConversationsOf(messages, conn.UserId))
	if err != nil {
		conn.SendError(err)
		return
	}
	wire := make([]models.PrivateMessage, 0, len(messages))
	syncedThrough := request.SinceSeq
	for _, message := range messages {
		wire = append(wire, message.ToWire(conn.UserId, false))
		if message.Seq > syncedThrough {
			syncedThrough = message.Seq
		}
	}
	peerList := make([]models.User, 0, len(peers))
	for _, peer := range peers {
		peerList = append(peerList, peer)
	}
	// Mutations are replayed only after the device holds the messages they talk
	// about, so an event cursor never outruns the archive page.
	events := []models.MessageEvent{}
	eventsThrough := int64(0)
	hasMore := len(messages) == syncPageSize
	if request.SinceEventSeq != nil {
		missed, err := database.EventsSince(conn.UserId, *request.SinceEventSeq, syncedThrough, syncPageSize)
		if err != nil {
			conn.SendError(err)
			return
		}
		eventsThrough = *request.SinceEventSeq
		for _, event := range missed {
			events = append(events, event.ToWire(conn.UserId))
			if event.EventSeq > eventsThrough {
				eventsThrough = event.EventSeq
			}
		}
		if len(missed) == syncPageSize {
			hasMore = true
		}
	}
	cleared, err := database.ClearedFor(conn.UserId)
	if err != nil {
		conn.SendError(err)
		return
	}
	reads, err := database.ReadCursorsFor(conn.UserId)
	if err != nil {
		conn.SendError(err)
		return
	}
	conn.Send(models.NewWsResponse(models.WsResponseTypeSynced, models.WsSynced{
		Messages:      wire,
		Peers:         peerList,
		Events:        events,
		Cleared:       cleared,
		Reads:         reads,
		SyncedThrough: syncedThrough,
		EventsThrough: eventsThrough,
		HasMore:       hasMore,
	}))
}

// userIsActive reports whether an account may still act: it exists and is not
// pending deletion.
func userIsActive(userId string) (bool, error) {
	toDelete, err := database.WillUserBeDeleted(userId)
	if err != nil {
		return false, err
	}
	if toDelete {
		return false, nil
	}
	return database.UserExistsById(userId)
}
