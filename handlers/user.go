package handlers

import (
	"jsfraz/whisper-server/database"
	"jsfraz/whisper-server/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Returns current user.
//
//	@param c
//	@return *models.User
//	@return error
func WhoAmI(c *gin.Context) (*models.User, error) {
	// User ID
	userId, _ := c.Get("userId")
	// Get user by ID
	user, err := database.GetUserById(userId.(uint64))
	if err != nil {
		return nil, c.AbortWithError(http.StatusInternalServerError, err)
	}
	return user, nil
}
