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
	// Push Firebase token to database
	err := database.PushFirebaseUserToken(userId.(uint64), request.Token)
	if err != nil {
		return c.AbortWithError(500, err)
	}
	return nil
}
