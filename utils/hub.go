package utils

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"jsfraz/whisper-server/models"
	"log/slog"
	"slices"
	"sync"
	"time"

	messaging "firebase.google.com/go/v4/messaging"
	"github.com/go-playground/validator/v10"
	"github.com/valkey-io/valkey-go"
)

// dedupTtl is how long a delivered message is remembered so a client resend
// (no delivery ack received) does not produce a duplicate.
const dedupTtl = 3600

// pushQueueSize bounds pending push notifications; a burst of messages to
// offline users must not spawn unbounded goroutines.
const pushQueueSize = 256

// Hub manages WebSocket connections and message broadcasting
type Hub struct {
	// Map of active WebSocket connections and their status
	Connections map[*WSConnection]bool
	// Channel for broadcasting messages to all subscribed connections
	// Anonymous struct is here for pairing senderId and message
	Broadcast chan struct {
		SenderId string
		Message  models.WsMessage
	}
	// Channel for registering new WebSocket connections
	Register chan *WSConnection
	// Channel for unregistering (removing) WebSocket connections
	Unregister chan *WSConnection
	// Pending push notifications, handled off the message path
	push chan string
	// Positive user existence results, so a message does not cost a query
	exists *userExistenceCache
	// Mutex for thread-safe access to the Connections map
	mu sync.RWMutex
}

// userExistenceCache remembers users that exist. Only positive results are
// cached: a missing user must be re-checked, a known one saves a query per
// message.
type userExistenceCache struct {
	mu   sync.Mutex
	seen map[string]time.Time
	ttl  time.Duration
}

func newUserExistenceCache(ttl time.Duration) *userExistenceCache {
	return &userExistenceCache{seen: make(map[string]time.Time), ttl: ttl}
}

func (c *userExistenceCache) get(userId string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	seen, ok := c.seen[userId]
	if !ok {
		return false
	}
	if time.Since(seen) > c.ttl {
		delete(c.seen, userId)
		return false
	}
	return true
}

func (c *userExistenceCache) put(userId string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.seen[userId] = time.Now()
}

// queueCounterPrefix counts the queued messages of one receiver.
const queueCounterPrefix = "msgqueue:"

// wsValidator validates WebSocket payloads. A validator is safe for concurrent
// use, so one instance is shared instead of rebuilt per message.
var wsValidator = func() *validator.Validate {
	v := validator.New()
	_ = v.RegisterValidation("type", models.ValidateWsMessageType)
	return v
}()

// NewHub creates and initializes a new Hub instance
//
//	@return *Hub
func NewHub() *Hub {
	return &Hub{
		Connections: make(map[*WSConnection]bool),
		Broadcast: make(chan struct {
			SenderId string
			Message  models.WsMessage
		}, 64),
		Register:   make(chan *WSConnection, 16),
		Unregister: make(chan *WSConnection, 16),
		push:       make(chan string, pushQueueSize),
		exists:     newUserExistenceCache(5 * time.Second),
	}
}

// Run starts the Hub's main event loop to handle connections and messages.
//
// Nothing in here may hold the connection map lock across network I/O: a single
// slow client or a slow push call would otherwise stop message delivery for
// every user.
func (h *Hub) Run() {
	go h.pushLoop()
	for {
		select {
		// Handle new connection registration
		case conn := <-h.Register:
			h.mu.Lock()
			h.Connections[conn] = true
			h.mu.Unlock()

		// Handle connection unregistration
		case conn := <-h.Unregister:
			h.mu.Lock()
			delete(h.Connections, conn)
			h.mu.Unlock()
			conn.Close()

		// Handle incoming message
		case msgSenderPair := <-h.Broadcast:
			h.handle(msgSenderPair.SenderId, msgSenderPair.Message)
		}
	}
}

// handle dispatches one incoming client message.
func (h *Hub) handle(senderId string, message models.WsMessage) {
	switch message.Type {
	case models.WsMessageTopicMessage:
		h.handlePrivateMessage(senderId, message.Payload)
	case models.WsMessageTopicAck:
		h.handleAck(senderId, message.Payload)
	default:
		h.sendError(senderId, errors.New(ErrorUnsupportedMessageType))
	}
}

