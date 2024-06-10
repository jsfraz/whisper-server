package routes

import (
	"jsfraz/whisper-server/errors"
	"jsfraz/whisper-server/handlers"
	"jsfraz/whisper-server/utils"

	"github.com/loopfz/gadgeto/tonic"
	"github.com/wI2L/fizz"
)

// Sets auth route group
func AuthRoute(grp *fizz.RouterGroup) {

	// Create new account
	grp.POST("register", utils.CreateOperationOption(
		"Creates new account and sends verification mail.",
		[]errors.Status{
			// Custom errors
			errors.UsernameTaken,
			errors.MailTaken,
			errors.VerificationMailNotSend,
			// Common errors
			errors.BadRequest,
			errors.InternalServerError,
		}, false),
		// Handler
		tonic.Handler(handlers.RegisterUser, 201))

	// Verify account
	grp.PATCH("verify",
		utils.CreateOperationOption("Verifies account.",
			[]errors.Status{
				// Custom errors
				errors.VerificationFailed,
				// Common errors
				errors.BadRequest,
				errors.InternalServerError,
			}, false),
		// handler
		tonic.Handler(handlers.VerifyUser, 204))

	/*
		// Login
		grp.POST("login",
			utils.CreateOperationOption("User login.",
				[]errors.Status{
					// Custom errors
					errors.UserNotVerified,
					// Common errors
					errors.BadRequest,
					errors.Unauthorized,
					errors.InternalServerError,
				}, false),
			// handler
			tonic.Handler(handlers.LoginUser, 200))

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
