package handlers

import (
	"errors"
	"jsfraz/whisper-server/database"
	"jsfraz/whisper-server/models"
	"jsfraz/whisper-server/utils"
	"net/http"
	"time"

	"github.com/asaskevich/govalidator"
	"github.com/gin-gonic/gin"
)

// maxActiveInvites caps how many unused invites may exist at once.
const maxActiveInvites = 100

// CreateInvite
//
//	@param c
//	@param request
//	@return error
func CreateInvite(c *gin.Context, request *models.CreateUser) error {
	userId, _ := c.Get("userId")
	// Check if user is admin
	admin, err := database.IsAdmin(userId.(string))
	if err != nil {
		return c.AbortWithError(http.StatusInternalServerError, err)
	}
	if !admin {
		return c.AbortWithError(http.StatusUnauthorized, errors.New(utils.ErrorNotAuthorized))
	}
	// An invite is sent to this address, so a typo is a bounced mail from our
	// own domain.
	if !govalidator.IsEmail(request.Mail) {
		return c.AbortWithError(http.StatusBadRequest, errors.New(utils.ErrorInvalidEmailAddress))
	}
	// Cap the number of live invites: an abused admin token must not turn the
	// server into an open relay for mail.
	active, err := database.GetAllInvites()
	if err != nil {
		return c.AbortWithError(http.StatusInternalServerError, err)
	}
	if len(*active) >= maxActiveInvites {
		return c.AbortWithError(http.StatusTooManyRequests, errors.New(utils.ErrorTooManyActiveInvites))
	}
	// Create invite
	ttl := utils.GetSingleton().Config.InviteTtl
	code, err := utils.RandomASCIIString(64)
	if err != nil {
		return c.AbortWithError(http.StatusInternalServerError, err)
	}
	err = database.PushInvite(code, *models.NewInvite(request.Mail, false, time.Now().Add(time.Duration(ttl)*time.Second)), ttl)
	if err != nil {
		return c.AbortWithError(http.StatusInternalServerError, err)
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
	admin, err := database.IsAdmin(userId.(string))
	if err != nil {
		return nil, c.AbortWithError(http.StatusInternalServerError, err)
	}
	if !admin {
		return nil, c.AbortWithError(http.StatusUnauthorized, errors.New(utils.ErrorNotAuthorized))
	}
	// Get invites
	invites, err := database.GetAllInvites()
	if err != nil {
		return nil, c.AbortWithError(http.StatusInternalServerError, err)
	}
	return invites, nil
}
