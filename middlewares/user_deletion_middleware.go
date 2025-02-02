package middlewares

import (
	"errors"
	"jsfraz/whisper-server/database"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Checks whether user should be deleted. If yes, returns status 401.
//
//	@param c Gin context
func UserDeletionMiddleware(c *gin.Context) {
	userId, _ := c.Get("userId")
	// Check if user is in delete list
	toDelete, err := database.WillUserBeDeleted(userId.(uint64))
	if err != nil {
		/*
			c.AbortWithStatus(http.StatusInternalServerError)
			c.Error(err)
		*/
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	// User should be deleted
	if toDelete {
		/*
			c.AbortWithStatus(http.StatusUnauthorized)
			c.Error(errors.New("your account will be deleted"))
		*/
		c.AbortWithError(http.StatusUnauthorized, errors.New("your account will be deleted"))
		return
	}
	c.Next()
}
