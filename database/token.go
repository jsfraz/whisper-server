package database

import (
	"context"
	"errors"
	"fmt"
	"jsfraz/shept-server/utils"

	"github.com/valkey-io/valkey-go"
)

// sessionPrefix keys the set of live refresh token ids of one device
// (sessions:<userId>:<deviceId>), so revoking a device is one DEL.
const sessionPrefix = "sessions:"

var refreshValkeyErr *valkey.ValkeyError

func sessionKey(userId string, deviceId string) string {
	return fmt.Sprintf("%s%s:%s", sessionPrefix, userId, deviceId)
}

// StoreRefreshSession records a refresh token id for one device. The set
// expires with the longest possible token lifetime.
//
//	@param userId
//	@param deviceId
//	@param tokenId
//	@param ttl
//	@return error
func StoreRefreshSession(userId string, deviceId string, tokenId string, ttl int) error {
	client := utils.GetSingleton().Valkey
	key := sessionKey(userId, deviceId)
	builder := client.B().Sadd().Key(key).Member(tokenId).Build()
	if err := client.Do(context.Background(), builder).Error(); err != nil {
		return err
	}
	// Keep the set alive as long as the token it holds could be.
	return client.Do(context.Background(), client.B().Expire().Key(key).Seconds(int64(ttl)).Build()).Error()
}

// ConsumeRefreshSession reports whether the token id is live and removes it, so
// a refresh token can be used exactly once.
//
//	@param userId
//	@param deviceId
//	@param tokenId
//	@return bool
//	@return error
func ConsumeRefreshSession(userId string, deviceId string, tokenId string) (bool, error) {
	client := utils.GetSingleton().Valkey
	removed, err := client.Do(context.Background(), client.B().Srem().Key(sessionKey(userId, deviceId)).Member(tokenId).Build()).AsInt64()
	if err != nil {
		return false, err
	}
	return removed == 1, nil
}

// DeleteRefreshSessions revokes every refresh token of a user (logout everywhere
// and account deletion).
//
//	@param userId
//	@return error
func DeleteRefreshSessions(userId string) error {
	return deleteSessionKeys(sessionPrefix + userId + ":*")
}

// DeleteDeviceRefreshSessions revokes every refresh token of one device, so a
// revoked device cannot renew its access token.
//
//	@param userId
//	@param deviceId
//	@return error
func DeleteDeviceRefreshSessions(userId string, deviceId string) error {
	client := utils.GetSingleton().Valkey
	err := client.Do(context.Background(), client.B().Del().Key(sessionKey(userId, deviceId)).Build()).Error()
	if err != nil && !errors.As(err, &refreshValkeyErr) {
		return err
	}
	return nil
}

// deleteSessionKeys deletes every key matching a pattern.
func deleteSessionKeys(pattern string) error {
	client := utils.GetSingleton().Valkey
	keys := []string{}
	iter := client.B().Scan().Cursor(0).Match(pattern).Count(100).Build()
	for {
		result, err := client.Do(context.Background(), iter).AsScanEntry()
		if err != nil {
			return err
		}
		keys = append(keys, result.Elements...)
		if result.Cursor == 0 {
			break
		}
		iter = client.B().Scan().Cursor(result.Cursor).Match(pattern).Count(100).Build()
	}
	if len(keys) == 0 {
		return nil
	}
	err := client.Do(context.Background(), client.B().Del().Key(keys...).Build()).Error()
	if err != nil && !errors.As(err, &refreshValkeyErr) {
		return err
	}
	return nil
}
