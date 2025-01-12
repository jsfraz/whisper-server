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

func (c *WSConnection) Subscribe(topic models.Topic) {
	c.Topics[string(topic)] = true
}

func (c *WSConnection) Unsubscribe(topic models.Topic) {
	delete(c.Topics, string(topic))
}

func (c *WSConnection) isSubscribed(topic models.Topic) bool {
	return c.Topics[string(topic)]
}

func (c *WSConnection) send(message models.Message) {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	binaryMessage, _ := models.MarshalMessage(message)
	c.Conn.WriteMessage(websocket.BinaryMessage, binaryMessage)
}
