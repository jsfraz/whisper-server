package database

import (
	"jsfraz/whisper-server/models"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Initializes database or panics.
//
//	@return *gorm.DB
func InitPostgres() *gorm.DB {
	connStr := "postgresql://" + os.Getenv("POSTGRES_USER") + ":" + os.Getenv("POSTGRES_PASSWORD") + "@" + os.Getenv("POSTGRES_SERVER") + ":" + os.Getenv("POSTGRES_PORT") + "/" + os.Getenv("POSTGRES_DB")
	postgres, err := gorm.Open(postgres.Open(connStr), &gorm.Config{Logger: logger.Default.LogMode(GetGormLogLevel())})
	if err != nil {
		panic(err)
	}
	// migrace schémat a tabulky
	err = postgres.AutoMigrate(&models.User{})
	if err != nil {
		panic(err)
	}
	return postgres
}

// Gets Gorm log level.
//
//	@return logger.LogLevel
func GetGormLogLevel() logger.LogLevel {
	if os.Getenv("GIN_MODE") == "release" {
		return logger.Error
	}
	return logger.Info
}
