package database

import (
	"fmt"
	"jsfraz/whisper-server/utils"
	"log"

	"github.com/valkey-io/valkey-go"
)

// Initializes Valkey client or panics.
func InitValkey() {
	client, err := valkey.NewClient(valkey.ClientOption{
		InitAddress: []string{
			fmt.Sprintf("%s:%d",
				utils.GetSingleton().Config.ValkeyHost,
				utils.GetSingleton().Config.ValkeyPort,
			),
		},
		Password: utils.GetSingleton().Config.ValkeyPassword,
	})
	if err != nil {
		log.Fatalln(err)
	}
	utils.GetSingleton().Valkey = client
}