// connectionOf returns the live connection of a user, if any.
func (h *Hub) connectionOf(userId string) *WSConnection {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for conn := range h.Connections {
		if conn.UserId == userId {
			return conn
		}
	}
	return nil
}

// handlePrivateMessage validates and delivers one private message.
func (h *Hub) handlePrivateMessage(senderId string, payload json.RawMessage) {
	var privateMessage models.NewPrivateMessageReceive
	if err := json.Unmarshal(payload, &privateMessage); err != nil {
		h.sendError(senderId, errors.New(ErrorInvalidMessage))
		return
	}
	// Validate private message
	if err := wsValidator.Struct(privateMessage); err != nil {
		h.sendError(senderId, errors.New(ErrorInvalidMessage))
		return
	}
	// The timestamp keys the queue and the dedup, so it has to be close to now.
	if !models.IsSentAtPlausible(privateMessage.SentAt) {
		h.sendError(senderId, errors.New(ErrorImplausibleTimestamp))
		return
	}
	// The socket was authenticated once; the account may be gone since.
	active, err := h.userIsActive(senderId)
	if err != nil {
		h.sendError(senderId, err)
		return
	}
	if !active {
		h.sendError(senderId, errors.New(ErrorAccountInactive))
		if conn := h.connectionOf(senderId); conn != nil {
			conn.Close()
		}
		return
	}
	// Check whether user is sending message to self (in production)
	if senderId == privateMessage.ReceiverId && GetSingleton().Config.GinMode == "release" {
		h.sendError(senderId, errors.New(ErrorCannotSendToSelf))
		return
	}
	// Check if message is being sent to user that will be deleted
	toDelete, err := h.willUserBeDeleted(privateMessage.ReceiverId)
	if err != nil {
		h.sendError(senderId, err)
		return
	}
	if toDelete {
		h.sendError(senderId, errors.New(ErrorCannotSendToUser))
		return
	}
	// Check if user exists
	exists, err := h.userExistsById(privateMessage.ReceiverId)
	if err != nil {
		h.sendError(senderId, err)
		return
	}
	if !exists {
		h.sendError(senderId, errors.New(ErrorUserNotFound))
		return
	}
	// A resend after a lost delivery ack must not deliver a second copy.
	alreadyDelivered, err := h.alreadyDelivered(senderId, privateMessage.ReceiverId, privateMessage.SentAt)
	if err != nil {
		h.sendError(senderId, err)
		return
	}
	if alreadyDelivered {
		h.sendDeliveryAck(senderId, privateMessage.ReceiverId, privateMessage.SentAt)
		return
	}

	pm := models.NewPrivateMessage(senderId, privateMessage.Content, privateMessage.Key, privateMessage.Nonce, privateMessage.Mac, privateMessage.SentAt, false)
	// Send message to the receiver's connection, or queue it for later
	if conn := h.connectionOf(privateMessage.ReceiverId); conn != nil {
		pm.RecipientOnline = true
		conn.Send(models.NewWsResponse(models.WsResponseTypeMessages, []models.PrivateMessage{pm}))
	} else {
		if err := h.pushMessage(privateMessage.ReceiverId, pm, GetSingleton().Config.MessageTtl); err != nil {
			// No ack: the client keeps the message pending and resends it.
			h.sendError(senderId, err)
			return
		}
		select {
		case h.push <- privateMessage.ReceiverId:
		default:
			slog.Warn("push queue full, notification dropped", "user_id", SanitizeLogValue(privateMessage.ReceiverId))
		}
	}
	// Recorded only after the delivery succeeded: a failed queue write must not
	// make the client's resend look like a duplicate.
	if err := h.markDelivered(senderId, privateMessage.ReceiverId, privateMessage.SentAt); err != nil {
		slog.Warn("failed to record delivered message", "error", SanitizeLogValue(err.Error()))
	}
	// Only now the sender may show the message as sent
	h.sendDeliveryAck(senderId, privateMessage.ReceiverId, privateMessage.SentAt)
}

