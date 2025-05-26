package routes

import (
	"jsfraz/whisper-server/handlers"
	"jsfraz/whisper-server/middlewares"
	"jsfraz/whisper-server/utils"
	"net/http"

	"github.com/loopfz/gadgeto/tonic"
	"github.com/wI2L/fizz"
)

// Sets Firebase route.
//
//	@param grp
func FirebaseRoute(g *fizz.RouterGroup) {

	grp := g.Group("firebase", "Firebase", "Operations associated with Firebase.")
	grp.Use(middlewares.AuthMiddleware)
	grp.Use(middlewares.UserDeletionMiddleware)

	// Set client Firebase token
	grp.PATCH("token",
		utils.CreateOperationOption(
			"Set client Firebase token",
			"",
			[]int{
				http.StatusBadRequest,
				http.StatusUnauthorized,
				http.StatusInternalServerError,
			},
			true),
		tonic.Handler(handlers.SetFirebaseToken, http.StatusOK),
	)
}
