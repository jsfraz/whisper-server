package middlewares

import (
	"errors"
	"jsfraz/whisper-server/database"
	"jsfraz/whisper-server/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Middleware for user authentication.
//
// If the user has a valid access token, it sets its ID in the context.
// If it is not valid, it returns a status of 401.
//
//	@param c Gin context
func Auth(c *gin.Context) {
	// Get access token from context and check it
	userId, err := utils.TokenValid(utils.ExtractTokenFromContext(c), utils.GetSingleton().Config.AccessTokenSecret)
	// Invalid token
	if err != nil {
		c.AbortWithStatus(http.StatusUnauthorized)
		c.Error(err)
		return
	}
	// Check if user exists
	exists, err := database.UserExistsById(userId)
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		c.Error(err)
		return
	}
	// User does not exist.
	if !exists {
		c.AbortWithStatus(http.StatusUnauthorized)
		c.Error(errors.New("user does not exist"))
		return
	}
	// Token is valid, set it to context and continue
	c.Set("userId", userId)
	c.Next()
}
