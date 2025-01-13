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
func UserRoute(g *fizz.RouterGroup) {

	grp := g.Group("user", "User", "Operations associated with a user account.")
	grp.Use(middlewares.AuthMiddleware)

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

	// Get all users
	grp.GET("all",
		utils.CreateOperationOption(
			"Get all users except the user",
			"",
			[]int{
				http.StatusBadRequest,
				http.StatusUnauthorized,
				http.StatusInternalServerError,
			},
			true),
		tonic.Handler(handlers.GetAllUsers, http.StatusOK),
	)

	// Delete users
	grp.PATCH("",
		utils.CreateOperationOption(
			"Delete users",
			"",
			[]int{
				http.StatusBadRequest,
				http.StatusUnauthorized,
				http.StatusInternalServerError,
			},
			true),
		tonic.Handler(handlers.DeleteUsers, http.StatusOK),
	)

	// Search users
	grp.GET("search",
		utils.CreateOperationOption("Search users",
			"",
			[]int{
				http.StatusBadRequest,
				http.StatusUnauthorized,
				http.StatusInternalServerError,
			},
			true),
		tonic.Handler(handlers.SearchUsers, http.StatusOK),
	)
}
