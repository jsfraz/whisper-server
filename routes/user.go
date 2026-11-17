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
	grp.Use(middlewares.UserDeletionMiddleware)

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

	// Get user by ID
	grp.GET("",
		utils.CreateOperationOption(
			"Get user by ID",
			"",
			[]int{
				http.StatusBadRequest,
				http.StatusUnauthorized,
				http.StatusNotFound,
				http.StatusInternalServerError,
			},
			true),
		tonic.Handler(handlers.GetUserById, http.StatusOK),
	)

	// Delete user
	grp.DELETE("me",
		utils.CreateOperationOption(
			"Delete my account",
			"",
			[]int{
				http.StatusBadRequest,
				http.StatusUnauthorized,
				http.StatusInternalServerError,
			},
			true),
		tonic.Handler(handlers.DeleteMe, http.StatusNoContent),
	)

	// Devices of the account
	grp.GET("devices",
		utils.CreateOperationOption(
			"Get devices",
			"Lists the devices signed into the account. The requesting device is flagged.",
			[]int{
				http.StatusUnauthorized,
				http.StatusInternalServerError,
			},
			true),
		tonic.Handler(handlers.GetDevices, http.StatusOK),
	)

	grp.POST("devices/:id/revoke",
		utils.CreateOperationOption(
			"Revoke a device",
			"Locks a device out immediately: it cannot log in, renew its token, call the API or keep its WebSocket.",
			[]int{
				http.StatusBadRequest,
				http.StatusUnauthorized,
				http.StatusNotFound,
				http.StatusInternalServerError,
			},
			true),
		tonic.Handler(handlers.RevokeDevice, http.StatusNoContent),
	)

	grp.POST("devices/:id/unrevoke",
		utils.CreateOperationOption(
			"Unrevoke a device",
			"Lets a revoked device sign in again. Only a live device of the same account may do this.",
			[]int{
				http.StatusBadRequest,
				http.StatusUnauthorized,
				http.StatusNotFound,
				http.StatusInternalServerError,
			},
			true),
		tonic.Handler(handlers.UnrevokeDevice, http.StatusNoContent),
	)

	// Archive retention
	grp.GET("retention",
		utils.CreateOperationOption(
			"Get archive retention",
			"Returns how long the server keeps the ciphertext archive of this account.",
			[]int{
				http.StatusUnauthorized,
				http.StatusInternalServerError,
			},
			true),
		tonic.Handler(handlers.GetRetention, http.StatusOK),
	)

	grp.PUT("retention",
		utils.CreateOperationOption(
			"Set archive retention",
			"The shorter retention of both participants wins, because both accounts can read a message.",
			[]int{
				http.StatusBadRequest,
				http.StatusUnauthorized,
				http.StatusInternalServerError,
			},
			true),
		tonic.Handler(handlers.SetRetention, http.StatusOK),
	)
}
