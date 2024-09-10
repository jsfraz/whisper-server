package utils

import (
	"fmt"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {

	// Gin mode
	GinMode string `envconfig:"GIN_MODE" default:"debug"` // Default debug

	// PostgreSQL
	PostgresUser     string `envconfig:"POSTGRES_USER" required:"true"`
	PostgresPassword string `envconfig:"POSTGRES_PASSWORD" required:"true"`
	PostgresHost     string `envconfig:"POSTGRES_HOST" required:"true"`
	PostgresPort     int    `envconfig:"POSTGRES_PORT" default:"5432"` // Default 5432
	PostgresDb       string `envconfig:"POSTGRES_DB" required:"true"`

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
}

// Loads config from environmental values.
//
//	@return *Config
//	@return error
func LoadConfig() (*Config, error) {
	var config Config
	err := envconfig.Process("", &config)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %v", err)
	}
	return &config, nil
}
