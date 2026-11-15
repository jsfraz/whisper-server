package database

import (
	"context"
	"errors"
	"jsfraz/whisper-server/utils"

	"github.com/valkey-io/valkey-go"
)

// refreshSessionPrefix keys the set of live refresh token ids of one user.
const refreshSessionPrefix = "refresh:"

var refreshValkeyErr *valkey.ValkeyError

// StoreRefreshSession records a refresh token id, so it can be rotated and
// revoked. The set expires with the longest possible token lifetime.
//
//	@param userId
//	@param tokenId
//	@param ttl
//	@return error
func StoreRefreshSession(userId string, tokenId string, ttl int) error {
	client := utils.GetSingleton().Valkey
	builder := client.B().Sadd().Key(refreshSessionPrefix + userId).Member(tokenId).Build()
	if err := client.Do(context.Background(), builder).Error(); err != nil {
		return err
	}
	// Keep the set alive as long as the token it holds could be.
	return client.Do(context.Background(), client.B().Expire().Key(refreshSessionPrefix+userId).Seconds(int64(ttl)).Build()).Error()
}

// ConsumeRefreshSession reports whether the token id is live and removes it, so
// a refresh token can be used exactly once.
//
//	@param userId
//	@param tokenId
//	@return bool
//	@return error
func ConsumeRefreshSession(userId string, tokenId string) (bool, error) {
	client := utils.GetSingleton().Valkey
	removed, err := client.Do(context.Background(), client.B().Srem().Key(refreshSessionPrefix+userId).Member(tokenId).Build()).AsInt64()
	if err != nil {
		return false, err
	}
	return removed == 1, nil
}

// SessionExists reports whether a refresh token id is currently live.
//
//	@param userId
//	@param tokenId
//	@return bool
//	@return error
func SessionExists(userId string, tokenId string) (bool, error) {
	client := utils.GetSingleton().Valkey
	exists, err := client.Do(context.Background(), client.B().Sismember().Key(refreshSessionPrefix+userId).Member(tokenId).Build()).AsBool()
	if err != nil {
		return false, err
	}
	return exists, nil
}

// DeleteRefreshSessions revokes every refresh token of a user (logout everywhere
// and account deletion).
//
//	@param userId
//	@return error
func DeleteRefreshSessions(userId string) error {
	client := utils.GetSingleton().Valkey
	err := client.Do(context.Background(), client.B().Del().Key(refreshSessionPrefix+userId).Build()).Error()
	if err != nil && !errors.As(err, &refreshValkeyErr) {
		return err
	}
	return nil
}
