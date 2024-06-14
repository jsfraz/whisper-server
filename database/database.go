package database

import (
	"jsfraz/whisper-server/models"
	"log"
	"os"
	"time"

	"github.com/lib/pq"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Initializes database or panics.
//
//	@return *gorm.DB
func InitPostgres(connStr string) *gorm.DB {
	postgres, err := gorm.Open(postgres.Open(connStr), &gorm.Config{Logger: logger.Default.LogMode(GetGormLogLevel())})
	if err != nil {
		log.Panicln(err)
	}
	// migrace schémat a tabulky
	err = postgres.AutoMigrate(&models.User{})
	if err != nil {
		log.Panicln(err)
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

// Method for creating listener for specific triggers
//
//	@param connStr
//	@param channel
//	@param callback
func TriggerListener(connStr string, channel string, callback func(string)) {
	// Create listener
	listener := pq.NewListener(connStr, 10*time.Second, time.Minute, func(event pq.ListenerEventType, err error) {
		if err != nil {
			log.Println(err.Error())
		}
	})
	err := listener.Listen(channel)
	if err != nil {
		log.Fatal(err)
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
