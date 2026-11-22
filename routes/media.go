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
		// Each request may write 25 MiB to disk.
		middlewares.RateLimit(utils.GetSingleton().Config.RateLimitBurst, utils.GetSingleton().Config.RateLimitPerMinute),
		handlers.UploadMedia,
	)

	// Chunked upload: a large attachment is sent in pieces and resumes at the
	// offset the server reports.
	grp.POST("/uploads",
		utils.CreateOperationOption(
			"Start a chunked upload",
			"Starts a resumable upload of an end-to-end encrypted media file bound to the receiver. Returns the upload id and the offset the server holds.",
			[]int{
				http.StatusBadRequest,
				http.StatusUnauthorized,
				http.StatusRequestEntityTooLarge,
				http.StatusTooManyRequests,
				http.StatusInternalServerError,
			},
			true),
		// A session is one request; its chunks are not rate limited (see UploadMediaChunk).
		middlewares.RateLimit(utils.GetSingleton().Config.RateLimitBurst, utils.GetSingleton().Config.RateLimitPerMinute),
		tonic.Handler(handlers.CreateMediaUpload, http.StatusOK),
	)

	// Append one chunk of a chunked upload.
	grp.PUT("/uploads/:id",
		utils.CreateOperationOption(
			"Upload one chunk",
			"Appends one chunk of the encrypted media file at the Upload-Offset header. A chunk at a different offset is refused with 409 and the offset the server holds.",
			[]int{
				http.StatusBadRequest,
				http.StatusUnauthorized,
				http.StatusForbidden,
				http.StatusNotFound,
				http.StatusConflict,
				http.StatusRequestEntityTooLarge,
				http.StatusInternalServerError,
			},
			true),
		handlers.UploadMediaChunk,
	)

	// Report how much of a chunked upload the server holds.
	grp.GET("/uploads/:id",
		utils.CreateOperationOption(
			"Get chunked upload state",
			"Returns how much of the encrypted media file the server holds, so a client can resume a broken upload.",
			[]int{
				http.StatusUnauthorized,
				http.StatusForbidden,
				http.StatusNotFound,
				http.StatusInternalServerError,
			},
			true),
		tonic.Handler(handlers.GetMediaUpload, http.StatusOK),
	)

	// Turn a fully received chunked upload into a downloadable media file.
	grp.POST("/uploads/:id/complete",
		utils.CreateOperationOption(
			"Complete a chunked upload",
			"Turns a fully received chunked upload into a media file the receiver can download. Only the sender may complete it, and only once every byte is in.",
			[]int{
				http.StatusBadRequest,
				http.StatusUnauthorized,
				http.StatusForbidden,
				http.StatusNotFound,
				http.StatusInternalServerError,
			},
			true),
		tonic.Handler(handlers.CompleteMediaUpload, http.StatusOK),
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
