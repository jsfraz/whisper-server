package utils

import (
	"testing"
	"time"

	"jsfraz/whisper-server/models"
)

// TestSendNeverBlocks is the property that keeps one stuck client from stopping
// message delivery for everyone: filling the send buffer must not block.
func TestSendNeverBlocks(t *testing.T) {
	conn := NewWSConnection(nil, "user", "device")
	defer conn.Close()

	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < wsSendBuffer*4; i++ {
			conn.Send(models.NewWsResponse(models.WsResponseTypeError, "frame"))
		}
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Send blocked with a full buffer and no reader")
	}
}

// TestSendAfterClose is a guard against panic on the shutdown path.
func TestSendAfterClose(t *testing.T) {
	conn := NewWSConnection(nil, "user", "device")
	conn.Close()
	conn.Close()
	conn.Send(models.NewWsResponse(models.WsResponseTypeError, "frame"))
}

// TestSendSyncReportsClosedConnection tells the caller to keep the message
// pending instead of assuming delivery.
func TestSendSyncReportsClosedConnection(t *testing.T) {
	conn := NewWSConnection(nil, "user", "device")
	conn.Close()
	if err := conn.SendSync(models.NewWsResponse(models.WsResponseTypeError, "frame")); err == nil {
		t.Fatal("SendSync on a closed connection returned no error")
	}
}
