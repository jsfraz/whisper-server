package handlers

import (
	"errors"
	"jsfraz/whisper-server/database"
	"jsfraz/whisper-server/models"
	"jsfraz/whisper-server/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetRetention returns the archive retention of the account.
//
//	@param c
//	@return *models.RetentionResponse
//	@return error
func GetRetention(c *gin.Context) (*models.RetentionResponse, error) {
	userId, _ := c.Get("userId")
	days, err := database.GetUserRetentionDays(userId.(string))
	if err != nil {
		return nil, c.AbortWithError(http.StatusInternalServerError, err)
	}
	return models.NewRetentionResponse(days), nil
}

// SetRetention stores the archive retention of the account. The archive is also
// the offline delivery buffer, so only the offered values are accepted.
//
//	@param c
//	@param request
//	@return error
func SetRetention(c *gin.Context, request *models.RetentionRequest) error {
	userId, _ := c.Get("userId")
	if !models.IsRetentionDaysAllowed(request.Days) {
		return c.AbortWithError(http.StatusBadRequest, errors.New(utils.ErrorInvalidRetention))
	}
	if err := database.SetUserRetentionDays(userId.(string), request.Days); err != nil {
		return c.AbortWithError(http.StatusInternalServerError, err)
	}
	return nil
}
