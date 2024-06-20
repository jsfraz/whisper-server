package routes

import (
	"jsfraz/whisper-server/handlers"
	"jsfraz/whisper-server/middlewares"
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
	grp.Use(middlewares.Auth)

	// Refresh
	grp.GET("whoAmI",
		utils.CreateOperationOption("Get current user.",
			[]int{
				http.StatusBadRequest,
				http.StatusInternalServerError,
			}, false),
		// Handler
		tonic.Handler(handlers.WhoAmI, 200))
}
