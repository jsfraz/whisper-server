package handlers

import (
	"encoding/json"
	"errors"
	"jsfraz/whisper-server/database"
	"jsfraz/whisper-server/models"
	"jsfraz/whisper-server/utils"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/gorilla/websocket"
)

const (
	// wsReadTimeout is how long a connection may stay silent before it is
	// considered dead. The client answers the server ping, which extends it.
	wsReadTimeout = 60 * time.Second
	// wsReadLimit bounds one inbound frame; the largest payload is an encrypted
	// 25 MiB attachment envelope, which stays far below this.
	wsReadLimit = 64 << 10
)

// Handles incoming WebSocket connections and their lifecycle
//
//	@param c
func WebSocketHandler(c *gin.Context) {
	// The token belongs in the Authorization header: a URL query ends up in
	// proxy access logs. The query is still read for older clients.
	accessToken := utils.ExtractTokenFromContext(c)
	if accessToken == "" {
		accessToken = c.Query("wsAccessToken")
	}
	if accessToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "access token is required"})
		slog.Warn("websocket auth missing token")
		return
	}
	// Validate access token
	userId, tokenId, err := utils.TokenValid(accessToken, utils.GetSingleton().Config.WsAccessTokenSecret, utils.TokenTypeWs)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	// Check if token exists in Valkey
	exists, accessTokenById, err := database.WsAccessTokenExists(*tokenId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if !exists {
		err = errors.New("access token not found")
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	// Check if provided token and token from Valkey are the same
	if accessToken != accessTokenById {
		err = errors.New("access token mismatch")
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
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

	// Bound the memory one client can make the server allocate, and drop
	// connections that stop answering.
	ws.SetReadLimit(wsReadLimit)
	if err = ws.SetReadDeadline(time.Now().Add(wsReadTimeout)); err != nil {
		_ = ws.Close()
		return
	}
	ws.SetPongHandler(func(string) error {
		return ws.SetReadDeadline(time.Now().Add(wsReadTimeout))
	})

	// Create new WebSocket connection, writing runs in its own goroutine so a
	// stalled client can never block the hub.
	conn := utils.NewWSConnection(ws, userId)
	go conn.WriteLoop()
	// Register new connection with the hub
	utils.GetSingleton().Hub.Register <- conn

	// Handle incoming messages
	go func() {
		// Ensure cleanup on connection close
		defer func() {
			utils.GetSingleton().Hub.Unregister <- conn
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

		// Send messages from cache. They stay queued until the client
		// acknowledges them, so a crash before storing loses nothing.
		sendMessagesFromCache(conn)

		// Register custom validators
		validator := validator.New()
		validator.RegisterValidation("type", models.ValidateWsMessageType)
		// One connection may not flood the shared offline queues.
		limiter := utils.NewMessageRateLimiter(utils.GetSingleton().Config.WsMessageRate)

		for {
			// Read and parse incoming message
			messageType, payload, err := conn.Conn.ReadMessage()
			if err != nil {
				return // Terminates goroutine on read error
			}

			// Flood protection: close the connection instead of buffering
			// unbounded work for everyone.
			if !limiter.Allow() {
				conn.SendError(errors.New("too many messages"))
				return
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
				SenderId string
				Message  models.WsMessage
			}{
				SenderId: conn.UserId,
				Message:  msg,
			}
		}
	}()
}

// Send messages from cache. The entries are removed by the client's
// acknowledgement, not here: a failed write must not lose them.
func sendMessagesFromCache(conn *utils.WSConnection) {
	// Get messages from cache
	messages, err := database.GetUserPrivateMessages(conn.UserId)
	if err != nil {
		conn.SendError(err)
		return
	}
	// Send messages. The queue is bounded by wsSendBuffer, so a large backlog is
	// written with a wait per frame: dropping frames here would lose messages
	// that the client has not acknowledged yet.
	if len(*messages) > 0 {
		response := models.NewWsResponse(models.WsResponseTypeMessages, messages)
		if err := conn.SendSync(response); err != nil {
			slog.Warn("failed to send queued messages", "user_id", utils.SanitizeLogValue(conn.UserId), "error", utils.SanitizeLogValue(err.Error()))
		}
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
		// Delete media from server
		database.DeleteUserMedia(conn.UserId)

	}
	return toDelete, nil
}
