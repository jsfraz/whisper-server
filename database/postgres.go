package database

import (
	"jsfraz/whisper-server/models"
	"jsfraz/whisper-server/utils"
	"log"
	"time"

	"github.com/lib/pq"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Initializes database or panics.
func InitPostgres() {
	postgres, err := gorm.Open(postgres.Open(utils.GetSingleton().GetPostgresConnStr()), &gorm.Config{Logger: logger.Default.LogMode(GetGormLogLevel())})
	if err != nil {
		log.Panicln(err)
	}
	// Schema migration
	err = postgres.AutoMigrate(
		&models.User{},
	)
	if err != nil {
		log.Panicln(err)
	}
	utils.GetSingleton().Postgres = *postgres
}

// Creates PostgreSQL triggers from SQL script paths. Panics on error.
//
//	@param paths
func CreatePostgresTriggers(paths ...string) {
	for _, p := range paths {
		// Read script
		sql, err := utils.ReadFile(p)
		if err != nil {
			log.Panicln(err)
		}
		// Register trigger
		err = utils.GetSingleton().Postgres.Exec(*sql).Error
		if err != nil {
			log.Panicln(err)
		}
	}
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

// Method for creating listener for specific triggers. Panics on error.
//
//	@param connStr
//	@param channel
//	@param callback
func PostgresTriggerListener(connStr string, channel string, callback func(string)) {
	// Create listener
	listener := pq.NewListener(connStr, 10*time.Second, time.Minute, func(event pq.ListenerEventType, err error) {
		if err != nil {
			log.Println(err.Error())
		}
	})
	err := listener.Listen(channel)
	if err != nil {
		log.Panicln(err)
	}
	// Listen
	for {
		select {
		case notification := <-listener.Notify:
			// Change detection
			if notification != nil {
				callback(notification.Extra)
			}
		case <-time.After(90 * time.Second):
			go listener.Ping()
		}
	}
}
