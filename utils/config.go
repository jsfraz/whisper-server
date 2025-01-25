package utils

import (
	"fmt"

	"github.com/asaskevich/govalidator"
	"github.com/kelseyhightower/envconfig"
)

type Config struct {

	// Gin mode
	GinMode string `envconfig:"GIN_MODE" default:"debug"` // Default debug

	// Server URL
	ServerUrl string `envconfig:"SERVER_URL" required:"true"`

	// PostgreSQL
	PostgresUser     string `envconfig:"POSTGRES_USER" required:"true"`
	PostgresPassword string `envconfig:"POSTGRES_PASSWORD" required:"true"`
	PostgresHost     string `envconfig:"POSTGRES_HOST" required:"true"`
	PostgresPort     int    `envconfig:"POSTGRES_PORT" default:"5432"` // Default 5432
	PostgresDb       string `envconfig:"POSTGRES_DB" required:"true"`

	// Valkey
	ValkeyHost     string `envconfig:"VALKEY_HOST" required:"true"`
	ValkeyPort     int    `envconfig:"VALKEY_PORT" default:"6379"` // Default 6379
	ValkeyPassword string `envconfig:"VALKEY_PASSWORD" required:"true"`

	// Admin
	AdminMail string `envconfig:"ADMIN_MAIL" required:"true"`

	// Admin invite time to live
	AdminInviteTtl int `envconfig:"ADMIN_INVITE_TTL" required:"true"`

	// Invite time to live
	InviteTtl int `envconfig:"INVITE_TTL" required:"true"`

	// SMTP
	SmtpHost     string `envconfig:"SMTP_HOST" required:"true"`
	SmtpPort     int    `envconfig:"SMTP_PORT" default:"465"` // Default 465
	SmtpUser     string `envconfig:"SMTP_USER" required:"true"`
	SmtpPassword string `envconfig:"SMTP_PASSWORD" required:"true"`

	// Access token
	AccessTokenSecret   string `envconfig:"ACCESS_TOKEN_SECRET" required:"true"`
	AccessTokenLifespan int    `envconfig:"ACCESS_TOKEN_LIFESPAN" required:"true"`

	// Refresh token
	RefreshTokenSecret   string `envconfig:"REFRESH_TOKEN_SECRET" required:"true"`
	RefreshTokenLifespan int    `envconfig:"REFRESH_TOKEN_LIFESPAN" required:"true"`

	// WebSocket short life access token
	WsTokenSecret   string `envconfig:"WS_ACCESS_TOKEN_SECRET" required:"true"`
	WsTokenLifespan int    `envconfig:"WS_ACCESS_TOKEN_LIFESPAN" required:"true"`

	// Message TTL
	MessageTtl int `envconfig:"MESSAGE_TTL" required:"true"`
}

// Loads config from environmental values.
//
//	@return *Config
//	@return error
func LoadConfig() (*Config, error) {
	// Load config
	var config Config
	err := envconfig.Process("", &config)
	if err != nil {
		return nil, err
	}
	// Validate mail
	validMail := govalidator.IsEmail(config.AdminMail)
	if !validMail {
		return nil, fmt.Errorf("invalid admin mail: %s", config.AdminMail)
	}
	return &config, nil
}
