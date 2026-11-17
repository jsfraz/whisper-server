package handlers

import (
	"errors"
	"jsfraz/whisper-server/database"
	"jsfraz/whisper-server/models"
	"jsfraz/whisper-server/push"
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
		return nil, c.AbortWithError(http.StatusUnauthorized, errors.New(utils.ErrorInviteDoesNotExist))
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
		return nil, c.AbortWithError(http.StatusConflict, errors.New(utils.ErrorUsernameAlreadyTaken))
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
	// A client older than the server supports is refused here, with a code the
	// app can translate, instead of failing later in a way nobody can explain.
	if !utils.ClientVersionSupported(c.GetHeader("X-Client-Version")) {
		return nil, c.AbortWithError(http.StatusUpgradeRequired, errors.New(utils.ErrorClientVersionUnsupported))
	}
	// A malformed token is the client's problem, not a server error.
	userId, err := utils.GetUserIdFromToken(request.Token)
	if err != nil {
		return nil, c.AbortWithError(http.StatusUnauthorized, errors.New(utils.ErrorInvalidToken))
	}
	// Check if user exists
	exists, err := database.UserExistsById(userId)
	if err != nil {
		return nil, c.AbortWithError(http.StatusInternalServerError, err)
	}
	// Current user does not exist
	if !exists {
		return nil, c.AbortWithError(http.StatusUnauthorized, errors.New(utils.ErrorCurrentUserDoesNotExist))
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
		return nil, c.AbortWithError(http.StatusUnauthorized, errors.New(utils.ErrorInvalidToken))
	}
	// The device id travels inside the signed token: the body cannot name a
	// different device than the one that proved the key.
	signedDeviceId, err := utils.GetStringClaimFromToken(request.Token, "did")
	if err != nil || signedDeviceId != request.DeviceId {
		return nil, c.AbortWithError(http.StatusUnauthorized, errors.New(utils.ErrorInvalidDevice))
	}
	// The signature is good, so the claim is trustworthy: spend the challenge.
	// A token without one, or a replayed token, is refused.
	challenge, err := utils.GetStringClaimFromToken(request.Token, "nonce")
	if err != nil {
		return nil, c.AbortWithError(http.StatusUnauthorized, errors.New(utils.ErrorLoginChallengeMissing))
	}
	// Register (or touch) the device before the challenge is spent: a revoked
	// device must not burn a login attempt.
	created, revoked, err := database.RegisterDevice(userId, request.DeviceId, request.Label, request.Platform)
	if err != nil {
		if utils.IsErrorCode(err.Error()) {
			return nil, c.AbortWithError(http.StatusUnauthorized, err)
		}
		return nil, c.AbortWithError(http.StatusInternalServerError, err)
	}
	if revoked {
		return nil, c.AbortWithError(http.StatusUnauthorized, errors.New(utils.ErrorDeviceRevoked))
	}
	valid, err := database.ConsumeChallenge(challenge)
	if err != nil {
		return nil, c.AbortWithError(http.StatusInternalServerError, err)
	}
	if !valid {
		return nil, c.AbortWithError(http.StatusUnauthorized, errors.New(utils.ErrorLoginChallengeUsed))
	}
	// Generate access token
	accessToken, err := utils.GenerateToken(userId, request.DeviceId, utils.GetSingleton().Config.AccessTokenLifespan, utils.GetSingleton().Config.AccessTokenSecret, nil, utils.TokenTypeAccess)
	if err != nil {
		return nil, c.AbortWithError(http.StatusInternalServerError, err)
	}
	// Generate refresh token. Its id is recorded per device, so the session can
	// be revoked without touching the other devices of the account.
	refreshTokenId := uuid.NewString()
	refreshLifespan := utils.GetSingleton().Config.RefreshTokenLifespan
	refreshToken, err := utils.GenerateToken(userId, request.DeviceId, refreshLifespan, utils.GetSingleton().Config.RefreshTokenSecret, &refreshTokenId, utils.TokenTypeRefresh)
	if err != nil {
		return nil, c.AbortWithError(http.StatusInternalServerError, err)
	}
	if err = database.StoreRefreshSession(userId, request.DeviceId, refreshTokenId, refreshLifespan); err != nil {
		return nil, c.AbortWithError(http.StatusInternalServerError, err)
	}
	// A new device warns the devices that already existed. This must not fail
	// the login: push is best effort.
	if created {
		go func() {
			if err := push.SendToOtherDevices(userId, request.DeviceId, "New login", request.Label); err != nil {
				slog.Warn("failed to send new login notification",
					"user_id", utils.SanitizeLogValue(userId),
					"error", utils.SanitizeLogValue(err.Error()))
			}
		}()
	}
	return models.NewAuthResponse(accessToken, refreshToken, utils.Version), nil
}

