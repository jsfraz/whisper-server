package routes

import (
	"jsfraz/whisper-server/handlers"
	"jsfraz/whisper-server/middlewares"
	"jsfraz/whisper-server/utils"
	"net/http"

	"github.com/loopfz/gadgeto/tonic"
	"github.com/wI2L/fizz"
)

// Sets invite route group.
//
//	@param grp
func InviteRoute(g *fizz.RouterGroup) {

	grp := g.Group("invite", "Invite", "Operations associated with registration invites.")
	grp.Use(middlewares.Auth)

	// Create registration invite
	grp.POST("",
		utils.CreateOperationOption(
			"Create registration invite",
			"",
			[]int{
				http.StatusBadRequest,
				http.StatusUnauthorized,
				http.StatusInternalServerError,
			},
			true),
		tonic.Handler(handlers.CreateInvite, http.StatusOK),
	)

	// Get all active registration invites
	grp.GET("all",
		utils.CreateOperationOption(
			"Get all active registration invites",
			"",
			[]int{
				http.StatusBadRequest,
				http.StatusUnauthorized,
				http.StatusInternalServerError,
			},
			true),
		tonic.Handler(handlers.GetAllInvites, http.StatusOK),
	)
}
