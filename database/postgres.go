package database

import (
	"embed"
	"jsfraz/whisper-server/utils"
	"log"

	"github.com/pressly/goose/v3"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

//go:embed migrations/*.sql
var migrations embed.FS

// Initializes the database or panics.
func InitDb() {
	initDb(utils.GetSingleton().Config.DatabaseUrl)
}

// Initializes the database at a custom DSN or panics. Used by E2E tests.
func InitDbAt(dsn string) {
	initDb(dsn)
}

func initDb(dsn string) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(GetGormLogLevel())})
	if err != nil {
		log.Fatalln(err)
	}
	if err := runMigrations(db); err != nil {
		log.Fatalln(err)
	}
	utils.GetSingleton().Db = db
}

// Applies the embedded goose migrations.
func runMigrations(db *gorm.DB) error {
	conn, err := db.DB()
	if err != nil {
		return err
	}
	goose.SetBaseFS(migrations)
	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}
	return goose.Up(conn, "migrations")
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
