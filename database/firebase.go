package database

import (
	"context"
	"fmt"
	"jsfraz/shept-server/utils"
	"strings"
)

// Push user Firebase token to Valkey, per device: a login notification goes to
// the other devices of the account, not to the one that just signed in.
//
//	@param userId
//	@param deviceId
//	@param token
//	@return error
func PushFirebaseUserToken(userId string, deviceId string, token string) error {
	client := utils.GetSingleton().Valkey
	key := fmt.Sprintf("fcm:%s:%s", userId, deviceId)
	return client.Do(context.Background(), client.B().Set().Key(key).Value(token).Build()).Error()
}

// FirebaseTokensOf returns the push targets of every device of an account,
// keyed by device id.
//
//	@param userId
//	@return map[string]string
//	@return error
func FirebaseTokensOf(userId string) (map[string]string, error) {
	client := utils.GetSingleton().Valkey
	tokens := map[string]string{}
	pattern := fmt.Sprintf("fcm:%s:*", userId)
	iter := client.B().Scan().Cursor(0).Match(pattern).Count(100).Build()
	keys := []string{}
	for {
		result, err := client.Do(context.Background(), iter).AsScanEntry()
		if err != nil {
			return nil, err
		}
		keys = append(keys, result.Elements...)
		if result.Cursor == 0 {
			break
		}
		iter = client.B().Scan().Cursor(result.Cursor).Match(pattern).Count(100).Build()
	}
	if len(keys) == 0 {
		return tokens, nil
	}
	values, err := client.Do(context.Background(), client.B().Mget().Key(keys...).Build()).AsStrSlice()
	if err != nil {
		return nil, err
	}
	for i, key := range keys {
		if i >= len(values) || values[i] == "" {
			continue
		}
		tokens[strings.TrimPrefix(key, fmt.Sprintf("fcm:%s:", userId))] = values[i]
	}
	return tokens, nil
}
