package utils

import (
	"sync"

	"jsfraz/whisper-server/models"
)

// Hub manages WebSocket connections and message broadcasting
type Hub struct {
	// Map of active WebSocket connections and their status
	Connections map[*WSConnection]bool
	// Channel for broadcasting messages to all subscribed connections
	Broadcast chan models.Message
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
		Broadcast:   make(chan models.Message),
		Register:    make(chan *WSConnection),
		Unregister:  make(chan *WSConnection),
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

		// Handle message broadcasting to subscribed connections
		case message := <-h.Broadcast:
			h.mu.RLock()
			for conn := range h.Connections {
				// Send message only to connections subscribed to the message topic
				if conn.isSubscribed(message.Topic) {
					conn.send(message)
				}
			}
			h.mu.RUnlock()
		}
	}
}
