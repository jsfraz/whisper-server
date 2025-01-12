package handlers

import (
	"encoding/json"
	"jsfraz/whisper-server/database"
	"jsfraz/whisper-server/models"
	"jsfraz/whisper-server/utils"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// Handles incoming WebSocket connections and their lifecycle
//
//	@param c
func WebSocketHandler(c *gin.Context) {
	// TODO redis for single use tokens
	// Get access token from request
	accessToken := c.Query("wsAccessToken")
	if accessToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "wsAccessToken is required"})
		log.Println("accessToken is required")
		return
	}
	// Validate access token
	userId, tokenId, err := utils.TokenValid(accessToken, utils.GetSingleton().Config.WsTokenSecret)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		log.Println(err.Error())
		return
	}
	// Check if token exists in Redis
	exists, accessTokenById, err := database.WsAccessTokenExists(*tokenId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		log.Println(err.Error())
		return
	}
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid access token"})
		log.Println("invalid access token")
		return
	}
	// Check if provided token and token from Redis are the same
	if accessToken != accessTokenById {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid access token"})
		log.Println("invalid access token")
		return
	}

	// Configure WebSocket upgrader
	upgrader := websocket.Upgrader{
		CheckOrigin: nil, // No need to check origin when users connect from the mobile app
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
		UserId: userId,
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
