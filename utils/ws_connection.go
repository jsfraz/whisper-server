package utils

import (
	"log"
	"sync"

	"jsfraz/whisper-server/models"

	"github.com/gorilla/websocket"
)

type WSConnection struct {
	Conn    *websocket.Conn
	UserId  uint64
	writeMu sync.Mutex
}

// Send WsResponse
//
//	@param message
func (c *WSConnection) Send(message models.WsResponse) {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	binaryMessage, err := models.MarshalWsResponse(message)
	if err != nil {
		log.Printf("failed to marshal ws response: %v", err)
		return
	}
	if err := c.Conn.WriteMessage(websocket.BinaryMessage, binaryMessage); err != nil {
		log.Printf("failed to write ws message: %v", err)
	}
}

// Send error as WsResponse
//
//	@param err
func (c *WSConnection) SendError(err error) {
	c.Send(models.NewWsResponse(models.WsResponseTypeError, err.Error()))
}
