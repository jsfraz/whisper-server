package handlers

import (
	"errors"
	"io"
	"jsfraz/shept-server/database"
	"jsfraz/shept-server/models"
	"jsfraz/shept-server/utils"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"

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
	// Bound the request body before parsing. Multipart parsing spools whatever
	// the client sends to a temp file, and the declared part size is attacker
	// supplied, so the size check below can only run after the bytes are on disk.
	maxUpload := utils.GetSingleton().Config.MediaMaxUploadSize
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxUpload+(1<<20))

	var request models.UploadMediaInput
	if err := c.ShouldBind(&request); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": utils.ErrorMediaFileTooLarge})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": utils.ErrorInvalidRequest})
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
	sender := senderId.(string)

	// Enforce maximum upload size
	maxUpload := utils.GetSingleton().Config.MediaMaxUploadSize
	if request.File.Size > maxUpload {
		return nil, c.AbortWithError(http.StatusRequestEntityTooLarge, errors.New(utils.ErrorMediaFileTooLarge))
	}

	if err := validateMediaReceiver(c, sender, request.ReceiverId); err != nil {
		return nil, err
	}
	if err := checkMediaQuota(c, sender); err != nil {
		return nil, err
	}

	// Generate unguessable id and persist the ciphertext to disk
	id := uuid.New().String()
	src, err := request.File.Open()
	if err != nil {
		return nil, c.AbortWithError(http.StatusInternalServerError, err)
	}
	defer src.Close()

	path, err := database.MediaFilePath(id)
	if err != nil {
		return nil, c.AbortWithError(http.StatusInternalServerError, err)
	}
	// Write under the temp suffix first; the file is renamed into place only once its
	// metadata exists, so the media sweeper can never delete an upload in flight.
	tmpPath := path + database.MediaUploadTempSuffix
	//nolint:gosec // path is built from a server-generated media id
	dst, err := os.Create(tmpPath)
	if err != nil {
		return nil, c.AbortWithError(http.StatusInternalServerError, err)
	}
	// LimitReader is the real cap: the declared size is not trusted.
	written, err := io.Copy(dst, io.LimitReader(src, maxUpload+1))
	if err != nil {
		_ = dst.Close()
		_ = os.Remove(tmpPath)
		return nil, c.AbortWithError(http.StatusInternalServerError, err)
	}
	if written > maxUpload {
		_ = dst.Close()
		_ = os.Remove(tmpPath)
		return nil, c.AbortWithError(http.StatusRequestEntityTooLarge, errors.New(utils.ErrorMediaFileTooLarge))
	}
	if err := dst.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return nil, c.AbortWithError(http.StatusInternalServerError, err)
	}

	// Store metadata in Valkey with TTL
	meta := models.NewMediaMetadata(id, sender, request.ReceiverId, written)
	if err := database.StoreMediaMetadata(meta, utils.GetSingleton().Config.MediaTtl); err != nil {
		_ = os.Remove(tmpPath)
		return nil, c.AbortWithError(http.StatusInternalServerError, err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		_ = database.DeleteMedia(id)
		return nil, c.AbortWithError(http.StatusInternalServerError, err)
	}

	// Index the file per participant, so cleanup does not have to scan every
	// media key in Valkey.
	if err := database.IndexUserMedia(sender, request.ReceiverId, id, utils.GetSingleton().Config.MediaTtl); err != nil {
		slog.Warn("failed to index media for cleanup", "media_id", id, "error", utils.SanitizeLogValue(err.Error()))
	}

	return models.NewMediaUploadResponse(id), nil
}

// validateMediaReceiver rejects a receiver the sender is not allowed to store
// media for. Shared by the one-shot and the chunked upload.
func validateMediaReceiver(c *gin.Context, senderId string, receiverId string) error {
	// Can not send media to self (in production)
	if senderId == receiverId && utils.GetSingleton().Config.GinMode == "release" {
		return c.AbortWithError(http.StatusBadRequest, errors.New(utils.ErrorCannotSendToSelf))
	}

	// Check if receiver is being deleted
	toDelete, err := database.WillUserBeDeleted(receiverId)
	if err != nil {
		return c.AbortWithError(http.StatusInternalServerError, err)
	}
	if toDelete {
		return c.AbortWithError(http.StatusBadRequest, errors.New(utils.ErrorCannotSendToUser))
	}

	// Check if receiver exists
	exists, err := database.UserExistsById(receiverId)
	if err != nil {
		return c.AbortWithError(http.StatusInternalServerError, err)
	}
	if !exists {
		return c.AbortWithError(http.StatusBadRequest, errors.New(utils.ErrorUserNotFound))
	}
	return nil
}

