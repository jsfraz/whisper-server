package utils

// https://dev.to/kittipat1413/understanding-the-singleton-pattern-in-go-5h99

import (
	messaging "firebase.google.com/go/v4/messaging"
	"github.com/valkey-io/valkey-go"
	"gorm.io/gorm"
)

var (
	instance *Singleton
)

type Singleton struct {
	Config         Config
	Sqlite         gorm.DB
	ValkeyInvite   valkey.Client
	ValkeyWs       valkey.Client
	ValkeyMessage  valkey.Client
	ValkeyDelUser  valkey.Client
	ValkeyFirebase valkey.Client
	FirebaseMsg    *messaging.Client
	Hub            *Hub
}

// Gets Singleton instance
//
//	@return *Singleton
func GetSingleton() *Singleton {
	if instance == nil {
		instance = new(Singleton)
	}
	return instance
}
