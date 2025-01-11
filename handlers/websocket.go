package handlers

import (
	"encoding/json"
	"jsfraz/whisper-server/models"
	"jsfraz/whisper-server/utils"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// WebSocketHandler handles incoming WebSocket connections and their lifecycle
func WebSocketHandler(c *gin.Context) {
	// Configure WebSocket upgrader
	upgrader := websocket.Upgrader{
		CheckOrigin: nil,
	}

	// Upgrade HTTP connection to WebSocket
	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Create new WebSocket connection with topic subscription support
	conn := &utils.WSConnection{
		Conn:   ws,
		Topics: make(map[string]bool),
	}

	// Register new connection with the hub
	utils.GetSingleton().Hub.Register <- conn

	// Handle incoming messages
	go func() {
		// Ensure cleanup on connection close
		defer func() {
			utils.GetSingleton().Hub.Unregister <- conn
			conn.Conn.Close()
		}()

		for {
			// Message structure for WebSocket communication
			var msg struct {
				Action  string          `json:"action"` // Supported actions: "subscribe", "unsubscribe", "publish"
				Topic   string          `json:"topic"`
				Payload json.RawMessage `json:"payload,omitempty"`
			}

			// Read and parse incoming message
			err := conn.Conn.ReadJSON(&msg)
			if err != nil {
				break
			}

			// Process message based on action type
			switch msg.Action {
			case "subscribe":
				conn.Subscribe(msg.Topic)
			case "unsubscribe":
				conn.Unsubscribe(msg.Topic)
			case "publish":
				utils.GetSingleton().Hub.Broadcast <- models.Message{
					Topic:   msg.Topic,
					Payload: msg.Payload,
				}
			}
		}
	}()
}
