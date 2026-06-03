package handlers

import (
	"errors"
	"io"
	"jsfraz/whisper-server/database"
	"jsfraz/whisper-server/models"
	"jsfraz/whisper-server/utils"
	"net/http"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Upload an encrypted media file. The server stores only the ciphertext (zero-knowledge)
// and binds it to the intended receiver. The file is deleted after the receiver confirms
// download or when its Valkey TTL expires.
//
// Registered as a raw gin handler: the body is multipart/form-data (file + receiverId).
// Tonic would bind application/json and fail on multipart boundaries ("invalid character '-'…").
//
//	@param c
func UploadMedia(c *gin.Context) {
	var request models.UploadMediaInput
	if err := c.ShouldBind(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	response, err := uploadMediaFromInput(c, &request)
	if err != nil {
		return
	}
	c.JSON(http.StatusOK, response)
}

func uploadMediaFromInput(c *gin.Context, request *models.UploadMediaInput) (*models.MediaUploadResponse, error) {
	senderId, _ := c.Get("userId")
	senderIdUint := senderId.(uint64)

	// Enforce maximum upload size
	if request.File.Size > utils.GetSingleton().Config.MediaMaxUploadSize {
		return nil, c.AbortWithError(http.StatusRequestEntityTooLarge, errors.New("media file too large"))
	}

	// Can not send media to self (in production)
	if senderIdUint == request.ReceiverId && utils.GetSingleton().Config.GinMode == "release" {
		return nil, c.AbortWithError(http.StatusBadRequest, errors.New("can not send media to self"))
	}

	// Check if receiver is being deleted
	toDelete, err := database.WillUserBeDeleted(request.ReceiverId)
	if err != nil {
		return nil, c.AbortWithError(http.StatusInternalServerError, err)
	}
	if toDelete {
		return nil, c.AbortWithError(http.StatusBadRequest, errors.New("can not send media to this user"))
	}

	// Check if receiver exists
	exists, err := database.UserExistsById(request.ReceiverId)
	if err != nil {
		return nil, c.AbortWithError(http.StatusInternalServerError, err)
	}
	if !exists {
		return nil, c.AbortWithError(http.StatusBadRequest, errors.New("user does not exist"))
	}

	// Generate unguessable id and persist the ciphertext to disk
	id := uuid.New().String()
	src, err := request.File.Open()
	if err != nil {
		return nil, c.AbortWithError(http.StatusInternalServerError, err)
	}
	defer src.Close()

	path := database.MediaFilePath(id)
	dst, err := os.Create(path)
	if err != nil {
		return nil, c.AbortWithError(http.StatusInternalServerError, err)
	}
	written, err := io.Copy(dst, src)
	if err != nil {
		dst.Close()
		os.Remove(path)
		return nil, c.AbortWithError(http.StatusInternalServerError, err)
	}
	if err := dst.Close(); err != nil {
		os.Remove(path)
		return nil, c.AbortWithError(http.StatusInternalServerError, err)
	}

	// Store metadata in Valkey with TTL
	meta := models.NewMediaMetadata(id, senderIdUint, request.ReceiverId, written)
	if err := database.StoreMediaMetadata(meta, utils.GetSingleton().Config.MediaTtl); err != nil {
		os.Remove(path)
		return nil, c.AbortWithError(http.StatusInternalServerError, err)
	}

	return models.NewMediaUploadResponse(id), nil
}

// Download an encrypted media file by id. Only the intended receiver may download it.
//
// The file is NOT deleted here: a server-side write success does not guarantee the
// client received, decrypted and persisted the (large) file. The client must confirm
// with ConfirmMediaDownload once it has the file, which makes downloads retriable.
// The Valkey TTL remains a backstop if the client never confirms.
//
// Registered as a raw gin handler so the ciphertext can be streamed directly.
//
//	@param c
func DownloadMedia(c *gin.Context) {
	id := c.Param("id")
	userId, _ := c.Get("userId")

	// Load metadata
	found, meta, err := database.GetMediaMetadata(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "media not found"})
		return
	}

	// Only the intended receiver may download
	if meta.ReceiverId != userId.(uint64) {
		c.JSON(http.StatusForbidden, gin.H{"error": "not authorized to download this media"})
		return
	}

	// Open file
	path := database.MediaFilePath(id)
	file, err := os.Open(path)
	if err != nil {
		// File missing but metadata present: clean up the orphaned key
		database.DeleteMedia(id)
		c.JSON(http.StatusNotFound, gin.H{"error": "media not found"})
		return
	}

	// Stream ciphertext to the client (deletion happens on explicit client confirmation)
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Disposition", "attachment; filename=\""+id+"\"")
	c.Header("Content-Length", strconv.FormatInt(meta.Size, 10))
	_, _ = io.Copy(c.Writer, file)
	file.Close()
}

// Confirm an encrypted media file has been downloaded and delete it from the server.
// Only the intended receiver may confirm. Idempotent: confirming an already
// deleted/expired media succeeds.
//
//	@param c
//	@param request
//	@return error
func ConfirmMediaDownload(c *gin.Context, request *models.MediaIdUri) error {
	userId, _ := c.Get("userId")

	// Load metadata
	found, meta, err := database.GetMediaMetadata(request.Id)
	if err != nil {
		return c.AbortWithError(http.StatusInternalServerError, err)
	}
	// Already deleted or expired: nothing to do
	if !found {
		return nil
	}

	// Only the intended receiver may delete
	if meta.ReceiverId != userId.(uint64) {
		return c.AbortWithError(http.StatusForbidden, errors.New("not authorized to delete this media"))
	}

	// Delete file and metadata
	if err := database.DeleteMedia(request.Id); err != nil {
		return c.AbortWithError(http.StatusInternalServerError, err)
	}
	return nil
}
