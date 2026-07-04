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
	SenderId   uint64    `json:"senderId"`
	ReceiverId uint64    `json:"receiverId"`
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
func NewMediaMetadata(id string, senderId uint64, receiverId uint64, size int64) MediaMetadata {
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
	ReceiverId uint64                `form:"receiverId" binding:"required"`
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
