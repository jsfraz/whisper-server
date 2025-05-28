package handlers

import (
	"errors"
	"jsfraz/whisper-server/database"
	"jsfraz/whisper-server/models"
	"jsfraz/whisper-server/utils"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// CreateInvite
//
//	@param c
//	@param request
//	@return error
func CreateInvite(c *gin.Context, request *models.CreateUser) error {
	userId, _ := c.Get("userId")
	// Check if user is admin
	admin, err := database.IsAdmin(userId.(uint64))
	if err != nil {
		return c.AbortWithError(http.StatusInternalServerError, err)
	}
	if !admin {
		return c.AbortWithError(http.StatusUnauthorized, errors.New("not authorized to create invites"))
	}
	// Create invite
	ttl := utils.GetSingleton().Config.InviteTtl
	err = database.PushInvite(utils.RandomASCIIString(64), *models.NewInvite(request.Mail, false, time.Now().Add(time.Duration(ttl)*time.Second)), ttl)
	if err != nil {
		return err
	}
	return nil
}

// Get all invites.
//
//	@param c
//	@return *[]models.Invite
//	@return error
func GetAllInvites(c *gin.Context) (*[]models.Invite, error) {
	userId, _ := c.Get("userId")
	// Check if user is admin
	admin, err := database.IsAdmin(userId.(uint64))
	if err != nil {
		return nil, c.AbortWithError(http.StatusInternalServerError, err)
	}
	if !admin {
		return nil, c.AbortWithError(http.StatusUnauthorized, errors.New("not authorized to get invites"))
	}
	// Get invites
	invites, err := database.GetAllInvites()
	if err != nil {
		return nil, c.AbortWithError(http.StatusInternalServerError, err)
	}
	return invites, nil
}
