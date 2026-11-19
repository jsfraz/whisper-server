package handlers

import (
	"errors"
	"jsfraz/whisper-server/database"
	"jsfraz/whisper-server/models"
	"jsfraz/whisper-server/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

const (
	defaultHistoryPageSize = 50
	maxHistoryPageSize     = 200
)

// GetHistory returns one page of a conversation from the ciphertext archive,
// newest first. The app prepends the page to what it already has locally.
//
//	@param c
//	@param request
//	@return *models.HistoryPage
//	@return error
func GetHistory(c *gin.Context, request *models.HistoryQuery) (*models.HistoryPage, error) {
	userId, _ := c.Get("userId")
	limit := request.Limit
	if limit <= 0 {
		limit = defaultHistoryPageSize
	}
	if limit > maxHistoryPageSize {
		limit = maxHistoryPageSize
	}
	var before *utils.Cursor
	if request.Before != "" {
		cursor, err := utils.DecodeCursor(request.Before)
		if err != nil {
			return nil, c.AbortWithError(http.StatusBadRequest, errors.New(utils.ErrorInvalidCursor))
		}
		before = cursor
	}
	messages, err := database.HistoryForPeer(userId.(string), request.PeerId, before, limit)
	if err != nil {
		return nil, c.AbortWithError(http.StatusInternalServerError, err)
	}
	peers, err := database.GetUsersByIds(database.ConversationsOf(messages, userId.(string)))
	if err != nil {
		return nil, c.AbortWithError(http.StatusInternalServerError, err)
	}
	wire := make([]models.PrivateMessage, 0, len(messages))
	for _, message := range messages {
		wire = append(wire, message.ToWire(userId.(string), false))
	}
	peerList := make([]models.User, 0, len(peers))
	for _, peer := range peers {
		peerList = append(peerList, peer)
	}
	// The next page starts below the oldest message of this one.
	nextCursor := ""
	if len(messages) == limit {
		oldest := messages[len(messages)-1]
		nextCursor = utils.EncodeCursor(oldest.SentAt, oldest.Id)
	}
	// The client drops whatever it still holds from before its own cut.
	clearedAt, err := database.ConversationClearedAt(userId.(string), request.PeerId)
	if err != nil {
		return nil, c.AbortWithError(http.StatusInternalServerError, err)
	}
	// Pins are shared, so every participant sees the same list; a new device
	// gets them without a separate round trip.
	pins, err := database.PinsForPeer(userId.(string), request.PeerId)
	if err != nil {
		return nil, c.AbortWithError(http.StatusInternalServerError, err)
	}
	return &models.HistoryPage{
		Messages:   wire,
		Peers:      peerList,
		Pins:       pins,
		NextCursor: nextCursor,
		ClearedAt:  clearedAt,
	}, nil
}
