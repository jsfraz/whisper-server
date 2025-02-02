package handlers

import (
	"errors"
	"fmt"
	"jsfraz/whisper-server/database"
	"jsfraz/whisper-server/models"
	"jsfraz/whisper-server/utils"
	"net/http"
	"slices"

	"github.com/gin-gonic/gin"
)

// Get all users except the user.
//
//	@param c
//	@return *models.User
//	@return error
func GetAllUsers(c *gin.Context) (*[]models.User, error) {
	userId, _ := c.Get("userId")
	// Check if user is admin
	admin, err := database.IsAdmin(userId.(uint64))
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
	}
	if !admin {
		c.AbortWithError(http.StatusUnauthorized, err)
	}
	// Get users
	users, err := database.GetAllUsersExceptUser(userId.(uint64))
	if err != nil {
		return nil, c.AbortWithError(http.StatusInternalServerError, err)
	}
	return users, nil
}

// Delete users by ID.
//
//	@param c
//	@param request
//	@return error
func DeleteUsers(c *gin.Context, request *models.IdsRequest) error {
	userId, _ := c.Get("userId")
	// Check if user is admin
	admin, err := database.IsAdmin(userId.(uint64))
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
	}
	if !admin {
		c.AbortWithError(http.StatusUnauthorized, err)
	}
	// Check if user is deleting self
	if slices.Contains(request.Ids, userId.(uint64)) {
		return c.AbortWithError(http.StatusInternalServerError, errors.New("cannot delete self"))
	}
	// Check if user is already in list
	toDelete, err := database.GetAllUsersToDelete()
	for _, userId := range request.Ids {
		if slices.Contains(toDelete, userId) {
			return c.AbortWithError(http.StatusInternalServerError, errors.New(fmt.Sprintf("user %d is already in delete list", userId)))
		}
	}
	// Push delete to Valkey
	err = database.PushUsersToDelete(request.Ids)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
	}
	// Delete messages from Valkey
	for _, userId := range request.Ids {
		database.DeleteUserPrivateMessages(userId)
	}
	// Send ws message to client
	onlineIds := utils.GetSingleton().Hub.DeleteUsers(request.Ids)
	// Delete IDs from Valkey
	for _, id := range onlineIds {
		database.RemoveDeletedUserId(id)
	}
	return nil
}

// Search users by username.
//
//	@param c
//	@return *[]models.User
//	@return error
func SearchUsers(c *gin.Context, request *models.UsernameQuery) (*[]models.User, error) {
	userId, _ := c.Get("userId")
	users, err := database.SearchUsersByUsername(request.Username, userId.(uint64))
	if err != nil {
		return nil, c.AbortWithError(http.StatusInternalServerError, err)
	}
	return users, nil
}

// Return user by ID.
//
//	@param c
//	@param request
//	@return *models.User
//	@return error
func GetUserById(c *gin.Context, request *models.IdQueryRequest) (*models.User, error) {
	// Check if user exists
	exists, err := database.UserExistsById(request.Id)
	if err != nil {
		return nil, c.AbortWithError(http.StatusInternalServerError, err)
	}
	if !exists {
		return nil, c.AbortWithError(http.StatusNotFound, err)
	}
	// Get user
	user, err := database.GetUserById(request.Id)
	if err != nil {
		return nil, c.AbortWithError(http.StatusInternalServerError, err)
	}
	return user, nil
}
