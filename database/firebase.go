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
func PushFirebaseUserToken(userId string, token string) error {
	client := utils.GetSingleton().Valkey
	return client.Do(context.Background(), client.B().Set().Key(fmt.Sprintf("fcm:%s", userId)).Value(token).Build()).Error()
}
