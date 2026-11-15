package routes

import (
	"jsfraz/whisper-server/handlers"
	"jsfraz/whisper-server/middlewares"
	"jsfraz/whisper-server/utils"
	"net/http"

	"github.com/loopfz/gadgeto/tonic"
	"github.com/wI2L/fizz"
)

// Sets auth route group.
//
//	@param grp
func AuthRoute(g *fizz.RouterGroup) {

	grp := g.Group("auth", "Authentication", "User Authentication and registration.")

	// Register user
	grp.POST("register",
		utils.CreateOperationOption(
			"Register new user",
			"",
			[]int{
				http.StatusBadRequest,
				http.StatusUnauthorized,
				http.StatusConflict,
				http.StatusInternalServerError,
			},
			false),
		// Registration is expensive (RSA validation) and invite inputs are
		// attacker supplied.
		middlewares.RateLimit(utils.GetSingleton().Config.RateLimitBurst, utils.GetSingleton().Config.RateLimitPerMinute),
		tonic.Handler(handlers.CreateUser, http.StatusOK),
	)

	// One-time login challenge
	grp.GET("challenge",
		utils.CreateOperationOption(
			"Get a login challenge",
			"Returns a one-time value the client embeds in the RSA-signed login JWT. Every challenge is accepted once.",
			[]int{
				http.StatusInternalServerError,
				http.StatusTooManyRequests,
			},
			false),
		middlewares.RateLimit(utils.GetSingleton().Config.RateLimitBurst, utils.GetSingleton().Config.RateLimitPerMinute),
		tonic.Handler(handlers.AuthChallenge, http.StatusOK),
	)

	// Auth
	grp.POST("",
		utils.CreateOperationOption(
			"User auth",
			"",
			[]int{
				http.StatusBadRequest,
				http.StatusUnauthorized,
				http.StatusInternalServerError,
			}, false),
		// Login verifies an RSA signature per request.
		middlewares.RateLimit(utils.GetSingleton().Config.RateLimitBurst, utils.GetSingleton().Config.RateLimitPerMinute),
		tonic.Handler(handlers.AuthUser, http.StatusOK))

	// Refresh
	grp.POST("refresh",
		utils.CreateOperationOption("Refresh access token.",
			"",
			[]int{
				http.StatusBadRequest,
				http.StatusUnauthorized,
				http.StatusInternalServerError,
			}, false),
		middlewares.RateLimit(utils.GetSingleton().Config.RateLimitBurst, utils.GetSingleton().Config.RateLimitPerMinute),
		tonic.Handler(handlers.RefreshUserAccessToken, http.StatusOK))
}
