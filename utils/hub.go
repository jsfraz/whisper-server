package utils

import (
	"encoding/json"
	"errors"
	"sync"

	"jsfraz/whisper-server/models"

	"github.com/go-playground/validator/v10"
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
				var privateMessage models.PrivateMessageReceive
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
				// Send message to connected client with receiverId
				online := false
				for conn := range h.Connections {
					if conn.UserId == privateMessage.ReceiverId {
						conn.send(models.NewWsResponse(models.WsResponseTypeMessage, models.NewPrivateMessage(msgSenderPair.SenderId, privateMessage.Message)))
						online = true
						break
					}
				}
				if !online {
					// TODO upload message to redis
				}
			}

			h.mu.RUnlock()
		}
	}
}

// Send error response to sender
func (h *Hub) SendError(senderId uint64, err error) {
	response := models.NewWsResponse(models.WsResponseTypeError, err.Error())
	for conn := range h.Connections {
		if conn.UserId == senderId {
			conn.send(response)
			break
		}
	}
}
