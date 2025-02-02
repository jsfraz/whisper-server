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
		// log.Println(err.Error())
		return
	}
	// Check if token exists in Redis
	exists, accessTokenById, err := database.WsAccessTokenExists(*tokenId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		// log.Println(err.Error())
		return
	}
	if !exists {
		err = errors.New("access token not found")
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		// log.Println(err.Error())
		return
	}
	// Check if provided token and token from Redis are the same
	if accessToken != accessTokenById {
		err = errors.New("access token mismatch")
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		// log.Println(err.Error())
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

		// Check if account should be deleted
		toDelete, err := checkAccountDeletion(conn)
		if err != nil {
			conn.SendError(err)
		}
		// Terminates goroutine
		if toDelete {
			return
		}

		// Send messages from cache
		sendMessagesFromCache(conn)

		// Register custom validators
		validator := validator.New()
		validator.RegisterValidation("type", models.ValidateWsMessageType)

		for {
			// Read and parse incoming message
			messageType, payload, err := conn.Conn.ReadMessage()
			if err != nil {
				/*
					if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
						log.Println(err)
					}
				*/
				return // Terminates goroutine on read error
			}

			// Check message type
			if messageType != websocket.BinaryMessage {
				err = errors.New("invalid message type, only binary messages are supported")
				conn.SendError(err)
				continue
			}

			// WebSocket message
			var msg models.WsMessage
			err = json.Unmarshal(payload, &msg)
			if err != nil {
				conn.SendError(err)
				continue
			}

			// Validate message
			err = validator.Struct(msg)
			if err != nil {
				conn.SendError(err)
				continue
			}

			// Process message based on action type
			utils.GetSingleton().Hub.Broadcast <- struct {
				SenderId uint64
				Message  models.WsMessage
			}{
				SenderId: conn.UserId,
				Message:  msg,
			}
		}
	}()
}

// Send messages from cache.
func sendMessagesFromCache(conn *utils.WSConnection) {
	// Get messages from cache
	messages, err := database.GetUserPrivateMessages(conn.UserId)
	if err != nil {
		conn.SendError(err)
		return
	}
	// Send messages
	if len(*messages) > 0 {
		response := models.NewWsResponse(models.WsResponseTypeMessages, messages)
		conn.Send(response)
	}
}

// Checks if user account should be deleted, sends delete message and deletes user.
//
//	@param conn
//	@return bool
//	@return error
func checkAccountDeletion(conn *utils.WSConnection) (bool, error) {
	toDelete, err := database.WillUserBeDeleted(conn.UserId)
	if err != nil {
		return false, err
	}
	if toDelete {
		// Delete user
		err = database.DeleteUserById(conn.UserId)
		if err != nil {
			return toDelete, err
		}
		// Send delete message
		response := models.NewWsResponse(models.WsResponseTypeDeleteAccount, nil)
		conn.Send(response)
		// Delete ID from Valkey
		database.RemoveDeletedUserId(conn.UserId)
		// Delete messages from Valkey
		database.DeleteUserPrivateMessages(conn.UserId)

	}
	return toDelete, nil
}
