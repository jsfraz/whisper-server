package routes

import (
	"jsfraz/whisper-server/handlers"
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
		tonic.Handler(handlers.CreateUser, http.StatusOK),
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
		tonic.Handler(handlers.RefreshUserAccessToken, http.StatusOK))
}
