package database

import (
	"fmt"
	"jsfraz/whisper-server/utils"
	"log"

	"github.com/valkey-io/valkey-go"
)

// Initializes database or panics.
func InitValkey() {
	// Valkey for invites
	valkeyInvite, err := valkey.NewClient(valkey.ClientOption{
		InitAddress: []string{
			fmt.Sprintf("%s:%d",
				utils.GetSingleton().Config.ValkeyHost,
				utils.GetSingleton().Config.ValkeyPort,
			),
		},
		Password: utils.GetSingleton().Config.ValkeyPassword,
		SelectDB: 0},
	)
	if err != nil {
		log.Panicln(err)
	}

	// Valkey for WebSocket access tokens
	valkeyWs, err := valkey.NewClient(valkey.ClientOption{
		InitAddress: []string{
			fmt.Sprintf("%s:%d",
				utils.GetSingleton().Config.ValkeyHost,
				utils.GetSingleton().Config.ValkeyPort,
			),
		},
		Password: utils.GetSingleton().Config.ValkeyPassword,
		SelectDB: 1},
	)
	if err != nil {
		log.Panicln(err)
	}

	// Valkey for user messages
	valkeyMessage, err := valkey.NewClient(valkey.ClientOption{
		InitAddress: []string{
			fmt.Sprintf("%s:%d",
				utils.GetSingleton().Config.ValkeyHost,
				utils.GetSingleton().Config.ValkeyPort,
			),
		},
		Password: utils.GetSingleton().Config.ValkeyPassword,
		SelectDB: 2},
	)
	if err != nil {
		log.Panicln(err)
	}

	utils.GetSingleton().ValkeyInvite = valkeyInvite
	utils.GetSingleton().ValkeyWs = valkeyWs
	utils.GetSingleton().ValkeyMessage = valkeyMessage
}