// handleAck removes the acknowledged messages from the receiver's offline queue.
func (h *Hub) handleAck(receiverId string, payload json.RawMessage) {
	var ack models.WsMessageAck
	if err := json.Unmarshal(payload, &ack); err != nil {
		h.sendError(receiverId, errors.New(ErrorInvalidMessage))
		return
	}
	if err := wsValidator.Struct(ack); err != nil {
		h.sendError(receiverId, errors.New(ErrorInvalidMessage))
		return
	}
	keys := make([]string, 0, len(ack.Messages))
	for _, message := range ack.Messages {
		keys = append(keys, models.OfflineQueueKey(receiverId, message.SenderId, message.SentAt))
	}
	client := GetSingleton().Valkey
	if err := client.Do(context.Background(), client.B().Del().Key(keys...).Build()).Error(); err != nil {
		slog.Error("failed to ack queued messages", "error", SanitizeLogValue(err.Error()))
	}
	// Give the queue slot back. A negative counter only loosens the cap.
	if _, err := client.Do(context.Background(), client.B().Decrby().Key(queueCounterPrefix+receiverId).Decrement(int64(len(keys))).Build()).AsInt64(); err != nil {
		slog.Warn("failed to update queue counter", "error", SanitizeLogValue(err.Error()))
	}
}

// Send error response to sender. Only a client-facing code reaches the client;
// a server side failure is logged and replaced by the generic code.
//
//	@param senderId
//	@param err
func (h *Hub) sendError(senderId string, err error) {
	if conn := h.connectionOf(senderId); conn != nil {
		conn.SendError(err)
	}
}

// sendDeliveryAck tells the sender the message is safely on the server side.
func (h *Hub) sendDeliveryAck(senderId string, receiverId string, sentAt time.Time) {
	if conn := h.connectionOf(senderId); conn != nil {
		conn.Send(models.NewWsResponse(models.WsResponseTypeAck, models.DeliveryAck{
			ReceiverId: receiverId,
			SentAt:     sentAt,
		}))
	}
}

// dedupKey identifies one message of one conversation. Client timestamps are
// stable across a resend, so they work as the delivery id.
func dedupKey(receiverId string, senderId string, sentAt time.Time) string {
	return fmt.Sprintf("dedup:%s:%s:%d", receiverId, senderId, sentAt.UTC().UnixNano())
}

// alreadyDelivered reports whether a message with the same participants and
// timestamp was delivered before (a resend after a lost ack).
func (h *Hub) alreadyDelivered(senderId string, receiverId string, sentAt time.Time) (bool, error) {
	client := GetSingleton().Valkey
	// markDelivered writes a string, so this must not be a set membership test:
	// a resend would answer WRONGTYPE and the message would be lost.
	count, err := client.Do(context.Background(), client.B().Exists().Key(dedupKey(receiverId, senderId, sentAt)).Build()).AsInt64()
	return count > 0, err
}

// markDelivered records a delivered message so a resend is answered instead of
// delivered twice.
func (h *Hub) markDelivered(senderId string, receiverId string, sentAt time.Time) error {
	client := GetSingleton().Valkey
	return client.Do(context.Background(), client.B().Set().Key(dedupKey(receiverId, senderId, sentAt)).Value("1").ExSeconds(dedupTtl).Build()).Error()
}

// pushLoop sends push notifications outside the message path.
func (h *Hub) pushLoop() {
	for userId := range h.push {
		if err := h.SendPushNotification(userId, "New message", "You have a new message"); err != nil {
			slog.Warn("failed to send push notification",
				"user_id", SanitizeLogValue(userId),
				"error", SanitizeLogValue(err.Error()))
		}
	}
}

// userIsActive reports whether an account may still act: it exists and is not
// pending deletion.
func (h *Hub) userIsActive(userId string) (bool, error) {
	toDelete, err := h.willUserBeDeleted(userId)
	if err != nil {
		return false, err
	}
	if toDelete {
		return false, nil
	}
	return h.userExistsById(userId)
}

// Check if user exists by ID.
//
//	@param userId
//	@return bool
//	@return error
func (h *Hub) userExistsById(userId string) (bool, error) {
	if h.exists.get(userId) {
		return true, nil
	}
	var count int64
	err := GetSingleton().Sqlite.Model(&models.User{}).Where("id = ?", userId).Count(&count).Error
	if err != nil {
		return false, err
	}
	if count > 0 {
		h.exists.put(userId)
	}
	return count > 0, nil
}

