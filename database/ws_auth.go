package database

import (
	"context"
	"jsfraz/whisper-server/utils"
)

// Push WebSocket access token to Redis.
//
//	@param tokenId
//	@param token
//	@param ttl
//	@return error
func PushWsAccessToken(tokenId string, token string, ttl int) error {
	// Push
	client := utils.GetSingleton().ValkeyWs
	return client.Do(context.Background(), client.B().Set().Key(tokenId).Value(token).ExSeconds(int64(ttl)).Build()).Error()
}

// Check if WebSocket access token exists in Redis.
//
//	@param tokenId
//	@return error
func WsAccessTokenExists(tokenId string) (bool, string, error) {
	client := utils.GetSingleton().ValkeyWs
	exists, err := client.Do(context.Background(), client.B().Exists().Key(tokenId).Build()).AsBool()
	if err != nil {
		return false, "", err
	}
	// If token exists, return it and delete it
	if exists {
		// Get token and delete it
		token, err := client.Do(context.Background(), client.B().Getdel().Key(tokenId).Build()).AsBytes()
		if err != nil {
			return false, "", err
		}
		return true, string(token), nil
	}
	return false, "", nil
}