// checkMediaQuota enforces MEDIA_MAX_PER_USER: one account must not fill the
// disk for everyone. In-progress chunked uploads count, since their bytes are
// already on disk.
func checkMediaQuota(c *gin.Context, senderId string) error {
	maxPerUser := utils.GetSingleton().Config.MediaMaxPerUser
	if maxPerUser <= 0 {
		return nil
	}
	count, err := database.UserMediaCount(senderId)
	if err != nil {
		return c.AbortWithError(http.StatusInternalServerError, err)
	}
	if count >= int64(maxPerUser) {
		return c.AbortWithError(http.StatusTooManyRequests, errors.New(utils.ErrorTooManyPendingMediaFiles))
	}
	return nil
}

// mediaUploadLocks serializes the appends of one upload session. The server is a
// single process, so an in-process map is enough; two chunks arriving at the same
// offset would otherwise interleave into a corrupt file.
var mediaUploadLocks sync.Map

// mediaUploadLock returns the append lock of one upload session.
func mediaUploadLock(id string) *sync.Mutex {
	lock, _ := mediaUploadLocks.LoadOrStore(id, &sync.Mutex{})
	return lock.(*sync.Mutex)
}

// CreateMediaUpload starts a chunked upload. The ciphertext is appended chunk by
// chunk (see UploadMediaChunk) and becomes a media file on CompleteMediaUpload,
// so a large attachment survives a connection that breaks halfway.
//
//	@param c
//	@param request
//	@return error
func CreateMediaUpload(c *gin.Context, request *models.MediaUploadSessionInput) error {
	senderId, _ := c.Get("userId")
	sender := senderId.(string)

	maxUpload := utils.GetSingleton().Config.MediaMaxUploadSize
	if request.TotalSize > maxUpload {
		return c.AbortWithError(http.StatusRequestEntityTooLarge, errors.New(utils.ErrorMediaFileTooLarge))
	}
	if err := validateMediaReceiver(c, sender, request.ReceiverId); err != nil {
		return err
	}
	if err := checkMediaQuota(c, sender); err != nil {
		return err
	}

	id := uuid.New().String()
	path, err := database.MediaFilePath(id)
	if err != nil {
		return c.AbortWithError(http.StatusInternalServerError, err)
	}
	// Create the partial file up front, so an early chunk never has to decide
	// whether the file or the session is missing.
	//nolint:gosec // path is built from a server-generated media id
	partial, err := os.Create(path + database.MediaUploadTempSuffix)
	if err != nil {
		return c.AbortWithError(http.StatusInternalServerError, err)
	}
	if err := partial.Close(); err != nil {
		_ = os.Remove(path + database.MediaUploadTempSuffix)
		return c.AbortWithError(http.StatusInternalServerError, err)
	}

	session := models.NewMediaUploadSession(id, sender, request.ReceiverId, request.TotalSize)
	ttl := utils.GetSingleton().Config.MediaUploadTtl
	if err := database.StoreMediaUploadSession(session, ttl); err != nil {
		_ = os.Remove(path + database.MediaUploadTempSuffix)
		return c.AbortWithError(http.StatusInternalServerError, err)
	}
	// Indexed like a finished file, so the quota counts it and account deletion
	// removes the partial bytes too.
	if err := database.IndexUserMedia(sender, request.ReceiverId, id, ttl); err != nil {
		slog.Warn("failed to index upload for cleanup", "media_id", id, "error", utils.SanitizeLogValue(err.Error()))
	}

	c.JSON(http.StatusOK, models.NewMediaUploadSessionResponse(session))
	return nil
}

