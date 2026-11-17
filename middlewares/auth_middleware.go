package middlewares

import (
	"errors"
	"jsfraz/whisper-server/database"
	"jsfraz/whisper-server/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

// The database lookups, behind variables so the middlewares can be tested
// without a database.
var (
	userExistsById    = database.UserExistsById
	willUserBeDeleted = database.WillUserBeDeleted
	isDeviceActive    = database.IsDeviceActive
)

// Middleware for user authentication.
//
// If the user has a valid access token, it sets its ID and device ID in the
// context. If it is not valid, or the device was revoked, it returns 401.
//
//	@param c Gin context
func AuthMiddleware(c *gin.Context) {
	// Get access token from context and check it
	claims, err := utils.TokenValid(utils.ExtractTokenFromContext(c), utils.GetSingleton().Config.AccessTokenSecret, utils.TokenTypeAccess)
	// Invalid token
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": utils.ErrorInvalidToken})
		_ = c.AbortWithError(http.StatusUnauthorized, err)
		return
	}
	// Check if user exists
	exists, err := userExistsById(claims.UserId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": utils.ErrorInternal})
		_ = c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	// Current user does not exist
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": utils.ErrorCurrentUserDoesNotExist})
		// err is nil here: AbortWithError would panic on it.
		c.Abort()
		return
	}
	// A revoked device loses its API access immediately, not on the next token
	// expiry: only the account key is not enough to stay in.
	deviceActive, err := isDeviceActive(claims.UserId, claims.DeviceId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": utils.ErrorInternal})
		_ = c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	if !deviceActive {
		c.JSON(http.StatusUnauthorized, gin.H{"error": utils.ErrorDeviceRevoked})
		_ = c.AbortWithError(http.StatusUnauthorized, errors.New(utils.ErrorDeviceRevoked))
		return
	}
	// A user an admin deleted must lose API access immediately, not only on the
	// next WebSocket connect.
	deletePending, err := willUserBeDeleted(claims.UserId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": utils.ErrorInternal})
		_ = c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	if deletePending {
		c.JSON(http.StatusUnauthorized, gin.H{"error": utils.ErrorCurrentUserDoesNotExist})
		// err is nil here: AbortWithError would panic on it.
		c.Abort()
		return
	}
	// Token is valid, set it to context and continue
	c.Set("userId", claims.UserId)
	c.Set("deviceId", claims.DeviceId)
	c.Next()
}
