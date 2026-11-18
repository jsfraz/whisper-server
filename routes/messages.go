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

	// Delete a whole conversation for the calling account only. The peer is a
	// query parameter, not a path segment: gin refuses a wildcard next to the
	// ":id" routes below.
	grp.DELETE("",
		utils.CreateOperationOption(
			"Delete a conversation for me",
			"Deletes a conversation for the calling account on every device: everything sent at or before the cut stops being visible to it. The other participant keeps the archive. Idempotent.",
			[]int{
				http.StatusBadRequest,
				http.StatusUnauthorized,
				http.StatusInternalServerError,
			},
			true),
		tonic.Handler(handlers.ClearConversation, http.StatusOK),
	)

	// Delete one own message for everyone.
	grp.DELETE(":id",
		utils.CreateOperationOption(
			"Delete my message for everyone",
			"Erases the ciphertext of a message the caller sent and leaves a tombstone, so the other side learns about it on the next sync. Idempotent.",
			[]int{
				http.StatusBadRequest,
				http.StatusUnauthorized,
				http.StatusForbidden,
				http.StatusNotFound,
				http.StatusInternalServerError,
			},
			true),
		tonic.Handler(handlers.DeleteMessage, http.StatusNoContent),
	)

	// Hide one message for the calling account only.
	grp.POST(":id/hide",
		utils.CreateOperationOption(
			"Hide a message for me",
			"Removes one message from the calling account on every device. The other participant keeps it. Idempotent.",
			[]int{
				http.StatusBadRequest,
				http.StatusUnauthorized,
				http.StatusForbidden,
				http.StatusNotFound,
				http.StatusInternalServerError,
			},
			true),
		tonic.Handler(handlers.HideMessage, http.StatusNoContent),
	)
}