// UploadMediaChunk appends one chunk of the ciphertext at the Upload-Offset
// header. A chunk that does not start at the stored offset is refused with 409
// and the offset the server holds, which is how a client resumes.
//
// Registered as a raw gin handler: the body is multipart/form-data (chunk).
// There is deliberately no per-request rate limit here: one file is many chunks,
// so it would throttle exactly the large uploads this endpoint exists for; the
// per-account quota is the bound that matters.
//
//	@param c
func UploadMediaChunk(c *gin.Context) {
	id := c.Param("id")
	maxUpload := utils.GetSingleton().Config.MediaMaxUploadSize
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxUpload+(1<<20))

	offset, err := strconv.ParseInt(c.GetHeader("Upload-Offset"), 10, 64)
	if err != nil || offset < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": utils.ErrorInvalidRequest})
		return
	}

	userId, _ := c.Get("userId")
	sender := userId.(string)

	lock := mediaUploadLock(id)
	lock.Lock()
	defer lock.Unlock()

	found, session, err := database.GetMediaUploadSession(id)
	if err != nil {
		slog.Error("failed to load upload session", "media_id", utils.SanitizeLogValue(id), "error", utils.SanitizeLogValue(err.Error()))
		c.JSON(http.StatusInternalServerError, gin.H{"error": utils.ErrorInternal})
		return
	}
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": utils.ErrorMediaUploadNotFound})
		return
	}
	if session.SenderId != sender {
		c.JSON(http.StatusForbidden, gin.H{"error": utils.ErrorNotAuthorized})
		return
	}
	if offset != session.Received {
		c.JSON(http.StatusConflict, gin.H{
			"error":    utils.ErrorUploadOffsetMismatch,
			"received": session.Received,
		})
		return
	}

	src, header, err := c.Request.FormFile("chunk")
	if err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": utils.ErrorMediaFileTooLarge})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": utils.ErrorInvalidRequest})
		return
	}
	defer func() { _ = src.Close() }()

	remaining := session.TotalSize - session.Received
	if header.Size > remaining {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": utils.ErrorMediaFileTooLarge})
		return
	}

	// Parse to a canonical UUID before the value becomes a path, so what is
	// opened on disk can only ever be a plain id.
	mediaID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": utils.ErrorInvalidMediaId})
		return
	}
	path := filepath.Join(utils.GetSingleton().Config.MediaDir, mediaID.String())
	// Open without O_APPEND: the offset is the authority, and the lock above
	// keeps two writers apart.
	dst, err := os.OpenFile(path+database.MediaUploadTempSuffix, os.O_WRONLY, 0) //nolint:gosec // path from a parsed UUID
	if err != nil {
		// The session outlived its bytes (swept or removed): the client restarts.
		_ = database.DeleteMediaUploadSession(id)
		c.JSON(http.StatusNotFound, gin.H{"error": utils.ErrorMediaUploadNotFound})
		return
	}
	defer func() { _ = dst.Close() }()
	if _, err := dst.Seek(offset, io.SeekStart); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": utils.ErrorInternal})
		return
	}
	written, err := io.Copy(dst, io.LimitReader(src, remaining+1))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": utils.ErrorInternal})
		return
	}
	if written > remaining {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": utils.ErrorMediaFileTooLarge})
		return
	}

	session.Received = offset + written
	if err := database.StoreMediaUploadSession(*session, utils.GetSingleton().Config.MediaUploadTtl); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": utils.ErrorInternal})
		return
	}
	c.JSON(http.StatusOK, models.NewMediaUploadSessionResponse(*session))
}

// GetMediaUpload reports how much of a chunked upload the server holds.
//
//	@param c
//	@param request
//	@return error
func GetMediaUpload(c *gin.Context, request *models.MediaUploadIdUri) error {
	userId, _ := c.Get("userId")
	sender := userId.(string)

	found, session, err := database.GetMediaUploadSession(request.Id)
	if err != nil {
		return c.AbortWithError(http.StatusInternalServerError, err)
	}
	if !found {
		return c.AbortWithError(http.StatusNotFound, errors.New(utils.ErrorMediaUploadNotFound))
	}
	if session.SenderId != sender {
		return c.AbortWithError(http.StatusForbidden, errors.New(utils.ErrorNotAuthorized))
	}
	c.JSON(http.StatusOK, models.NewMediaUploadSessionResponse(*session))
	return nil
}

