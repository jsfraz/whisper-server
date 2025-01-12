package handlers

import (
	"encoding/json"
	"errors"
	"jsfraz/whisper-server/database"
	"jsfraz/whisper-server/models"
	"jsfraz/whisper-server/utils"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
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
		err = errors.New("access token not found")
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		log.Println(err.Error())
		return
	}
	// Check if provided token and token from Redis are the same
	if accessToken != accessTokenById {
		err = errors.New("access token mismatch")
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		log.Println(err.Error())
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

		// Register custom validators
		validator := validator.New()
		validator.RegisterValidation("action", models.ValidateAction)
		validator.RegisterValidation("topic", models.ValidateTopic)

		for {
			// Read and parse incoming message
			messageType, payload, err := conn.Conn.ReadMessage()
			if err != nil {
				log.Println(err)
				response := models.NewResponse(models.ResponseTypeError, err.Error())
				binaryResponse, _ := models.MarshalResponse(response)
				conn.Conn.WriteMessage(websocket.BinaryMessage, binaryResponse)
				continue
			}

			// Check message type
			if messageType != websocket.BinaryMessage {
				err = errors.New("invalid message type, only binary messages are supported")
				log.Println(err)
				response := models.NewResponse(models.ResponseTypeError, err.Error())
				binaryResponse, _ := models.MarshalResponse(response)
				conn.Conn.WriteMessage(websocket.BinaryMessage, binaryResponse)
				continue
			}

			// WebSocket message
			var msg models.WsMessage
			err = json.Unmarshal(payload, &msg)
			if err != nil {
				log.Println(err)
				response := models.NewResponse(models.ResponseTypeError, err.Error())
				binaryResponse, _ := models.MarshalResponse(response)
				conn.Conn.WriteMessage(websocket.BinaryMessage, binaryResponse)
				continue
			}

			// Validate message
			err = validator.Struct(msg)
			if err != nil {
				log.Println(err)
				response := models.NewResponse(models.ResponseTypeError, err.Error())
				binaryResponse, _ := models.MarshalResponse(response)
				conn.Conn.WriteMessage(websocket.BinaryMessage, binaryResponse)
				continue
			}

			// Process message based on action type
			switch msg.Action {
			case models.ActionSubscribe:
				conn.Subscribe(msg.Topic)
			case models.ActionUnsubscribe:
				conn.Unsubscribe(msg.Topic)
			case models.ActionPublish:
				utils.GetSingleton().Hub.Broadcast <- models.Message{
					Topic:   msg.Topic,
					Payload: msg.Payload,
				}
			}
		}
	}()
}
