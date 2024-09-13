package database

import (
	"fmt"
	"jsfraz/whisper-server/utils"
	"log"

	"github.com/valkey-io/valkey-go"
)

// Initializes database or panics.
func InitValkey() {
	valkey, err := valkey.NewClient(valkey.ClientOption{
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

	utils.GetSingleton().Valkey = valkey
}
