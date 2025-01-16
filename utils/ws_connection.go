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

func (c *WSConnection) send(message models.WsResponse) {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	binaryMessage, _ := models.MarshalWsResponse(message)
	c.Conn.WriteMessage(websocket.BinaryMessage, binaryMessage)
}
