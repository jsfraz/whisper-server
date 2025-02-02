package utils

import (
	"sync"

	"jsfraz/whisper-server/models"

	"github.com/gorilla/websocket"
)

type WSConnection struct {
	Conn    *websocket.Conn
	UserId  uint64
	writeMu sync.Mutex
}

// Send WsRespons
//
//	@param message
func (c *WSConnection) Send(message models.WsResponse) {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	binaryMessage, _ := models.MarshalWsResponse(message)
	c.Conn.WriteMessage(websocket.BinaryMessage, binaryMessage)
}

// Send error as WsResponse
//
//	@param err
func (c *WSConnection) SendError(err error) {
	c.Send(models.NewWsResponse(models.WsResponseTypeError, err.Error()))
}
