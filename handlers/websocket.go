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
		c.JSON(http.StatusBadRequest, gin.H{"error": utils.ErrorAccessTokenRequired})
		slog.Warn("websocket auth missing token")
		return
	}
	// Validate access token
	claims, err := utils.TokenValid(accessToken, utils.GetSingleton().Config.WsAccessTokenSecret, utils.TokenTypeWs)
	if err != nil || claims.TokenId == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": utils.ErrorInvalidToken})
		return
	}
	// Check if token exists in Valkey
	exists, accessTokenById, err := database.WsAccessTokenExists(*claims.TokenId)
	if err != nil {
		slog.Error("failed to load ws access token", "error", utils.SanitizeLogValue(err.Error()))
		c.JSON(http.StatusInternalServerError, gin.H{"error": utils.ErrorInternal})
		return
	}
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": utils.ErrorAccessTokenNotFound})
		return
	}
	// Check if provided token and token from Valkey are the same
	if accessToken != accessTokenById {
		c.JSON(http.StatusUnauthorized, gin.H{"error": utils.ErrorAccessTokenMismatch})
		return
	}
	// A revoked device has no live socket, even with an unexpired ws token.
	active, err := database.IsDeviceActive(claims.UserId, claims.DeviceId)
	if err != nil {
		slog.Error("failed to load device", "error", utils.SanitizeLogValue(err.Error()))
		c.JSON(http.StatusInternalServerError, gin.H{"error": utils.ErrorInternal})
		return
	}
	if !active {
		c.JSON(http.StatusUnauthorized, gin.H{"error": utils.ErrorDeviceRevoked})
		return
	}

	// Configure WebSocket upgrader
	upgrader := websocket.Upgrader{
		CheckOrigin: nil, // No need to check origin when users connect from the mobile app
	}
	// Upgrade HTTP connection to WebSocket
	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		slog.Warn("ws upgrade failed", "error", utils.SanitizeLogValue(err.Error()))
		c.JSON(http.StatusInternalServerError, gin.H{"error": utils.ErrorInternal})
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
	// stalled client can never block delivery.
	conn := utils.NewWSConnection(ws, claims.UserId, claims.DeviceId)
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

		// Nothing is pushed on connect: the client asks for its delta with a
		// sync frame carrying its cursor, so a phone that was away for a week
		// fetches only what it missed.

		// One connection may not flood the shared server.
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
				conn.SendError(errors.New(utils.ErrorTooManyMessages))
				return
			}

			// Check message type
			if messageType != websocket.BinaryMessage {
				err = errors.New(utils.ErrorInvalidMessageType)
				conn.SendError(err)
				continue
			}

			// WebSocket message
			var msg models.WsMessage
			err = json.Unmarshal(payload, &msg)
			if err != nil {
				conn.SendError(errors.New(utils.ErrorInvalidMessage))
				continue
			}

			// Validate message
			err = wsValidator.Struct(msg)
			if err != nil {
				conn.SendError(errors.New(utils.ErrorInvalidMessage))
				continue
			}

			// Archiving touches the database and a slow write must not stall
			// the read loop of the other devices, so each frame is handled in
			// its own goroutine.
			go ProcessWsMessage(conn, msg)
		}
	}()
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
		if err := database.RemoveDeletedUserId(conn.UserId); err != nil {
			slog.Error("failed to clear deleted user id", "user_id", utils.SanitizeLogValue(conn.UserId), "error", utils.SanitizeLogValue(err.Error()))
		}
		// Delete messages from the archive
		if err := database.DeleteUserMessages(conn.UserId); err != nil {
			slog.Error("failed to delete user messages", "user_id", utils.SanitizeLogValue(conn.UserId), "error", utils.SanitizeLogValue(err.Error()))
		}
		// Delete devices
		if err := database.DeleteUserDevices(conn.UserId); err != nil {
			slog.Error("failed to delete user devices", "user_id", utils.SanitizeLogValue(conn.UserId), "error", utils.SanitizeLogValue(err.Error()))
		}
		// Delete media from server
		if err := database.DeleteUserMedia(conn.UserId); err != nil {
			slog.Error("failed to delete user media", "user_id", utils.SanitizeLogValue(conn.UserId), "error", utils.SanitizeLogValue(err.Error()))
		}
	}
	return toDelete, nil
}
