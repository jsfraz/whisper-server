package handlers

import (
	"jsfraz/whisper-server/database"
	"jsfraz/whisper-server/models"
	"jsfraz/whisper-server/utils"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Get short life access token for WebSocket.
//
//	@param c
//	@return *models.WsAuthResponse
//	@return error
func WebSocketAuth(c *gin.Context) (*models.WsAuthResponse, error) {
	userId, _ := c.Get("userId")
	// Generate access token
	tokenId := uuid.New().String()
	accessToken, err := utils.GenerateToken(userId.(uint64), utils.GetSingleton().Config.WsAccessTokenLifespan, utils.GetSingleton().Config.WsAccessTokenSecret, &tokenId)
	if err != nil {
		return nil, c.AbortWithError(http.StatusInternalServerError, err)
	}
	// Insert token to Valkey
	err = database.PushWsAccessToken(tokenId, accessToken, utils.GetSingleton().Config.WsAccessTokenLifespan)
	if err != nil {
		return nil, c.AbortWithError(http.StatusInternalServerError, err)
	}
	return models.NewWsAuthResponse(accessToken), nil
}
