package utils

import (
	"sync"

	"jsfraz/whisper-server/models"

	"github.com/gorilla/websocket"
)

// WSConnection for managing WebSocket connections and subscriptions
type WSConnection struct {
	Conn    *websocket.Conn
	Topics  map[string]bool
	UserId  uint64
	writeMu sync.Mutex
}

func (c *WSConnection) Subscribe(topic string) {
	c.Topics[topic] = true
}

func (c *WSConnection) Unsubscribe(topic string) {
	delete(c.Topics, topic)
}

func (c *WSConnection) isSubscribed(topic string) bool {
	return c.Topics[topic]
}

func (c *WSConnection) send(message models.Message) {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	c.Conn.WriteJSON(message)
}