// Push PrivateMessage to Valkey.
//
// The per receiver counter keeps one account from filling the queue of another;
// it expires with the queue itself.
//
//	@param receiverId
//	@param message
//	@param ttl
//	@return error
func (h *Hub) pushMessage(receiverId string, message models.PrivateMessage, ttl int) error {
	// Marshall JSON
	m, err := message.MarshalBinary()
	if err != nil {
		return err
	}
	client := GetSingleton().Valkey
	counterKey := queueCounterPrefix + receiverId
	// Count first, so a full queue rejects the message before it is written.
	queued := int64(-1)
	if maxQueued := int64(GetSingleton().Config.MessageMaxQueued); maxQueued > 0 {
		queued, err = client.Do(context.Background(), client.B().Incr().Key(counterKey).Build()).AsInt64()
		if err != nil {
			return err
		}
		if err := client.Do(context.Background(), client.B().Expire().Key(counterKey).Seconds(int64(ttl)).Build()).Error(); err != nil {
			slog.Warn("failed to set queue counter ttl", "error", SanitizeLogValue(err.Error()))
		}
		if queued > maxQueued {
			_, _ = client.Do(context.Background(), client.B().Decr().Key(counterKey).Build()).AsInt64()
			return errors.New(ErrorReceiverQueueFull)
		}
	}
	// Push
	key := models.OfflineQueueKey(receiverId, message.SenderId, message.SentAt)
	if err := client.Do(context.Background(), client.B().Set().Key(key).Value(string(m)).ExSeconds(int64(ttl)).Build()).Error(); err != nil {
		if queued > 0 {
			_, _ = client.Do(context.Background(), client.B().Decr().Key(counterKey).Build()).AsInt64()
		}
		return err
	}
	return nil
}

// Check if user with given ID will be deleted (O(1) with SET).
//
//	@param userId
//	@return bool
//	@return error
func (h *Hub) willUserBeDeleted(userId string) (bool, error) {
	client := GetSingleton().Valkey
	result, err := client.Do(context.Background(), client.B().Sismember().Key("del:users").Member(userId).Build()).AsBool()
	if err != nil {
		return false, err
	}
	return result, nil
}

// Deletes users and sends delete message. Returns slice of online users that have been deleted.
//
//	@param ids
func (h *Hub) DeleteUsers(ids []string) []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	var online = []string{}
	for conn := range h.Connections {
		// Find online user
		if slices.Contains(ids, conn.UserId) {
			// Delete user
			err := GetSingleton().Sqlite.Where("id = ?", conn.UserId).Delete(&models.User{}).Error
			if err != nil {
				continue
			}
			online = append(online, conn.UserId)
			// Send delete message and drop the connection: the client cannot
			// decide to stay online.
			response := models.NewWsResponse(models.WsResponseTypeDeleteAccount, nil)
			conn.Send(response)
			conn.Close()
		}
	}
	return online
}

// SendPushNotification
//
//	@param userId
//	@param title
//	@param body
//	@return error
func (h *Hub) SendPushNotification(userId string, title, body string) error {
	client := GetSingleton().Valkey
	// Get Firebase Installation ID by userId
	fid, err := client.Do(context.Background(), client.B().Get().Key(fmt.Sprintf("fcm:%s", userId)).Build()).AsBytes()
	if err != nil {
		// No push target (e.g. desktop clients never register one) is not an error
		var valkeyErr *valkey.ValkeyError
		if errors.As(err, &valkeyErr) {
			return nil
		}
		return fmt.Errorf("failed to get push target for user %s: %w", userId, err)
	}
	if len(fid) == 0 {
		return nil
	}
	// Message.Token accepts only FCM tokens, FIDs must go through multicast
	resp, err := GetSingleton().FirebaseMsg.SendEachForMulticast(context.Background(), &messaging.MulticastMessage{
		Notification: &messaging.Notification{
			Title: title,
			Body:  body,
		},
		Fids: []string{string(fid)},
	})
	if err != nil {
		return fmt.Errorf("failed to send push notification to user %s: %w", userId, err)
	}
	if resp.FailureCount > 0 && len(resp.Responses) > 0 {
		return fmt.Errorf("failed to send push notification to user %s: %v", userId, resp.Responses[0].Error)
	}
	return nil
}
