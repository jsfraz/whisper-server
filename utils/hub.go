package utils

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"jsfraz/whisper-server/models"
	"slices"
	"sync"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

// Hub manages WebSocket connections and message broadcasting
type Hub struct {
	// Map of active WebSocket connections and their status
	Connections map[*WSConnection]bool
	// Channel for broadcasting messages to all subscribed connections
	// Anonymous struct is here for pairing senderId and message
	Broadcast chan struct {
		SenderId uint64
		Message  models.WsMessage
	}
	// Channel for registering new WebSocket connections
	Register chan *WSConnection
	// Channel for unregistering (removing) WebSocket connections
	Unregister chan *WSConnection
	// Mutex for thread-safe access to the Connections map
	mu sync.RWMutex
}

// NewHub creates and initializes a new Hub instance
//
//	@return *Hub
func NewHub() *Hub {
	return &Hub{
		Connections: make(map[*WSConnection]bool),
		Broadcast: make(chan struct {
			SenderId uint64
			Message  models.WsMessage
		}),
		Register:   make(chan *WSConnection),
		Unregister: make(chan *WSConnection),
	}
}

// Run starts the Hub's main event loop to handle connections and messages
func (h *Hub) Run() {
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
			if _, ok := h.Connections[conn]; ok {
				delete(h.Connections, conn)
			}
			h.mu.Unlock()

		// Handle incoming message
		case msgSenderPair := <-h.Broadcast:
			h.mu.RLock()

			switch msgSenderPair.Message.Type {

			// Private message
			case models.WsMessageTopicMessage:
				// Unmarshall private message
				var privateMessage models.NewPrivateMessageReceive
				err := json.Unmarshal(msgSenderPair.Message.Payload, &privateMessage)
				if err != nil {
					h.sendError(msgSenderPair.SenderId, err)
					continue
				}
				// Validate private message
				validator := validator.New()
				err = validator.Struct(privateMessage)
				if err != nil {
					h.sendError(msgSenderPair.SenderId, err)
					continue
				}
				// Check wheteher user is sending message to self
				if msgSenderPair.SenderId == privateMessage.ReceiverId {
					err = errors.New("can not send message to self")
					h.sendError(msgSenderPair.SenderId, err)
					continue
				}
				// Check if message is being sent to user that will be deleted
				toDelete, err := h.willUserBeDeleted(privateMessage.ReceiverId)
				if err != nil {
					h.sendError(msgSenderPair.SenderId, err)
					continue
				}
				if toDelete {
					err = errors.New("can not send message to this user")
					h.sendError(msgSenderPair.SenderId, err)
					continue
				}
				// Check if user exists
				exists, err := h.userExistsById(privateMessage.ReceiverId)
				if err != nil {
					h.sendError(msgSenderPair.SenderId, err)
					continue
				}
				if !exists {
					h.sendError(msgSenderPair.SenderId, errors.New("user does not exist"))
					continue
				}
				// Send message to connected client with receiverId
				online := false
				pm := models.NewPrivateMessage(msgSenderPair.SenderId, privateMessage.Message, privateMessage.SentAt)
				for conn := range h.Connections {
					if conn.UserId == privateMessage.ReceiverId {
						conn.Send(models.NewWsResponse(models.WsResponseTypeMessages, []models.PrivateMessage{pm}))
						online = true
						break
					}
				}
				if !online {
					// Push message to Valkey
					h.pushMessage(privateMessage.ReceiverId, pm, GetSingleton().Config.MessageTtl)
				}
			}

			h.mu.RUnlock()
		}
	}
}

// Send error response to sender
//
//	@param senderId
//	@param err
func (h *Hub) sendError(senderId uint64, err error) {
	//log.Println(err)
	for conn := range h.Connections {
		if conn.UserId == senderId {
			conn.SendError(err)
			h.mu.RUnlock()
			break
		}
	}
}

// Check if user exists by ID.
//
//	@param userId
//	@return bool
//	@return error
func (h *Hub) userExistsById(userId uint64) (bool, error) {
	var count int64
	err := GetSingleton().Postgres.Model(&models.User{}).Where("id = ?", userId).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count == 1, nil
}

// Push PrivateMessage to Valkey.
//
//	@param receiverId
//	@param message
//	@param ttl
//	@return error
func (h *Hub) pushMessage(receiverId uint64, message models.PrivateMessage, ttl int) error {
	// Marshall JSON
	m, err := message.MarshalBinary()
	if err != nil {
		return err
	}
	// Push
	client := GetSingleton().ValkeyMessage
	return client.Do(context.Background(), client.B().Set().Key(fmt.Sprintf("%d_%s", receiverId, uuid.New().String())).Value(string(m)).ExSeconds(int64(ttl)).Build()).Error()
}

// Check if user with given ID will be deleted.
//
//	@param userId
//	@return bool
//	@return error
func (h *Hub) willUserBeDeleted(userId uint64) (bool, error) {
	client := GetSingleton().ValkeyDelUser
	// Check len
	length, err := client.Do(context.Background(), client.B().Llen().Key("delete").Build()).AsInt64()
	if err != nil {
		return false, err
	}
	// Get all IDs
	ids, err := client.Do(context.Background(), client.B().Lrange().Key("delete").Start(0).Stop(length).Build()).AsIntSlice()
	if err != nil {
		return false, err
	}
	result := make([]uint64, len(ids))
	for i, v := range ids {
		result[i] = uint64(v)
	}
	// Check slice
	return slices.Contains(result, userId), nil
}

// Deletes users and sends delete message. Returns slice of online users that have been deleted.
//
//	@param ids
func (h *Hub) DeleteUsers(ids []uint64) []uint64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	var online []uint64 = []uint64{}
	for conn := range h.Connections {
		// Find online user
		if slices.Contains(ids, conn.UserId) {
			// Delete user
			err := GetSingleton().Postgres.Where("id = ?", conn.UserId).Delete(&models.User{}).Error
			if err != nil {
				continue
			}
			online = append(online, conn.UserId)
			// Send delete message
			response := models.NewWsResponse(models.WsResponseTypeDeleteAccount, nil)
			conn.Send(response)
		}
	}
	return online
}
