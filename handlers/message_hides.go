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

// ClearConversation deletes a whole conversation for the calling account only.
// Everything sent at or before the cut (now, or the `before` query parameter)
// stops being visible to this account on every device; the other participant
// keeps the archive untouched.
//
//	@param c
//	@param request
//	@return *models.ClearConversationResponse
//	@return error
func ClearConversation(c *gin.Context, request *models.ClearConversationQuery) (*models.ClearConversationResponse, error) {
	userId, _ := c.Get("userId")
	// The self-chat only exists in debug, where clearing it is allowed; a
	// release server has no such conversation to clear.
	if request.PeerId == userId.(string) &&
		utils.GetSingleton().Config.GinMode == "release" {
		return nil, c.AbortWithError(http.StatusBadRequest, errors.New(utils.ErrorCannotSendToSelf))
	}
	before := time.Now().UTC()
	if request.Before != "" {
		parsed, err := time.Parse(time.RFC3339, request.Before)
		if err != nil {
			return nil, c.AbortWithError(http.StatusBadRequest, errors.New(utils.ErrorInvalidRequest))
		}
		before = parsed.UTC()
	}
	clearedAt, err := database.ClearConversation(userId.(string), request.PeerId, before)
	if err != nil {
		return nil, c.AbortWithError(http.StatusInternalServerError, err)
	}
	return &models.ClearConversationResponse{ClearedAt: clearedAt}, nil
}

// HideMessage removes one message from the calling account only, on every
// device. Both participants may hide a message; the other side keeps it.
//
//	@param c
//	@param request
//	@return error
func HideMessage(c *gin.Context, request *models.MessageIdPath) error {
	userId, _ := c.Get("userId")
	messageId := request.Id
	senderId, receiverId, found, err := database.MessageParticipants(messageId)
	if err != nil {
		return c.AbortWithError(http.StatusInternalServerError, err)
	}
	if !found {
		return c.AbortWithError(http.StatusNotFound, errors.New(utils.ErrorMessageNotFound))
	}
	if senderId != userId.(string) && receiverId != userId.(string) {
		return c.AbortWithError(http.StatusForbidden, errors.New(utils.ErrorNotAuthorized))
	}
	if err := database.HideMessage(userId.(string), messageId); err != nil {
		return c.AbortWithError(http.StatusInternalServerError, err)
	}
	return nil
}

// DeleteMessage drops an own message for everyone: the ciphertext is erased and
// the row keeps a tombstone, so the other side learns about it on the next sync.
// A message the caller did not send cannot be deleted for the other side.
//
//	@param c
//	@param request
//	@return error
func DeleteMessage(c *gin.Context, request *models.MessageIdPath) error {
	userId, _ := c.Get("userId")
	messageId := request.Id
	senderId, _, found, err := database.MessageParticipants(messageId)
	if err != nil {
		return c.AbortWithError(http.StatusInternalServerError, err)
	}
	if !found {
		return c.AbortWithError(http.StatusNotFound, errors.New(utils.ErrorMessageNotFound))
	}
	if senderId != userId.(string) {
		return c.AbortWithError(http.StatusForbidden, errors.New(utils.ErrorNotAuthorized))
	}
	if err := database.TombstoneMessage(messageId); err != nil {
		return c.AbortWithError(http.StatusInternalServerError, err)
	}
	return nil
}
