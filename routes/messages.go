package routes

import (
	"jsfraz/whisper-server/handlers"
	"jsfraz/whisper-server/middlewares"
	"jsfraz/whisper-server/utils"
	"net/http"

	"github.com/loopfz/gadgeto/tonic"
	"github.com/wI2L/fizz"
)

// Sets the message history route group.
//
//	@param g
func MessageRoute(g *fizz.RouterGroup) {
	grp := g.Group("messages", "Messages", "Ciphertext history of a conversation.")
	grp.Use(middlewares.AuthMiddleware)
	grp.Use(middlewares.UserDeletionMiddleware)

	grp.GET("",
		utils.CreateOperationOption(
			"Get message history",
			"One page of a conversation from the ciphertext archive, newest first. The server never sees the plaintext.",
			[]int{
				http.StatusBadRequest,
				http.StatusUnauthorized,
				http.StatusInternalServerError,
			},
			true),
		tonic.Handler(handlers.GetHistory, http.StatusOK),
	)
}
