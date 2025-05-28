package middlewares

import (
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
func AuthMiddleware(c *gin.Context) {
	// Get access token from context and check it
	userId, _, err := utils.TokenValid(utils.ExtractTokenFromContext(c), utils.GetSingleton().Config.AccessTokenSecret)
	// Invalid token
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		c.AbortWithError(http.StatusUnauthorized, err)
		return
	}
	// Check if user exists
	exists, err := database.UserExistsById(userId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	// User does not exist.
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user does not exist"})
		c.AbortWithError(http.StatusUnauthorized, err)
		return
	}
	// Token is valid, set it to context and continue
	c.Set("userId", userId)
	c.Next()
}
