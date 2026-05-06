package database

import (
	"context"
	"errors"
	"jsfraz/whisper-server/utils"

	"github.com/valkey-io/valkey-go"
)

var wsValkeyErr *valkey.ValkeyError

// Push WebSocket access token to Valkey.
//
//	@param tokenId
//	@param token
//	@param ttl
//	@return error
func PushWsAccessToken(tokenId string, token string, ttl int) error {
	// Push with ws: prefix
	client := utils.GetSingleton().Valkey
	return client.Do(context.Background(), client.B().Set().Key("ws:"+tokenId).Value(token).ExSeconds(int64(ttl)).Build()).Error()
}

// Atomically get and delete WebSocket access token from Valkey.
// Uses GETDEL to prevent race conditions between EXISTS and GET.
//
//	@param tokenId
//	@return bool
//	@return string
//	@return error
func WsAccessTokenExists(tokenId string) (bool, string, error) {
	client := utils.GetSingleton().Valkey
	result, err := client.Do(context.Background(), client.B().Getdel().Key("ws:"+tokenId).Build()).AsBytes()
	if err != nil {
		if errors.As(err, &wsValkeyErr) {
			return false, "", nil // Key doesn't exist
		}
		return false, "", err
	}
	return true, string(result), nil
}