// CompleteMediaUpload turns a fully received chunked upload into a media file
// the receiver can download. Only the sender may complete it, and only once
// every byte is in.
//
//	@param c
//	@param request
//	@return error
func CompleteMediaUpload(c *gin.Context, request *models.MediaUploadIdUri) error {
	userId, _ := c.Get("userId")
	sender := userId.(string)

	lock := mediaUploadLock(request.Id)
	lock.Lock()
	defer lock.Unlock()

	found, session, err := database.GetMediaUploadSession(request.Id)
	if err != nil {
		return c.AbortWithError(http.StatusInternalServerError, err)
	}
	if !found {
		return c.AbortWithError(http.StatusNotFound, errors.New(utils.ErrorMediaUploadNotFound))
	}
	if session.SenderId != sender {
		return c.AbortWithError(http.StatusForbidden, errors.New(utils.ErrorNotAuthorized))
	}
	if session.Received != session.TotalSize {
		return c.AbortWithError(http.StatusBadRequest, errors.New(utils.ErrorUploadIncomplete))
	}

	path, err := database.MediaFilePath(request.Id)
	if err != nil {
		return c.AbortWithError(http.StatusInternalServerError, err)
	}
	tmpPath := path + database.MediaUploadTempSuffix
	if _, err := os.Stat(tmpPath); err != nil {
		_ = database.DeleteMediaUploadSession(request.Id)
		return c.AbortWithError(http.StatusNotFound, errors.New(utils.ErrorMediaUploadNotFound))
	}

	// Same order as the one-shot upload: metadata first, so the sweeper never
	// sees a finished file without one.
	meta := models.NewMediaMetadata(request.Id, session.SenderId, session.ReceiverId, session.Received)
	if err := database.StoreMediaMetadata(meta, utils.GetSingleton().Config.MediaTtl); err != nil {
		return c.AbortWithError(http.StatusInternalServerError, err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = database.DeleteMedia(request.Id)
		return c.AbortWithError(http.StatusInternalServerError, err)
	}
	if err := database.DeleteMediaUploadSession(request.Id); err != nil {
		slog.Warn("failed to delete upload session", "media_id", request.Id, "error", utils.SanitizeLogValue(err.Error()))
	}
	mediaUploadLocks.Delete(request.Id)

	c.JSON(http.StatusOK, models.NewMediaUploadResponse(request.Id))
	return nil
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
		slog.Error("failed to load media metadata", "media_id", utils.SanitizeLogValue(id), "error", utils.SanitizeLogValue(err.Error()))
		c.JSON(http.StatusInternalServerError, gin.H{"error": utils.ErrorInternal})
		return
	}
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": utils.ErrorMediaNotFound})
		return
	}

	// Only the intended receiver may download
	if meta.ReceiverId != userId.(string) {
		c.JSON(http.StatusForbidden, gin.H{"error": utils.ErrorNotAuthorized})
		return
	}

	// Open file
	path, err := database.MediaFilePath(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": utils.ErrorInvalidMediaId})
		return
	}
	file, err := os.Open(path)
	if err != nil {
		// File missing but metadata present: clean up the orphaned key
		_ = database.DeleteMedia(id)
		c.JSON(http.StatusNotFound, gin.H{"error": utils.ErrorMediaNotFound})
		return
	}

	// The header comes from the file itself: metadata can drift, and a wrong
	// length truncates the download without an error.
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		slog.Error("failed to stat media file", "media_id", utils.SanitizeLogValue(id), "error", utils.SanitizeLogValue(err.Error()))
		c.JSON(http.StatusInternalServerError, gin.H{"error": utils.ErrorInternal})
		return
	}
	// Stream ciphertext to the client (deletion happens on explicit client confirmation)
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Disposition", "attachment; filename=\""+id+"\"")
	c.Header("Content-Length", strconv.FormatInt(info.Size(), 10))
	if _, err := io.Copy(c.Writer, file); err != nil {
		slog.Warn("media download aborted", "media_id", utils.SanitizeLogValue(id), "error", utils.SanitizeLogValue(err.Error()))
	}
	if err := file.Close(); err != nil {
		slog.Warn("failed to close media file", "media_id", utils.SanitizeLogValue(id), "error", utils.SanitizeLogValue(err.Error()))
	}
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
	if meta.ReceiverId != userId.(string) {
		return c.AbortWithError(http.StatusForbidden, errors.New(utils.ErrorNotAuthorized))
	}

	// Delete file and metadata
	if err := database.DeleteMedia(request.Id); err != nil {
		return c.AbortWithError(http.StatusInternalServerError, err)
	}
	return nil
}
