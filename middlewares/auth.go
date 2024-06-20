package middlewares

import (
	"jsfraz/whisper-server/database"
	"jsfraz/whisper-server/utils"
	"net/http"
	"os"

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
	userId, err := utils.TokenValid(utils.ExtractTokenFromContext(c), os.Getenv("ACCESS_TOKEN_SECRET"))
	// Invalid token
	if err != nil {
		c.AbortWithError(http.StatusUnauthorized, err)
	}
	// Check if user exists
	exists, err := database.UserExists(userId)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
	}
	// User does not exist.
	if !exists {
		c.AbortWithError(http.StatusUnauthorized, err)
	}
	// Token is valid, set it to context and continue
	c.Set("userId", userId)
	c.Next()
}
