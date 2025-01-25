package utils

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"jsfraz/whisper-server/models"
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
					// log.Println(err)
					h.SendError(msgSenderPair.SenderId, err)
					h.mu.RUnlock()
					continue
				}
				// Validate private message
				validator := validator.New()
				err = validator.Struct(privateMessage)
				if err != nil {
					//log.Println(err)
					h.SendError(msgSenderPair.SenderId, err)
					h.mu.RUnlock()
					continue
				}
				// Check wheteher user is sending message to self
				if msgSenderPair.SenderId == privateMessage.ReceiverId {
					err = errors.New("can not send message to self")
					// log.Println(err)
					h.SendError(msgSenderPair.SenderId, err)
					h.mu.RUnlock()
					continue
				}
				// Check if user exists
				exists, err := userExistsById(msgSenderPair.SenderId)
				if err != nil {
					//log.Println(err)
					h.SendError(msgSenderPair.SenderId, err)
					h.mu.RUnlock()
					continue
				}
				if !exists {
					//log.Println(err)
					h.SendError(msgSenderPair.SenderId, errors.New("user does not exist"))
					h.mu.RUnlock()
					continue
				}
				// Send message to connected client with receiverId
				online := false
				pm := models.NewPrivateMessage(msgSenderPair.SenderId, privateMessage.Message, privateMessage.SentAt)
				for conn := range h.Connections {
					if conn.UserId == privateMessage.ReceiverId {
						conn.send(models.NewWsResponse(models.WsResponseTypeMessages, []models.PrivateMessage{pm}))
						online = true
						break
					}
				}
				if !online {
					// Push message to Valkey
					pushMessage(privateMessage.ReceiverId, pm, GetSingleton().Config.MessageTtl)
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
func (h *Hub) SendError(senderId uint64, err error) {
	response := models.NewWsResponse(models.WsResponseTypeError, err.Error())
	for conn := range h.Connections {
		if conn.UserId == senderId {
			conn.send(response)
			break
		}
	}
}

// Check if user exists by ID.
//
//	@param userId
//	@return bool
//	@return error
func userExistsById(userId uint64) (bool, error) {
	var count int64
	err := GetSingleton().Postgres.Model(&models.User{}).Where("id = ?", userId).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count == 1, nil
}

// Push PrivateMessage to Valkey.
//
//	@param code
//	@param message
//	@param ttl
//	@return error
func pushMessage(receiverId uint64, message models.PrivateMessage, ttl int) error {
	// Marshall JSON
	m, err := message.MarshalBinary()
	if err != nil {
		return err
	}
	// Push
	client := GetSingleton().ValkeyMessage
	return client.Do(context.Background(), client.B().Set().Key(fmt.Sprintf("%d_%s", receiverId, uuid.New().String())).Value(string(m)).ExSeconds(int64(ttl)).Build()).Error()
}
