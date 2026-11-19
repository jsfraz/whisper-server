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

// broadcastEvent hands one mutation to every live device of both participants.
// The actor hears its own change back and folds it idempotently, which is what
// keeps its other devices in step.
func broadcastEvent(event *models.MessageEvent) {
	hub := utils.GetSingleton().Hub
	for _, userId := range models.ParticipantsOfConversation(event.Conversation) {
		hub.SendToUser(userId, models.NewWsResponse(
			models.ResponseTypeForKind(event.Kind), event.ToWire(userId)))
	}
}

// mutableMessage loads a message for a lifecycle change and checks that the
// caller may make it. A tombstone is gone for everyone, so it answers 404.
func mutableMessage(c *gin.Context, messageId string, senderOnly bool) (database.MessageRow, bool, error) {
	userId, _ := c.Get("userId")
	row, found, err := database.GetMessageRow(messageId)
	if err != nil {
		_ = c.AbortWithError(http.StatusInternalServerError, err)
		return database.MessageRow{}, false, err
	}
	if !found || row.DeletedAt != nil {
		err := c.AbortWithError(http.StatusNotFound, errors.New(utils.ErrorMessageNotFound))
		return database.MessageRow{}, false, err
	}
	if senderOnly && row.SenderId != userId.(string) {
		err := c.AbortWithError(http.StatusForbidden, errors.New(utils.ErrorNotAuthorized))
		return database.MessageRow{}, false, err
	}
	if !senderOnly && row.SenderId != userId.(string) && row.ReceiverId != userId.(string) {
		err := c.AbortWithError(http.StatusForbidden, errors.New(utils.ErrorNotAuthorized))
		return database.MessageRow{}, false, err
	}
	return row, true, nil
}

// EditMessage replaces the ciphertext of a message the caller sent. Anyone who
// holds the conversation can still read the new ciphertext; the server only
// swaps the bytes and stamps the edit.
//
//	@param c
//	@param request
//	@return *models.EditMessageResponse
//	@return error
func EditMessage(c *gin.Context, request *models.EditMessageRequest) (*models.EditMessageResponse, error) {
	row, ok, err := mutableMessage(c, request.Id, true)
	if !ok {
		return nil, err
	}
	editedAt := time.Now().UTC()
	event, err := database.EditMessage(row, models.MessageCiphertext{
		Content: request.Content,
		Key:     request.Key,
		Nonce:   request.Nonce,
		Mac:     request.Mac,
	}, editedAt)
	if err != nil {
		return nil, c.AbortWithError(http.StatusInternalServerError, err)
	}
	broadcastEvent(event)
	return &models.EditMessageResponse{EditedAt: editedAt}, nil
}

// SetReaction stores the caller's reaction on a message. Any participant may
// react; the ciphertext is opened per participant, the server only keeps bytes.
//
//	@param c
//	@param request
//	@return error
func SetReaction(c *gin.Context, request *models.ReactionRequest) error {
	row, ok, err := mutableMessage(c, request.Id, false)
	if !ok {
		return err
	}
	userId, _ := c.Get("userId")
	event, err := database.SetReaction(row, userId.(string), &models.MessageCiphertext{
		Content: request.Content,
		Key:     request.Key,
		Nonce:   request.Nonce,
		Mac:     request.Mac,
	}, time.Now().UTC())
	if err != nil {
		return c.AbortWithError(http.StatusInternalServerError, err)
	}
	broadcastEvent(event)
	return nil
}

// RemoveReaction clears the caller's reaction on a message. Idempotent.
//
//	@param c
//	@param request
//	@return error
func RemoveReaction(c *gin.Context, request *models.MessageIdPath) error {
	row, ok, err := mutableMessage(c, request.Id, false)
	if !ok {
		return err
	}
	userId, _ := c.Get("userId")
	event, err := database.SetReaction(row, userId.(string), nil, time.Now().UTC())
	if err != nil {
		return c.AbortWithError(http.StatusInternalServerError, err)
	}
	broadcastEvent(event)
	return nil
}

// PinMessage pins a message of the chat for both participants. Anyone in the
// conversation may pin.
//
//	@param c
//	@param request
//	@return error
func PinMessage(c *gin.Context, request *models.MessageIdPath) error {
	return setPinned(c, request.Id, true)
}

// UnpinMessage removes a pin for both participants. Idempotent.
//
//	@param c
//	@param request
//	@return error
func UnpinMessage(c *gin.Context, request *models.MessageIdPath) error {
	return setPinned(c, request.Id, false)
}

func setPinned(c *gin.Context, messageId string, pinned bool) error {
	row, ok, err := mutableMessage(c, messageId, false)
	if !ok {
		return err
	}
	userId, _ := c.Get("userId")
	event, changed, err := database.SetPinned(row, userId.(string), pinned, time.Now().UTC())
	if err != nil {
		return c.AbortWithError(http.StatusInternalServerError, err)
	}
	// A retry that changes nothing must not append a second log entry.
	if changed {
		broadcastEvent(event)
	}
	return nil
}
