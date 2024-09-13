package routes

import (
	"jsfraz/whisper-server/handlers"
	"jsfraz/whisper-server/utils"
	"net/http"

	"github.com/loopfz/gadgeto/tonic"
	"github.com/wI2L/fizz"
)

// Sets user route group.
//
//	@param grp
func UserRoute(grp *fizz.RouterGroup) {

	// Use auth middleware
	// grp.Use(middlewares.Auth)

	/*
		// Who am I
		grp.GET("whoAmI",
			utils.CreateOperationOption("Get current user.",
				[]int{
					http.StatusBadRequest,
					http.StatusInternalServerError,
				}, false),
			// Handler
			tonic.Handler(handlers.WhoAmI, 200))
	*/

	// Create user
	grp.POST("",
		utils.CreateOperationOption(
			"Create user",
			"**Public key _MUST_ be passed without the newline characters.**",
			[]int{
				http.StatusBadRequest,
				http.StatusUnauthorized,
				http.StatusConflict,
				http.StatusInternalServerError,
			},
			false),
		// Handler
		tonic.Handler(handlers.CreateUser, http.StatusNoContent),
	)
}
