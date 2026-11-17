package handlers

import (
	"jsfraz/whisper-server/database"
	"jsfraz/whisper-server/models"

	"github.com/gin-gonic/gin"
)

// Sets client Firebase token.
//
//	@param c
//	@param request
//	@return error
func SetFirebaseToken(c *gin.Context, request *models.SetFirebaseTokenRequest) error {
	userId, _ := c.Get("userId")
	deviceId, _ := c.Get("deviceId")
	// Push Firebase token to database, per device
	err := database.PushFirebaseUserToken(userId.(string), deviceId.(string), request.Token)
	if err != nil {
		return c.AbortWithError(500, err)
	}
	return nil
}
