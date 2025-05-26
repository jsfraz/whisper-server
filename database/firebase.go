package database

import (
	"context"
	"fmt"
	"jsfraz/whisper-server/utils"
)

// Push user Firebase token to Valkey.
//
//	@param userId
//	@param token
//	@return error
func PushFirebaseUserToken(userId uint64, token string) error {
	client := utils.GetSingleton().ValkeyFirebase
	return client.Do(context.Background(), client.B().Set().Key(fmt.Sprintf("%d", userId)).Value(token).Build()).Error()
}