// Refresh user access token.
//
//	@param c
//	@param refresh
//	@return *models.RefreshResponse
//	@return error
func RefreshUserAccessToken(c *gin.Context, refresh *models.Refresh) (*models.RefreshResponse, error) {
	// Validate token and get its claims
	claims, err := utils.TokenValid(refresh.RefreshToken, utils.GetSingleton().Config.RefreshTokenSecret, utils.TokenTypeRefresh)
	if err != nil {
		return nil, c.AbortWithError(http.StatusUnauthorized, errors.New(utils.ErrorInvalidToken))
	}
	// A token without an id predates revocation and is not accepted.
	if claims.TokenId == nil {
		return nil, c.AbortWithError(http.StatusUnauthorized, errors.New(utils.ErrorRefreshTokenNotRevocable))
	}
	// The session must be live and is spent by this call: a stolen refresh
	// token is only usable once, and the client gets a new one.
	live, err := database.ConsumeRefreshSession(claims.UserId, claims.DeviceId, *claims.TokenId)
	if err != nil {
		return nil, c.AbortWithError(http.StatusInternalServerError, err)
	}
	if !live {
		return nil, c.AbortWithError(http.StatusUnauthorized, errors.New(utils.ErrorRefreshSessionRevoked))
	}
	// Revocation also deletes the sessions, but a racing revoke must not be
	// undone by this renewal.
	active, err := database.IsDeviceActive(claims.UserId, claims.DeviceId)
	if err != nil {
		return nil, c.AbortWithError(http.StatusInternalServerError, err)
	}
	if !active {
		return nil, c.AbortWithError(http.StatusUnauthorized, errors.New(utils.ErrorDeviceRevoked))
	}
	// Check if user exists
	exists, err := database.UserExistsById(claims.UserId)
	if err != nil {
		return nil, c.AbortWithError(http.StatusInternalServerError, err)
	}
	// Current user does not exist
	if !exists {
		return nil, c.AbortWithError(http.StatusUnauthorized, errors.New(utils.ErrorCurrentUserDoesNotExist))
	}
	// Generate access token
	accessToken, err := utils.GenerateToken(claims.UserId, claims.DeviceId, utils.GetSingleton().Config.AccessTokenLifespan, utils.GetSingleton().Config.AccessTokenSecret, nil, utils.TokenTypeAccess)
	if err != nil {
		return nil, c.AbortWithError(http.StatusInternalServerError, err)
	}
	// Rotate the refresh session.
	refreshTokenId := uuid.NewString()
	refreshLifespan := utils.GetSingleton().Config.RefreshTokenLifespan
	refreshToken, err := utils.GenerateToken(claims.UserId, claims.DeviceId, refreshLifespan, utils.GetSingleton().Config.RefreshTokenSecret, &refreshTokenId, utils.TokenTypeRefresh)
	if err != nil {
		return nil, c.AbortWithError(http.StatusInternalServerError, err)
	}
	if err = database.StoreRefreshSession(claims.UserId, claims.DeviceId, refreshTokenId, refreshLifespan); err != nil {
		return nil, c.AbortWithError(http.StatusInternalServerError, err)
	}
	return models.NewRefreshResponse(accessToken, refreshToken), nil
}
