package database

import (
	"fmt"
	"jsfraz/whisper-server/models"
	"jsfraz/whisper-server/utils"
	"log"

	sqliteEncrypt "github.com/ShaoQ1ang/gorm-sqlite-cipher"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Initializes database or panics.
func InitSqlite() {
	dbpath := "data/whisper.sqlite"
	dbnameWithDSN := dbpath + fmt.Sprintf("?_pragma_key=%s&_pragma_cipher_page_size=4096", utils.GetSingleton().Config.SqlitePassword)
	sqlite, err := gorm.Open(sqliteEncrypt.Open(dbnameWithDSN), &gorm.Config{Logger: logger.Default.LogMode(GetGormLogLevel())})
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
