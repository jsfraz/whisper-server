package models

import (
	"encoding/json"
	"mime/multipart"
	"time"
)

// MediaMetadata is stored in Valkey and describes an encrypted media file kept on disk.
// The server is zero-knowledge: it never stores the media type or any plaintext,
// only the information required to authorize a download and clean up the file.
type MediaMetadata struct {
	Id         string    `json:"id"`
	SenderId   string    `json:"senderId"`
	ReceiverId string    `json:"receiverId"`
	Size       int64     `json:"size"`
	CreatedAt  time.Time `json:"createdAt"`
}

// Return new MediaMetadata.
//
//	@param id
//	@param senderId
//	@param receiverId
//	@param size
//	@return MediaMetadata
func NewMediaMetadata(id string, senderId string, receiverId string, size int64) MediaMetadata {
	return MediaMetadata{
		Id:         id,
		SenderId:   senderId,
		ReceiverId: receiverId,
		Size:       size,
		CreatedAt:  time.Now().UTC(),
	}
}

// Marshall MediaMetadata to binary.
//
//	@return []byte
//	@return error
func (m MediaMetadata) MarshalBinary() ([]byte, error) {
	return json.Marshal(m)
}

// Unmarshall MediaMetadata.
//
//	@param jsonBytes
//	@return *MediaMetadata
//	@return error
func MediaMetadataFromJson(jsonBytes []byte) (*MediaMetadata, error) {
	var m MediaMetadata
	err := json.Unmarshal(jsonBytes, &m)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// UploadMediaInput is the multipart/form-data body for uploading an encrypted media file.
type UploadMediaInput struct {
	File       *multipart.FileHeader `form:"file" binding:"required"`
	ReceiverId string                `form:"receiverId" binding:"required,uuid"`
}

// MediaUploadResponse is returned after a successful upload.
type MediaUploadResponse struct {
	Id string `json:"id"`
}

// Return new MediaUploadResponse.
//
//	@param id
//	@return *MediaUploadResponse
func NewMediaUploadResponse(id string) *MediaUploadResponse {
	return &MediaUploadResponse{
		Id: id,
	}
}

// MediaIdUri binds the media id from the download path.
type MediaIdUri struct {
	Id string `path:"id" validate:"required"`
}

// MediaUploadSession is a chunked upload in progress. The ciphertext is written
// to <MediaDir>/<id>.part until the client completes the upload, so a broken
// connection loses only the bytes that were not acknowledged.
type MediaUploadSession struct {
	V          int       `json:"v"`
	Id         string    `json:"id"`
	SenderId   string    `json:"senderId"`
	ReceiverId string    `json:"receiverId"`
	TotalSize  int64     `json:"totalSize"`
	Received   int64     `json:"received"`
	CreatedAt  time.Time `json:"createdAt"`
}

// Return new MediaUploadSession.
//
//	@param id
//	@param senderId
//	@param receiverId
//	@param totalSize
//	@return MediaUploadSession
func NewMediaUploadSession(id string, senderId string, receiverId string, totalSize int64) MediaUploadSession {
	return MediaUploadSession{
		V:          1,
		Id:         id,
		SenderId:   senderId,
		ReceiverId: receiverId,
		TotalSize:  totalSize,
		CreatedAt:  time.Now().UTC(),
	}
}

// Marshall MediaUploadSession to binary.
//
//	@return []byte
//	@return error
func (s MediaUploadSession) MarshalBinary() ([]byte, error) {
	return json.Marshal(s)
}

// Unmarshall MediaUploadSession.
//
//	@param jsonBytes
//	@return *MediaUploadSession
//	@return error
func MediaUploadSessionFromJson(jsonBytes []byte) (*MediaUploadSession, error) {
	var s MediaUploadSession
	if err := json.Unmarshal(jsonBytes, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

// MediaUploadSessionInput starts a chunked upload: the receiver it is bound to
// and the ciphertext length. The client knows the length before the first chunk.
type MediaUploadSessionInput struct {
	ReceiverId string `json:"receiverId" validate:"required,uuid"`
	TotalSize  int64  `json:"totalSize" validate:"required,gt=0"`
}

// MediaUploadSessionResponse reports how much of the ciphertext the server holds,
// which is what a client needs to resume a broken upload.
type MediaUploadSessionResponse struct {
	Id        string `json:"id"`
	Received  int64  `json:"received"`
	TotalSize int64  `json:"totalSize"`
}

// Return new MediaUploadSessionResponse.
//
//	@param session
//	@return *MediaUploadSessionResponse
func NewMediaUploadSessionResponse(session MediaUploadSession) *MediaUploadSessionResponse {
	return &MediaUploadSessionResponse{
		Id:        session.Id,
		Received:  session.Received,
		TotalSize: session.TotalSize,
	}
}

// MediaUploadIdUri binds the upload session id from the chunk/complete path.
type MediaUploadIdUri struct {
	Id string `path:"id" validate:"required"`
}
