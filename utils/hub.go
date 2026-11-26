package utils

import (
	"jsfraz/shept-server/models"
	"sync"
)

// Hub is the registry of live WebSocket connections. It holds no database
// state: validation, archiving and push live in the handlers, so this package
// never has to import the storage layer.
type Hub struct {
	// Map of active WebSocket connections and their status
	Connections map[*WSConnection]bool
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
		Register:    make(chan *WSConnection, 16),
		Unregister:  make(chan *WSConnection, 16),
	}
}

// Run starts the Hub's main event loop to handle connections.
//
// Nothing in here may hold the connection map lock across network I/O: a single
// slow client would otherwise stop delivery for every user.
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
			delete(h.Connections, conn)
			h.mu.Unlock()
			conn.Close()
		}
	}
}

// ConnectionOf returns the live connection of one device, if any.
//
//	@param userId
//	@param deviceId
//	@return *WSConnection
func (h *Hub) ConnectionOf(userId string, deviceId string) *WSConnection {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for conn := range h.Connections {
		if conn.UserId == userId && conn.DeviceId == deviceId {
			return conn
		}
	}
	return nil
}

// ConnectionsOf returns every live connection of a user (one per device).
//
//	@param userId
//	@return []*WSConnection
func (h *Hub) ConnectionsOf(userId string) []*WSConnection {
	h.mu.RLock()
	defer h.mu.RUnlock()
	conns := []*WSConnection{}
	for conn := range h.Connections {
		if conn.UserId == userId {
			conns = append(conns, conn)
		}
	}
	return conns
}

// Online reports whether at least one device of a user is connected.
//
//	@param userId
//	@return bool
func (h *Hub) Online(userId string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for conn := range h.Connections {
		if conn.UserId == userId {
			return true
		}
	}
	return false
}

// SendToUser hands a response to every live device of a user.
//
//	@param userId
//	@param response
func (h *Hub) SendToUser(userId string, response models.WsResponse) {
	for _, conn := range h.ConnectionsOf(userId) {
		conn.Send(response)
	}
}

// CloseDevice drops every socket of one device. A revoked device must not keep
// receiving messages until its token happens to expire.
//
//	@param userId
//	@param deviceId
//	@param reason
func (h *Hub) CloseDevice(userId string, deviceId string, reason string) {
	h.mu.RLock()
	conns := []*WSConnection{}
	for conn := range h.Connections {
		if conn.UserId == userId && conn.DeviceId == deviceId {
			conns = append(conns, conn)
		}
	}
	h.mu.RUnlock()
	for _, conn := range conns {
		if reason != "" {
			conn.Send(models.NewWsResponse(models.WsResponseTypeError, reason))
		}
		conn.Close()
	}
}

// CloseUser tells every device of a user that the account is gone, then drops
// the sockets.
//
//	@param userId
func (h *Hub) CloseUser(userId string) {
	h.mu.RLock()
	conns := []*WSConnection{}
	for conn := range h.Connections {
		if conn.UserId == userId {
			conns = append(conns, conn)
		}
	}
	h.mu.RUnlock()
	for _, conn := range conns {
		conn.Send(models.NewWsResponse(models.WsResponseTypeDeleteAccount, nil))
		conn.Close()
	}
}
