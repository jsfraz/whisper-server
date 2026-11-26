package database

import (
	"context"
	"errors"
	"jsfraz/shept-server/utils"

	"github.com/valkey-io/valkey-go"
)

// challengeKeyPrefix keys the one-time login challenges.
const challengeKeyPrefix = "challenge:"

// ChallengeTtl is how long a login challenge stays valid. The client fetches it
// right before signing, so it only has to cover the round trip.
const ChallengeTtl = 30

var challengeErr *valkey.ValkeyError

// PushChallenge stores a one-time login challenge.
//
//	@param code
//	@return error
func PushChallenge(code string) error {
	client := utils.GetSingleton().Valkey
	return client.Do(context.Background(), client.B().Set().Key(challengeKeyPrefix+code).Value("1").ExSeconds(ChallengeTtl).Build()).Error()
}

// ConsumeChallenge atomically takes (and deletes) a login challenge. A JWT
// carrying it can be used exactly once, which is what makes the login token
// replay-proof and a revoked session actually revoked.
//
//	@param code
//	@return bool
//	@return error
func ConsumeChallenge(code string) (bool, error) {
	client := utils.GetSingleton().Valkey
	_, err := client.Do(context.Background(), client.B().Getdel().Key(challengeKeyPrefix+code).Build()).AsBytes()
	if err != nil {
		if errors.As(err, &challengeErr) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
