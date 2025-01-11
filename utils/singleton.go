package utils

// https://dev.to/kittipat1413/understanding-the-singleton-pattern-in-go-5h99

import (
	"fmt"

	"github.com/valkey-io/valkey-go"
	"gorm.io/gorm"
)

var (
	instance *Singleton
)

type Singleton struct {
	Config   Config
	Postgres gorm.DB
	Valkey   valkey.Client
	Hub      *Hub
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

// Return PostgreSQL connection string
//
//	@receiver s
//	@return string
func (s Singleton) GetPostgresConnStr() string {
	return fmt.Sprintf("postgresql://%s:%s@%s:%d/%s?sslmode=disable", // TODO remove sslMode???
		s.Config.PostgresUser,
		s.Config.PostgresPassword,
		s.Config.PostgresHost,
		s.Config.PostgresPort,
		s.Config.PostgresDb)
}
