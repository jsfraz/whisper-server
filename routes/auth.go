package routes

import (
	"jsfraz/whisper-server/handlers"
	"jsfraz/whisper-server/utils"
	"net/http"

	"github.com/loopfz/gadgeto/tonic"
	"github.com/wI2L/fizz"
)

// Sets auth route group
func AuthRoute(grp *fizz.RouterGroup) {

	// Create new account
	grp.POST("register", utils.CreateOperationOption(
		"Creates new account.",
		[]int{
			http.StatusBadRequest,
			http.StatusInternalServerError,
		}, false),
		// Handler
		tonic.Handler(handlers.RegisterUser, http.StatusCreated))

	// Verify account
	grp.PATCH("verify",
		utils.CreateOperationOption("Verifies account.",
			[]int{
				http.StatusBadRequest,
				http.StatusInternalServerError,
			}, false),
		// handler
		tonic.Handler(handlers.VerifyUser, http.StatusNoContent))

	// Login
	grp.POST("login",
		utils.CreateOperationOption("User login.",
			[]int{
				http.StatusBadRequest,
				http.StatusInternalServerError,
			}, false),
		// handler
		tonic.Handler(handlers.LoginUser, http.StatusOK))

	/*
		// Refresh
		grp.GET("refresh",
			utils.CreateOperationOption("Refresh access token.",
				[]errors.Status{
					// Common errors
					errors.BadRequest,
					errors.Unauthorized,
					errors.InternalServerError,
				}, false),
			// handler
			tonic.Handler(handlers.RefreshUserAccessToken, 200))
	*/
}
