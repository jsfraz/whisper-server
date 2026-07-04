package database

import (
	"context"
	"errors"
	"fmt"
	"jsfraz/whisper-server/models"
	"jsfraz/whisper-server/utils"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/valkey-io/valkey-go"
)

// Prefix for media metadata keys in Valkey.
const mediaKeyPrefix = "media:"

// Returns the on-disk path for a media file by id.
//
//	@param id
//	@return string
func MediaFilePath(id string) string {
	// filepath.Base prevents path traversal via crafted ids.
	return filepath.Join(utils.GetSingleton().Config.MediaDir, filepath.Base(id))
}

// Store MediaMetadata in Valkey with a TTL (the file itself lives on disk).
//
//	@param meta
//	@param ttl
//	@return error
func StoreMediaMetadata(meta models.MediaMetadata, ttl int) error {
	m, err := meta.MarshalBinary()
	if err != nil {
		return err
	}
	client := utils.GetSingleton().Valkey
	return client.Do(context.Background(), client.B().Set().Key(mediaKeyPrefix+meta.Id).Value(string(m)).ExSeconds(int64(ttl)).Build()).Error()
}

// Get MediaMetadata by id from Valkey.
//
//	@param id
//	@return bool
//	@return *models.MediaMetadata
//	@return error
func GetMediaMetadata(id string) (bool, *models.MediaMetadata, error) {
	client := utils.GetSingleton().Valkey
	result, err := client.Do(context.Background(), client.B().Get().Key(mediaKeyPrefix+id).Build()).AsBytes()
	// Return error except if is Valkey error (key not found)
	if err != nil && !errors.As(err, &valkeyErr) {
		return false, nil, err
	}
	if result == nil {
		return false, nil, nil
	}
	meta, err := models.MediaMetadataFromJson(result)
	if err != nil {
		return false, nil, err
	}
	return true, meta, nil
}

// Delete media file from disk and its metadata key from Valkey.
//
//	@param id
//	@return error
func DeleteMedia(id string) error {
	// Remove file from disk (ignore missing file)
	err := os.Remove(MediaFilePath(id))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	// Remove metadata key from Valkey
	client := utils.GetSingleton().Valkey
	return client.Do(context.Background(), client.B().Del().Key(mediaKeyPrefix+id).Build()).Error()
}

// Delete all media where given user is either sender or receiver.
//
//	@param userId
//	@return error
func DeleteUserMedia(userId uint64) error {
	client := utils.GetSingleton().Valkey
	// Scan all media keys
	var cursor uint64 = 0
	for {
		result, err := client.Do(context.Background(), client.B().Scan().Cursor(cursor).Match(mediaKeyPrefix+"*").Count(100).Build()).AsScanEntry()
		if err != nil {
			return err
		}
		for _, key := range result.Elements {
			metaBytes, err := client.Do(context.Background(), client.B().Get().Key(key).Build()).AsBytes()
			if err != nil && !errors.As(err, &valkeyErr) {
				log.Println(err)
				continue
			}
			if metaBytes == nil {
				continue
			}
			meta, err := models.MediaMetadataFromJson(metaBytes)
			if err != nil {
				log.Println(err)
				continue
			}
			if meta.SenderId == userId || meta.ReceiverId == userId {
				if err := DeleteMedia(meta.Id); err != nil {
					log.Println(err)
				}
			}
		}
		cursor = result.Cursor
		if cursor == 0 {
			break
		}
	}
	return nil
}

// Subscribe for expired media keys and delete the corresponding files from disk.
// Requires Valkey to be started with keyspace expiry notifications enabled
// (e.g. --notify-keyspace-events Ex).
func SubscribeExpiredMedia() {
	c, cancel := utils.GetSingleton().Valkey.Dedicate()
	defer cancel()
	wait := c.SetPubSubHooks(valkey.PubSubHooks{
		OnMessage: func(m valkey.PubSubMessage) {
			// Message payload is the expired key name, e.g. "media:{id}"
			if !strings.HasPrefix(m.Message, mediaKeyPrefix) {
				return
			}
			id := strings.TrimPrefix(m.Message, mediaKeyPrefix)
			// Metadata key has already expired, just remove the file from disk
			err := os.Remove(MediaFilePath(id))
			if err != nil && !errors.Is(err, os.ErrNotExist) {
				log.Println(fmt.Errorf("failed to delete expired media file %s: %w", id, err))
			}
		},
	})
	// Database 0 is the default Valkey database
	c.Do(context.Background(), c.B().Subscribe().Channel("__keyevent@0__:expired").Build())
	<-wait
}
