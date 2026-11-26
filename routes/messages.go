package routes

import (
	"jsfraz/shept-server/handlers"
	"jsfraz/shept-server/middlewares"
	"jsfraz/shept-server/utils"
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
			"Erases the ciphertext of a message the caller sent, unpins it and logs the tombstone, so both sides learn about it. Idempotent.",
			[]int{
				http.StatusBadRequest,
				http.StatusUnauthorized,
				http.StatusForbidden,
				http.StatusInternalServerError,
			},
			true),
		tonic.Handler(handlers.DeleteMessage, http.StatusNoContent),
	)

	// Replace the ciphertext of one own message.
	grp.PATCH(":id",
		utils.CreateOperationOption(
			"Edit my message",
			"Replaces the ciphertext of a message the caller sent with a new version and stamps the edit. Both sides learn about it through sync and a live edit frame. The server never sees the plaintext.",
			[]int{
				http.StatusBadRequest,
				http.StatusUnauthorized,
				http.StatusForbidden,
				http.StatusNotFound,
				http.StatusInternalServerError,
			},
			true),
		tonic.Handler(handlers.EditMessage, http.StatusOK),
	)

	// Set the caller's reaction on a message.
	grp.PUT(":id/reaction",
		utils.CreateOperationOption(
			"React to a message",
			"Stores the caller's encrypted reaction on a message. A later reaction replaces it; both sides learn about it through sync and a live reaction frame.",
			[]int{
				http.StatusBadRequest,
				http.StatusUnauthorized,
				http.StatusForbidden,
				http.StatusNotFound,
				http.StatusInternalServerError,
			},
			true),
		tonic.Handler(handlers.SetReaction, http.StatusNoContent),
	)

	// Clear the caller's reaction on a message.
	grp.DELETE(":id/reaction",
		utils.CreateOperationOption(
			"Remove my reaction",
			"Clears the caller's reaction on a message. Idempotent.",
			[]int{
				http.StatusBadRequest,
				http.StatusUnauthorized,
				http.StatusForbidden,
				http.StatusNotFound,
				http.StatusInternalServerError,
			},
			true),
		tonic.Handler(handlers.RemoveReaction, http.StatusNoContent),
	)

	// Pin a message for both participants.
	grp.PUT(":id/pin",
		utils.CreateOperationOption(
			"Pin a message",
			"Pins a message of the chat for both participants. Anyone in the conversation may pin. Idempotent.",
			[]int{
				http.StatusBadRequest,
				http.StatusUnauthorized,
				http.StatusForbidden,
				http.StatusNotFound,
				http.StatusInternalServerError,
			},
			true),
		tonic.Handler(handlers.PinMessage, http.StatusNoContent),
	)

	// Remove a pin for both participants.
	grp.DELETE(":id/pin",
		utils.CreateOperationOption(
			"Unpin a message",
			"Removes a pin for both participants. Idempotent.",
			[]int{
				http.StatusBadRequest,
				http.StatusUnauthorized,
				http.StatusForbidden,
				http.StatusNotFound,
				http.StatusInternalServerError,
			},
			true),
		tonic.Handler(handlers.UnpinMessage, http.StatusNoContent),
	)

	// Record that the receiver opened a view-once attachment.
	grp.POST(":id/open",
		utils.CreateOperationOption(
			"Mark a view-once attachment opened",
			"Records that the receiver opened a view-once attachment and tells both participants through sync and a live open frame, so the message stays in the conversation as a tombstone. The view-once flag itself lives inside the ciphertext, so this is a client-reported state. Only the receiver may call it. Idempotent.",
			[]int{
				http.StatusBadRequest,
				http.StatusUnauthorized,
				http.StatusForbidden,
				http.StatusNotFound,
				http.StatusInternalServerError,
			},
			true),
		tonic.Handler(handlers.OpenMessage, http.StatusNoContent),
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
