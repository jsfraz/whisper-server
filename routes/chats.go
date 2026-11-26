package routes

import (
	"jsfraz/shept-server/handlers"
	"jsfraz/shept-server/middlewares"
	"jsfraz/shept-server/utils"
	"net/http"

	"github.com/loopfz/gadgeto/tonic"
	"github.com/wI2L/fizz"
)

// Sets the shared conversation state route group. It is separate from the
// message routes because "read" is a static segment that would collide with the
// ":id" wildcard there.
//
//	@param g
func ChatsRoute(g *fizz.RouterGroup) {
	grp := g.Group("chats", "Chats", "Shared state of a conversation.")
	grp.Use(middlewares.AuthMiddleware)
	grp.Use(middlewares.UserDeletionMiddleware)

	grp.PUT(":peerId/read",
		utils.CreateOperationOption(
			"Mark a conversation read",
			"Moves the caller's read cursor of a conversation and tells the other participant. Only the instant is stored: the server never learns which message was opened, and a client that keeps receipts off never calls this. Idempotent, and it never moves backwards.",
			[]int{
				http.StatusBadRequest,
				http.StatusUnauthorized,
				http.StatusInternalServerError,
			},
			true),
		tonic.Handler(handlers.SetReadCursor, http.StatusNoContent),
	)
}
