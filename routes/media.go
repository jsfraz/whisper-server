package routes

import (
	"jsfraz/whisper-server/handlers"
	"jsfraz/whisper-server/middlewares"
	"jsfraz/whisper-server/utils"
	"net/http"

	"github.com/loopfz/gadgeto/tonic"
	"github.com/wI2L/fizz"
)

// Sets media route group.
//
//	@param grp
func MediaRoute(g *fizz.RouterGroup) {

	grp := g.Group("media", "Media", "Operations associated with encrypted media files.")
	grp.Use(middlewares.AuthMiddleware)
	grp.Use(middlewares.UserDeletionMiddleware)

	// Upload encrypted media file
	grp.POST("",
		utils.CreateOperationOption(
			"Upload encrypted media file",
			"Uploads an end-to-end encrypted media file bound to the receiver. Returns the media id.",
			[]int{
				http.StatusBadRequest,
				http.StatusUnauthorized,
				http.StatusRequestEntityTooLarge,
				http.StatusInternalServerError,
			},
			true),
		tonic.Handler(handlers.UploadMedia, http.StatusOK),
	)

	// Download encrypted media file by id (raw gin handler so ciphertext can be streamed).
	// The file is kept until the client confirms the download (see DELETE below) or its TTL expires.
	grp.GET(":id",
		utils.CreateOperationOption(
			"Download encrypted media file by id",
			"Downloads the encrypted media file. Only the intended receiver may download it. The file is retained until the client confirms the download or its TTL expires.",
			[]int{
				http.StatusUnauthorized,
				http.StatusForbidden,
				http.StatusNotFound,
				http.StatusInternalServerError,
			},
			true),
		handlers.DownloadMedia,
	)

	// Confirm download and delete the media file from the server.
	grp.DELETE(":id",
		utils.CreateOperationOption(
			"Confirm media download and delete it",
			"Confirms the encrypted media file has been downloaded and deletes it from the server. Only the intended receiver may confirm. Idempotent.",
			[]int{
				http.StatusUnauthorized,
				http.StatusForbidden,
				http.StatusInternalServerError,
			},
			true),
		tonic.Handler(handlers.ConfirmMediaDownload, http.StatusNoContent),
	)
}
