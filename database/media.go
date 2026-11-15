package database

import (
	"context"
	"errors"
	"jsfraz/whisper-server/models"
	"jsfraz/whisper-server/utils"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/valkey-io/valkey-go"
)

// Prefix for media metadata keys in Valkey.
const mediaKeyPrefix = "media:"

// Suffix of an upload that is still being written. The file is renamed to its
// final name only after the metadata is stored, so a sweep never sees a half-written upload.
const MediaUploadTempSuffix = ".part"

// Upload temp files older than this are considered abandoned (e.g. after a crash).
const mediaUploadTempMaxAge = time.Hour

// Returns the on-disk path for a media file by id.
//
//	@param id
//	@return string
func MediaFilePath(id string) (string, error) {
	if !isValidMediaID(id) {
		return "", errors.New("invalid media id")
	}
	// Valid UUID ids are single path components; Base is retained as defense-in-depth.
	return filepath.Join(utils.GetSingleton().Config.MediaDir, filepath.Base(id)), nil
}

// Validate media id format (UUID).
func isValidMediaID(id string) bool {
	_, err := uuid.Parse(id)
	return err == nil
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
	path, err := MediaFilePath(id)
	if err != nil {
		return err
	}
	path = filepath.Clean(path)
	mediaDir := filepath.Clean(utils.GetSingleton().Config.MediaDir)
	if path == mediaDir || !strings.HasPrefix(path, mediaDir+string(os.PathSeparator)) {
		return errors.New("invalid media path")
	}

	// Remove file from disk (ignore missing file)
	err = os.Remove(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	// Remove metadata key from Valkey
	client := utils.GetSingleton().Valkey
	return client.Do(context.Background(), client.B().Del().Key(mediaKeyPrefix+id).Build()).Error()
}

// mediaUserKeyPrefix keys the set of media ids a user participates in, so media
// cleanup is a set read instead of a scan of every media key.
const mediaUserKeyPrefix = "media:user:"

// IndexUserMedia records a media id under both participants. The set is given a
// TTL a little longer than the media itself, so stale entries disappear on their
// own.
//
//	@param senderId
//	@param receiverId
//	@param id
//	@param ttl
//	@return error
func IndexUserMedia(senderId string, receiverId string, id string, ttl int) error {
	client := utils.GetSingleton().Valkey
	setTtl := int64(ttl) + 3600
	keys := []string{mediaUserKeyPrefix + senderId}
	if receiverId != senderId {
		keys = append(keys, mediaUserKeyPrefix+receiverId)
	}
	for _, key := range keys {
		if err := client.Do(context.Background(), client.B().Sadd().Key(key).Member(id).Build()).Error(); err != nil {
			return err
		}
		if err := client.Do(context.Background(), client.B().Expire().Key(key).Seconds(setTtl).Build()).Error(); err != nil {
			return err
		}
	}
	return nil
}

// Delete all media where given user is either sender or receiver.
//
//	@param userId
//	@return error
func DeleteUserMedia(userId string) error {
	client := utils.GetSingleton().Valkey
	key := mediaUserKeyPrefix + userId
	ids, err := client.Do(context.Background(), client.B().Smembers().Key(key).Build()).AsStrSlice()
	if err != nil {
		if errors.As(err, &valkeyErr) {
			return nil
		}
		return err
	}
	for _, id := range ids {
		if !isValidMediaID(id) {
			continue
		}
		if err := DeleteMedia(id); err != nil {
			slog.Error("media cleanup failed", "media_id", id, "error", err)
		}
	}
	return client.Do(context.Background(), client.B().Del().Key(key).Build()).Error()
}

// Deletes media files whose metadata key no longer exists in Valkey and upload
// temp files abandoned by a crash. The pub/sub listener below only fires while
// the server is running and subscribed, so a missed expiry, a restart or a crash
// can leave a file on disk forever; this sweep reconciles that.
// Returns the number of removed files.
func SweepOrphanedMedia() (int, error) {
	dir := utils.GetSingleton().Config.MediaDir
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return 0, nil
		}
		return 0, err
	}
	removed := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		path := filepath.Join(dir, name)
		// Abandoned upload
		if strings.HasSuffix(name, MediaUploadTempSuffix) {
			info, err := entry.Info()
			if err == nil && time.Since(info.ModTime()) > mediaUploadTempMaxAge {
				if err = os.Remove(path); err == nil {
					removed++
				}
			}
			continue
		}
		// Leave anything that is not a media file alone
		if !isValidMediaID(name) {
			continue
		}
		exists, err := mediaMetadataExists(name)
		if err != nil {
			return removed, err
		}
		if !exists {
			if err = os.Remove(path); err == nil {
				removed++
			}
		}
	}
	return removed, nil
}

// RunMediaSweeper sweeps orphaned media now and then every interval.
func RunMediaSweeper(interval time.Duration) {
	for {
		removed, err := SweepOrphanedMedia()
		if err != nil {
			slog.Error("media sweep failed", "error", err)
		} else if removed > 0 {
			slog.Info("removed orphaned media files", "count", removed)
		}
		time.Sleep(interval)
	}
}

// Check whether the metadata key for a media id exists in Valkey.
func mediaMetadataExists(id string) (bool, error) {
	client := utils.GetSingleton().Valkey
	exists, err := client.Do(context.Background(), client.B().Exists().Key(mediaKeyPrefix+id).Build()).AsInt64()
	if err != nil {
		return false, err
	}
	return exists == 1, nil
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
			path, err := MediaFilePath(id)
			if err != nil {
				slog.Error("failed to resolve expired media file path", "media_id", id, "error", err)
				return
			}
			err = os.Remove(path)
			if err != nil && !errors.Is(err, os.ErrNotExist) {
				slog.Error("failed to delete expired media file", "media_id", id, "error", err)
			}
		},
	})
	// Database 0 is the default Valkey database
	c.Do(context.Background(), c.B().Subscribe().Channel("__keyevent@0__:expired").Build())
	<-wait
}
