package handlers

import (
	"errors"
	"jsfraz/whisper-server/database"
	"jsfraz/whisper-server/models"
	"jsfraz/whisper-server/utils"
	"log/slog"
	"net/http"
	"slices"
	"strings"

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
	admin, err := database.IsAdmin(userId.(string))
	if err != nil {
		return nil, c.AbortWithError(http.StatusInternalServerError, err)
	}
	if !admin {
		return nil, c.AbortWithError(http.StatusUnauthorized, errors.New(utils.ErrorNotAuthorized))
	}
	// Get users
	users, err := database.GetAllUsersExceptUser(userId.(string))
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
	admin, err := database.IsAdmin(userId.(string))
	if err != nil {
		return c.AbortWithError(http.StatusInternalServerError, err)
	}
	if !admin {
		return c.AbortWithError(http.StatusUnauthorized, errors.New(utils.ErrorNotAuthorized))
	}
	// Check if user is deleting self
	if slices.Contains(request.Ids, userId.(string)) {
		return c.AbortWithError(http.StatusInternalServerError, errors.New(utils.ErrorCannotDeleteSelf))
	}
	// Check if user is already in list
	toDelete, err := database.GetAllUsersToDelete()
	if err != nil {
		return c.AbortWithError(http.StatusInternalServerError, err)
	}
	for _, userId := range request.Ids {
		if slices.Contains(toDelete, userId) {
			return c.AbortWithError(http.StatusInternalServerError, errors.New(utils.ErrorUserAlreadyInDeleteList))
		}
	}
	// Push delete to Valkey
	err = database.PushUsersToDelete(request.Ids)
	if err != nil {
		return c.AbortWithError(http.StatusInternalServerError, err)
	}
	// Delete messages, devices, sessions and media. A failure here leaves
	// ciphertext behind, so it must not pass silently.
	for _, userId := range request.Ids {
		if err := database.DeleteUserMessages(userId); err != nil {
			slog.Error("failed to delete user messages", "user_id", utils.SanitizeLogValue(userId), "error", utils.SanitizeLogValue(err.Error()))
			return c.AbortWithError(http.StatusInternalServerError, err)
		}
		if err := database.DeleteUserDevices(userId); err != nil {
			slog.Error("failed to delete user devices", "user_id", utils.SanitizeLogValue(userId), "error", utils.SanitizeLogValue(err.Error()))
			return c.AbortWithError(http.StatusInternalServerError, err)
		}
		if err := database.DeleteRefreshSessions(userId); err != nil {
			slog.Error("failed to revoke user sessions", "user_id", utils.SanitizeLogValue(userId), "error", utils.SanitizeLogValue(err.Error()))
			return c.AbortWithError(http.StatusInternalServerError, err)
		}
		if err := database.DeleteUserMedia(userId); err != nil {
			slog.Error("failed to delete user media", "user_id", utils.SanitizeLogValue(userId), "error", utils.SanitizeLogValue(err.Error()))
			return c.AbortWithError(http.StatusInternalServerError, err)
		}
	}
	// Close the sockets of the deleted accounts.
	for _, id := range request.Ids {
		utils.GetSingleton().Hub.CloseUser(id)
	}
	return nil
}

// Search users by username.
//
//	@param c
//	@return *[]models.User
//	@return error
func SearchUsers(c *gin.Context, request *models.UsernameQuery) (*[]models.User, error) {
	username := strings.TrimSpace(request.Username)
	if username == "" {
		return nil, c.AbortWithError(http.StatusBadRequest, errors.New(utils.ErrorUsernameRequired))
	}
	userId, _ := c.Get("userId")
	users, err := database.SearchUsersByUsername(username, userId.(string))
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
		return nil, c.AbortWithError(http.StatusNotFound, errors.New(utils.ErrorUserNotFound))
	}
	// Get user
	user, err := database.GetUserById(request.Id)
	if err != nil {
		return nil, c.AbortWithError(http.StatusInternalServerError, err)
	}
	return user, nil
}

// Delete current user.
//
//	@param c
//	@return error
func DeleteMe(c *gin.Context) error {
	userId, _ := c.Get("userId")
	// Check if user is admin
	admin, err := database.IsAdmin(userId.(string))
	if err != nil {
		return c.AbortWithError(http.StatusInternalServerError, err)
	}
	// Delete user
	err = database.DeleteUserById(userId.(string))
	if err != nil {
		return c.AbortWithError(http.StatusInternalServerError, err)
	}

	// Revoke every refresh token of the account
	if err = database.DeleteRefreshSessions(userId.(string)); err != nil {
		return c.AbortWithError(http.StatusInternalServerError, err)
	}

	// Send admin invite
	if admin {
		err = database.CreateAdminInvite()
		if err != nil {
			return c.AbortWithError(http.StatusInternalServerError, err)
		}
	}

	// Delete messages for the user
	err = database.DeleteUserMessages(userId.(string))
	if err != nil {
		return c.AbortWithError(http.StatusInternalServerError, err)
	}

	// Delete the device registry rows
	err = database.DeleteUserDevices(userId.(string))
	if err != nil {
		return c.AbortWithError(http.StatusInternalServerError, err)
	}

	// Delete media for the user
	err = database.DeleteUserMedia(userId.(string))
	if err != nil {
		return c.AbortWithError(http.StatusInternalServerError, err)
	}

	return nil
}
