package handlers

import (
	"errors"
	"jsfraz/whisper-server/database"
	"jsfraz/whisper-server/models"
	"jsfraz/whisper-server/utils"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Create user.
//
//	@param c
//	@param request
//	@return error
func CreateUser(c *gin.Context, request *models.Register) (*models.User, error) {
	// The invite is consumed atomically: a code is single-use even under a race
	// between two registrations.
	exists, inviteDataBytes, err := database.ConsumeInviteDataByCode(request.InviteCode)
	if err != nil {
		return nil, c.AbortWithError(http.StatusInternalServerError, err)
	}
	// Check if invite exists
	if !exists {
		return nil, c.AbortWithError(http.StatusUnauthorized, errors.New("invite does not exist"))
	}
	// Unmarshall invite data
	inviteData, err := models.InviteFromJson(inviteDataBytes)
	if err != nil {
		return nil, c.AbortWithError(http.StatusInternalServerError, err)
	}
	// Registration failed below: hand the still valid invite back instead of
	// burning it.
	restoreInvite := func() {
		if err := database.RestoreInviteData(request.InviteCode, inviteDataBytes, inviteData.ValidUntil); err != nil {
			slog.Error("failed to restore invite", "error", utils.SanitizeLogValue(err.Error()))
		}
	}
	// Check if username is taken
	taken, err := database.UserExistsByUsername(request.Username)
	if err != nil {
		restoreInvite()
		return nil, c.AbortWithError(http.StatusInternalServerError, err)
	}
	if taken {
		restoreInvite()
		return nil, c.AbortWithError(http.StatusConflict, errors.New("username already taken"))
	}
	// Validate public key (add newlines to start/end)
	// publicKey := strings.Replace(strings.Replace(request.PublicKey, "-----BEGIN PUBLIC KEY-----", "-----BEGIN PUBLIC KEY-----\n", 1), "-----END PUBLIC KEY-----", "\n-----END PUBLIC KEY-----", 1)
	publicKey := request.PublicKey
	_, err = utils.LoadRsaPublicKey([]byte(publicKey))
	if err != nil {
		restoreInvite()
		return nil, c.AbortWithError(http.StatusInternalServerError, err)
	}
	// Create user
	newUser := models.NewUser(request.Username, publicKey, inviteData.Admin)
	err = database.InsertUser(newUser, request.InviteCode)
	if err != nil {
		restoreInvite()
		return nil, c.AbortWithError(http.StatusInternalServerError, err)
	}
	return newUser, nil
}

// Returns a one-time challenge for the RSA login.
//
// The client signs a JWT carrying this value and the server consumes it, so the
// login token is single-use: a captured token cannot be replayed and a revoked
// session stays revoked.
//
//	@param c
//	@return *models.AuthChallengeResponse
//	@return error
func AuthChallenge(c *gin.Context) (*models.AuthChallengeResponse, error) {
	challenge, err := utils.RandomASCIIString(64)
	if err != nil {
		return nil, c.AbortWithError(http.StatusInternalServerError, err)
	}
	if err = database.PushChallenge(challenge); err != nil {
		return nil, c.AbortWithError(http.StatusInternalServerError, err)
	}
	return models.NewAuthChallengeResponse(challenge, database.ChallengeTtl), nil
}

// User auth.
//
//	@param c
//	@param request
//	@return error
func AuthUser(c *gin.Context, request *models.Auth) (*models.AuthResponse, error) {
	// A malformed token is the client's problem, not a server error.
	userId, err := utils.GetUserIdFromToken(request.Token)
	if err != nil {
		return nil, c.AbortWithError(http.StatusUnauthorized, err)
	}
	// Check if user exists
	exists, err := database.UserExistsById(userId)
	if err != nil {
		return nil, c.AbortWithError(http.StatusInternalServerError, err)
	}
	// Current user does not exist
	if !exists {
		return nil, c.AbortWithError(http.StatusUnauthorized, errors.New("current user does not exist"))
	}
	// Get user public key
	publicKeyPem, err := database.GetUserPublicKey(userId)
	if err != nil {
		return nil, c.AbortWithError(http.StatusInternalServerError, err)
	}
	publicKey, err := utils.LoadRsaPublicKey([]byte(publicKeyPem))
	if err != nil {
		return nil, c.AbortWithError(http.StatusInternalServerError, err)
	}
	// Validate token
	err = utils.ValidateRsaJwtToken(request.Token, publicKey)
	if err != nil {
		return nil, c.AbortWithError(http.StatusUnauthorized, err)
	}
	// The signature is good, so the claim is trustworthy: spend the challenge.
	// A token without one, or a replayed token, is refused.
	challenge, err := utils.GetStringClaimFromToken(request.Token, "nonce")
	if err != nil {
		return nil, c.AbortWithError(http.StatusUnauthorized, errors.New("login token has no challenge"))
	}
	valid, err := database.ConsumeChallenge(challenge)
	if err != nil {
		return nil, c.AbortWithError(http.StatusInternalServerError, err)
	}
	if !valid {
		return nil, c.AbortWithError(http.StatusUnauthorized, errors.New("login token challenge is unknown or already used"))
	}
	// Generate access token
	accessToken, err := utils.GenerateToken(userId, utils.GetSingleton().Config.AccessTokenLifespan, utils.GetSingleton().Config.AccessTokenSecret, nil, utils.TokenTypeAccess)
	if err != nil {
		return nil, c.AbortWithError(http.StatusInternalServerError, err)
	}
	// Generate refresh token. Its id is recorded, so the session can be revoked
	// later (account deletion, logout everywhere) instead of living out its TTL.
	refreshTokenId := uuid.NewString()
	refreshLifespan := utils.GetSingleton().Config.RefreshTokenLifespan
	refreshToken, err := utils.GenerateToken(userId, refreshLifespan, utils.GetSingleton().Config.RefreshTokenSecret, &refreshTokenId, utils.TokenTypeRefresh)
	if err != nil {
		return nil, c.AbortWithError(http.StatusInternalServerError, err)
	}
	if err = database.StoreRefreshSession(userId, refreshTokenId, refreshLifespan); err != nil {
		return nil, c.AbortWithError(http.StatusInternalServerError, err)
	}
	return models.NewAuthResponse(accessToken, refreshToken), nil
}

// Refresh user access token.
//
//	@param c
//	@param refresh
//	@return *models.RefreshResponse
//	@return error
func RefreshUserAccessToken(c *gin.Context, refresh *models.Refresh) (*models.RefreshResponse, error) {
	// Validate token and get user id
	userId, tokenId, err := utils.TokenValid(refresh.RefreshToken, utils.GetSingleton().Config.RefreshTokenSecret, utils.TokenTypeRefresh)
	if err != nil {
		return nil, c.AbortWithError(http.StatusUnauthorized, err)
	}
	// A token without an id predates revocation and is not accepted.
	if tokenId == nil {
		return nil, c.AbortWithError(http.StatusUnauthorized, errors.New("refresh token is not revocable"))
	}
	// The session must be live and is spent by this call: a stolen refresh
	// token is only usable once, and the client gets a new one.
	live, err := database.ConsumeRefreshSession(userId, *tokenId)
	if err != nil {
		return nil, c.AbortWithError(http.StatusInternalServerError, err)
	}
	if !live {
		return nil, c.AbortWithError(http.StatusUnauthorized, errors.New("refresh session was revoked or already used"))
	}
	// Check if user exists
	exists, err := database.UserExistsById(userId)
	if err != nil {
		return nil, c.AbortWithError(http.StatusInternalServerError, err)
	}
	// Current user does not exist
	if !exists {
		return nil, c.AbortWithError(http.StatusUnauthorized, errors.New("current user does not exist"))
	}
	// Generate access token
	accessToken, err := utils.GenerateToken(userId, utils.GetSingleton().Config.AccessTokenLifespan, utils.GetSingleton().Config.AccessTokenSecret, nil, utils.TokenTypeAccess)
	if err != nil {
		return nil, c.AbortWithError(http.StatusInternalServerError, err)
	}
	// Rotate the refresh session.
	refreshTokenId := uuid.NewString()
	refreshLifespan := utils.GetSingleton().Config.RefreshTokenLifespan
	refreshToken, err := utils.GenerateToken(userId, refreshLifespan, utils.GetSingleton().Config.RefreshTokenSecret, &refreshTokenId, utils.TokenTypeRefresh)
	if err != nil {
		return nil, c.AbortWithError(http.StatusInternalServerError, err)
	}
	if err = database.StoreRefreshSession(userId, refreshTokenId, refreshLifespan); err != nil {
		return nil, c.AbortWithError(http.StatusInternalServerError, err)
	}
	return models.NewRefreshResponse(accessToken, refreshToken), nil
}
