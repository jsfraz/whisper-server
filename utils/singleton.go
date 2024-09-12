package utils

// https://dev.to/kittipat1413/understanding-the-singleton-pattern-in-go-5h99

import (
	"github.com/valkey-io/valkey-go"
	"gorm.io/gorm"
)

var (
	instance *Singleton
)

type Singleton struct {
	Config     Config
	PostgresDb gorm.DB
	Valkey     valkey.Client
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
