package database

import (
	"jsfraz/whisper-server/models"
	"jsfraz/whisper-server/utils"
	"log"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Initializes database or panics.
func InitSqlite() {
	sqlite, err := gorm.Open(sqlite.Open("whisper.db"), &gorm.Config{Logger: logger.Default.LogMode(GetGormLogLevel())})
	if err != nil {
		log.Panicln(err)
	}
	// Schema migration
	err = sqlite.AutoMigrate(
		&models.User{},
	)
	if err != nil {
		log.Panicln(err)
	}
	utils.GetSingleton().Sqlite = *sqlite
}

// Gets Gorm log level.
//
//	@return logger.LogLevel
func GetGormLogLevel() logger.LogLevel {
	if utils.GetSingleton().Config.GinMode == "release" {
		return logger.Error
	}
	return logger.Info
}
