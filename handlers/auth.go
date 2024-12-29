package handlers

import (
	"errors"
	"jsfraz/whisper-server/database"
	"jsfraz/whisper-server/models"
	"jsfraz/whisper-server/utils"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// Create user.
//
//	@param c
//	@param request
//	@return error
func CreateUser(c *gin.Context, request *models.Register) (*models.User, error) {
	exists, inviteDataBytes, err := database.GetInviteDataByCode(request.InviteCode)
	if err != nil {
		return nil, c.AbortWithError(http.StatusInternalServerError, err)
	}
	// Check if invite exists
	if !exists {
		return nil, c.AbortWithError(http.StatusUnauthorized, errors.New("invite does not exist"))
	}
	// Unmarshall invite data
	inviteData, err := models.InviteDataFromJson(inviteDataBytes)
	if err != nil {
		return nil, c.AbortWithError(http.StatusInternalServerError, err)
	}
	// Check if username is taken
	taken, err := database.UserExistsByUsername(request.Username)
	if taken {
		return nil, c.AbortWithError(http.StatusConflict, errors.New("username already taken"))
	}
	// Validate public key (add newlines to start/end)
	publicKey := strings.Replace(strings.Replace(request.PublicKey, "-----BEGIN PUBLIC KEY-----", "-----BEGIN PUBLIC KEY-----\n", 1), "-----END PUBLIC KEY-----", "\n-----END PUBLIC KEY-----", 1)
	_, err = utils.LoadRsaPublicKey([]byte(publicKey))
	if err != nil {
		return nil, c.AbortWithError(http.StatusInternalServerError, err)
	}
	// Create user
	newUser := models.NewUser(request.Username, inviteData.Mail, publicKey, inviteData.Admin)
	err = database.InsertUser(newUser, request.InviteCode)
	if err != nil {
		return nil, c.AbortWithError(http.StatusInternalServerError, err)
	}
	return newUser, nil
}

// User auth.
//
//	@param c
//	@param request
//	@return error
func AuthUser(c *gin.Context, request *models.Auth) (*models.AuthResponse, error) {
	// Check if user exists
	exists, err := database.UserExistsById(request.UserId)
	if err != nil {
		return nil, c.AbortWithError(http.StatusInternalServerError, err)
	}
	// User does not exist
	if !exists {
		return nil, c.AbortWithError(http.StatusUnauthorized, errors.New("user does not exist"))
	}
	// Get user public key
	publicKeyPem, err := database.GetUserPublicKey(request.UserId)
	if err != nil {
		return nil, c.AbortWithError(http.StatusInternalServerError, err)
	}
	publicKey, err := utils.LoadRsaPublicKey([]byte(publicKeyPem))
	if err != nil {
		return nil, c.AbortWithError(http.StatusInternalServerError, err)
	}
	// Verify signature
	err = utils.VerifyRSASignature(publicKey, request.Nonce, request.SignedNonce)
	if err != nil {
		return nil, c.AbortWithError(http.StatusUnauthorized, err)
	}
	// Generate access token
	accessToken, err := utils.GenerateToken(request.UserId, utils.GetSingleton().Config.AccessTokenLifespan, utils.GetSingleton().Config.AccessTokenSecret)
	if err != nil {
		return nil, c.AbortWithError(http.StatusInternalServerError, err)
	}
	// Generate refresh token
	refreshToken, err := utils.GenerateToken(request.UserId, utils.GetSingleton().Config.RefreshTokenLifespan, utils.GetSingleton().Config.RefreshTokenSecret)
	if err != nil {
		return nil, c.AbortWithError(http.StatusInternalServerError, err)
	}
	return models.NewAuth(accessToken, refreshToken), nil
}

// Refresh user access token.
//
//	@param c
//	@param refresh
//	@return *models.RefreshResponse
//	@return error
func RefreshUserAccessToken(c *gin.Context, refresh *models.Refresh) (*models.RefreshResponse, error) {
	// Validate token and get user id
	userId, err := utils.TokenValid(refresh.RefreshToken, utils.GetSingleton().Config.RefreshTokenSecret)
	if err != nil {
		return nil, c.AbortWithError(http.StatusUnauthorized, err)
	}
	// Check if user exists
	exists, err := database.UserExistsById(userId)
	if err != nil {
		return nil, c.AbortWithError(http.StatusInternalServerError, err)
	}
	// User does not exist
	if !exists {
		return nil, c.AbortWithError(http.StatusUnauthorized, errors.New("user does not exist"))
	}
	// Generate access token
	accessToken, err := utils.GenerateToken(userId, utils.GetSingleton().Config.AccessTokenLifespan, utils.GetSingleton().Config.AccessTokenSecret)
	if err != nil {
		return nil, c.AbortWithError(http.StatusInternalServerError, err)
	}
	return models.NewRefreshResponse(accessToken), nil
}
