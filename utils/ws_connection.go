package utils

import (
	"errors"
	"log/slog"
	"sync"
	"time"

	"jsfraz/whisper-server/models"

	"github.com/gorilla/websocket"
)

const (
	// wsWriteTimeout bounds one frame write. A peer that stops reading must not
	// hold the hub, which is why writes carry a deadline and run in the
	// connection's own goroutine.
	wsWriteTimeout = 10 * time.Second
	// wsPingInterval keeps half-open connections detectable; a dead peer is
	// removed instead of looking online forever.
	wsPingInterval = 30 * time.Second
	// wsSendBuffer is how many frames may queue for one client. A client that
	// falls behind loses frames (and then reconnects) instead of blocking
	// delivery for everyone else.
	wsSendBuffer = 64
)

// outgoing is one frame queued for a connection. done, when set, receives the
// write result so a caller can wait for the actual socket write.
type outgoing struct {
	data []byte
	done chan error
}

type WSConnection struct {
	Conn   *websocket.Conn
	UserId string

	out  chan outgoing
	done chan struct{}
	once sync.Once
}

// NewWSConnection wraps an upgraded socket. Call WriteLoop in its own goroutine
// before using Send.
func NewWSConnection(conn *websocket.Conn, userId string) *WSConnection {
	return &WSConnection{
		Conn:   conn,
		UserId: userId,
		out:    make(chan outgoing, wsSendBuffer),
		done:   make(chan struct{}),
	}
}

// WriteLoop is the only goroutine that writes to the socket. It also pings the
// peer and exits on the first write error.
func (c *WSConnection) WriteLoop() {
	ticker := time.NewTicker(wsPingInterval)
	defer ticker.Stop()
	for {
		select {
		case msg := <-c.out:
			err := c.write(msg.data)
			if msg.done != nil {
				msg.done <- err
			}
			if err != nil {
				slog.Warn("ws write failed", "user_id", SanitizeLogValue(c.UserId), "error", SanitizeLogValue(err.Error()))
				return
			}
		case <-ticker.C:
			if err := c.writeControl(websocket.PingMessage); err != nil {
				return
			}
		case <-c.done:
			return
		}
	}
}

func (c *WSConnection) write(data []byte) error {
	// Deadline per frame: a stalled peer fails fast instead of blocking forever.
	if err := c.Conn.SetWriteDeadline(time.Now().Add(wsWriteTimeout)); err != nil {
		return err
	}
	return c.Conn.WriteMessage(websocket.BinaryMessage, data)
}

func (c *WSConnection) writeControl(messageType int) error {
	if err := c.Conn.SetWriteDeadline(time.Now().Add(wsWriteTimeout)); err != nil {
		return err
	}
	return c.Conn.WriteMessage(messageType, nil)
}

// Send queues a frame and never blocks. Frames are dropped when the client is
// not reading; the connection is closed by the read deadline instead of holding
// the message loop.
//
//	@param message
func (c *WSConnection) Send(message models.WsResponse) {
	data, err := models.MarshalWsResponse(message)
	if err != nil {
		slog.Error("failed to marshal ws response", "error", err)
		return
	}
	select {
	case c.out <- outgoing{data: data}:
	case <-c.done:
	default:
		slog.Warn("ws send buffer full, dropping frame", "user_id", SanitizeLogValue(c.UserId))
	}
}

// SendSync queues a frame and waits until it reaches the socket, returning the
// write error. Used where delivery has to be known before the next step.
//
//	@param message
//	@return error
func (c *WSConnection) SendSync(message models.WsResponse) error {
	data, err := models.MarshalWsResponse(message)
	if err != nil {
		return err
	}
	done := make(chan error, 1)
	select {
	case c.out <- outgoing{data: data, done: done}:
	case <-c.done:
		return errors.New("connection closed")
	}
	select {
	case err := <-done:
		return err
	case <-c.done:
		return errors.New("connection closed")
	}
}

// Send error as WsResponse
//
//	@param err
func (c *WSConnection) SendError(err error) {
	c.Send(models.NewWsResponse(models.WsResponseTypeError, err.Error()))
}

// Close stops the write loop and closes the socket. It is safe to call twice.
func (c *WSConnection) Close() {
	c.once.Do(func() {
		close(c.done)
		if c.Conn != nil {
			_ = c.Conn.Close()
		}
	})
}
