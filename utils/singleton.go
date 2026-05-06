package utils

// https://dev.to/kittipat1413/understanding-the-singleton-pattern-in-go-5h99

import (
	"sync"

	messaging "firebase.google.com/go/v4/messaging"
	"github.com/valkey-io/valkey-go"
	"gopkg.in/gomail.v2"
	"gorm.io/gorm"
)

var (
	instance *Singleton
	once     sync.Once
)

type Singleton struct {
	Config                 Config
	Sqlite                 *gorm.DB
	Valkey                 valkey.Client
	FirebaseMsg            *messaging.Client
	Hub                    *Hub
	RegisterAdminTemplate  string
	RegisterInviteTemplate string
	smtpDialer             *gomail.Dialer
}

// Gets Singleton instance
//
//	@return *Singleton
func GetSingleton() *Singleton {
	once.Do(func() {
		instance = new(Singleton)
	})
	return instance
}
