package handlers

import (
	"jsfraz/whisper-server/database"
	"jsfraz/whisper-server/models"
	"jsfraz/whisper-server/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

// SetReadCursor moves the caller's read cursor of one conversation and tells the
// other participant, so the ticks in their chat turn to "read". Only the time is
// stored; the server never learns which message was opened.
//
//	@param c
//	@param request
//	@return error
func SetReadCursor(c *gin.Context, request *models.SetReadRequest) error {
	userId, _ := c.Get("userId")
	readAt, err := database.SetReadCursor(userId.(string), request.PeerId, request.ReadAt.UTC())
	if err != nil {
		return c.AbortWithError(http.StatusInternalServerError, err)
	}
	// Only the peer needs it: an account's own other devices already know what
	// it has read.
	utils.GetSingleton().Hub.SendToUser(request.PeerId, models.NewWsResponse(
		models.WsResponseTypeRead,
		models.ReadReceipt{V: 1, PeerId: userId.(string), ReadAt: readAt}))
	return nil
}
