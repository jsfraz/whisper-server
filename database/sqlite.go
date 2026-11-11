package database

import (
	"embed"
	"fmt"
	"jsfraz/whisper-server/utils"
	"log"
	"net/url"

	sqliteEncrypt "github.com/ShaoQ1ang/gorm-sqlite-cipher"
	"github.com/pressly/goose/v3"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

//go:embed migrations/*.sql
var migrations embed.FS

// Initializes database or panics.
func InitSqlite() {
	initSqlite("data/whisper.sqlite")
}

// Initializes database at a custom path or panics. Used by E2E tests.
func InitSqliteAt(dbpath string) {
	initSqlite(dbpath)
}

func initSqlite(dbpath string) {
	dbnameWithDSN := dbpath + fmt.Sprintf("?_pragma_key=%s&_pragma_cipher_page_size=4096", url.QueryEscape(utils.GetSingleton().Config.SqlitePassword))
	sqlite, err := gorm.Open(sqliteEncrypt.Open(dbnameWithDSN), &gorm.Config{Logger: logger.Default.LogMode(GetGormLogLevel())})
	if err != nil {
		log.Fatalln(err)
	}
	if err := runMigrations(sqlite); err != nil {
		log.Fatalln(err)
	}
	utils.GetSingleton().Sqlite = sqlite
}

// Applies the embedded goose migrations.
func runMigrations(sqlite *gorm.DB) error {
	conn, err := sqlite.DB()
	if err != nil {
		return err
	}
	goose.SetBaseFS(migrations)
	if err := goose.SetDialect("sqlite3"); err != nil {
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
