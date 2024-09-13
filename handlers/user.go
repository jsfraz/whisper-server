package handlers

import (
	"errors"
	"jsfraz/whisper-server/database"
	"jsfraz/whisper-server/models"
	"jsfraz/whisper-server/utils"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

/*
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
*/

// Create user.
//
//	@param c
//	@param request
//	@return error
func CreateUser(c *gin.Context, request *models.CreateUser) error {
	exists, inviteDataBytes, err := database.GetInviteDataByCode(request.InviteCode)
	if err != nil {
		return c.AbortWithError(http.StatusInternalServerError, err)
	}
	// Check if invite exists
	if !exists {
		return c.AbortWithError(http.StatusUnauthorized, errors.New("invite does not exist"))
	}
	// Unmarshall invite data
	inviteData, err := models.InviteDataFromJson(inviteDataBytes)
	if err != nil {
		return c.AbortWithError(http.StatusInternalServerError, err)
	}
	// Check if username is taken
	taken, err := database.UserExistsByUsername(request.Username)
	if taken {
		return c.AbortWithError(http.StatusConflict, errors.New("username already taken"))
	}
	// Validate public key (add newlines to start/end)
	publicKey := strings.Replace(strings.Replace(request.PublicKey, "-----BEGIN PUBLIC KEY-----", "-----BEGIN PUBLIC KEY-----\n", 1), "-----END PUBLIC KEY-----", "\n-----END PUBLIC KEY-----", 1)
	_, err = utils.LoadRsaPublicKey([]byte(publicKey))
	if err != nil {
		return c.AbortWithError(http.StatusInternalServerError, err)
	}
	// Create user
	err = database.InsertUser(*models.NewUser(request.Username, inviteData.Mail, publicKey, inviteData.Admin), request.InviteCode)
	if err != nil {
		return c.AbortWithError(http.StatusInternalServerError, err)
	}
	return nil
}
