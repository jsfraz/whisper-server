package handlers

import (
	"jsfraz/shept-server/models"
	"jsfraz/shept-server/utils"

	"github.com/gin-gonic/gin"
)

// GetVersion describes the server build and the oldest client it accepts.
//
//	@param c
//	@return *models.VersionResponse
//	@return error
func GetVersion(c *gin.Context) (*models.VersionResponse, error) {
	return models.NewVersionResponse(
		utils.Version,
		utils.ApiVersion,
		utils.GetSingleton().Config.MinClientVersion,
	), nil
}
