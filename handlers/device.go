package handlers

import (
	"jsfraz/whisper-server/database"
	"jsfraz/whisper-server/models"
	"jsfraz/whisper-server/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetDevices lists the devices of the account. The device that made the request
// is flagged so the app can say "this device".
//
//	@param c
//	@return *models.DevicesResponse
//	@return error
func GetDevices(c *gin.Context) (*models.DevicesResponse, error) {
	userId, _ := c.Get("userId")
	deviceId, _ := c.Get("deviceId")
	devices, err := database.GetDevices(userId.(string))
	if err != nil {
		return nil, c.AbortWithError(http.StatusInternalServerError, err)
	}
	for i := range *devices {
		(*devices)[i].Current = (*devices)[i].Id == deviceId.(string)
	}
	return &models.DevicesResponse{Devices: *devices}, nil
}

// RevokeDevice locks one device of the account out: it cannot log in, renew its
// token, call the API or keep its socket.
//
//	@param c
//	@param request
//	@return error
func RevokeDevice(c *gin.Context, request *models.DeviceIdPath) error {
	userId, _ := c.Get("userId")
	deviceId := request.Id
	if err := database.RevokeDevice(userId.(string), deviceId); err != nil {
		return abortDeviceError(c, err)
	}
	// Sessions and the live socket go with the revocation, otherwise the device
	// keeps working until its tokens expire.
	if err := database.DeleteDeviceRefreshSessions(userId.(string), deviceId); err != nil {
		return c.AbortWithError(http.StatusInternalServerError, err)
	}
	utils.GetSingleton().Hub.CloseDevice(userId.(string), deviceId, utils.ErrorDeviceRevoked)
	return nil
}

// UnrevokeDevice lets a device sign in again. Only a live device of the same
// account can do it.
//
//	@param c
//	@param request
//	@return error
func UnrevokeDevice(c *gin.Context, request *models.DeviceIdPath) error {
	userId, _ := c.Get("userId")
	deviceId := request.Id
	if err := database.UnrevokeDevice(userId.(string), deviceId); err != nil {
		return abortDeviceError(c, err)
	}
	return nil
}

// abortDeviceError maps a device lookup failure to its status.
func abortDeviceError(c *gin.Context, err error) error {
	if utils.IsErrorCode(err.Error()) {
		return c.AbortWithError(http.StatusNotFound, err)
	}
	return c.AbortWithError(http.StatusInternalServerError, err)
}
