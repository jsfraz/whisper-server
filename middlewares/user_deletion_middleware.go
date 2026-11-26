package middlewares

import (
	"jsfraz/shept-server/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Checks whether user should be deleted. If yes, returns status 401.
//
//	@param c Gin context
func UserDeletionMiddleware(c *gin.Context) {
	userId, _ := c.Get("userId")
	// Check if user is in delete list
	toDelete, err := willUserBeDeleted(userId.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": utils.ErrorInternal})
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	// User should be deleted
	if toDelete {
		c.JSON(http.StatusUnauthorized, gin.H{"error": utils.ErrorAccountWillBeDeleted})
		// err is nil here: AbortWithError would panic on it.
		c.Abort()
		return
	}
	c.Next()
}
