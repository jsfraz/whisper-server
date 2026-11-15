package utils

import (
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"strings"

	"github.com/asaskevich/govalidator"
	"github.com/kelseyhightower/envconfig"
)

type Config struct {

	// Gin mode
	GinMode string `envconfig:"GIN_MODE" oneof:"debug release" default:"debug"` // Default debug

	// Server URL
	ServerUrl string `envconfig:"SERVER_URL" required:"true"`

	// SQLite
	SqlitePassword string `envconfig:"SQLITE_PASSWORD" required:"true"`

	// Valkey
	ValkeyHost     string `envconfig:"VALKEY_HOST" required:"true"`
	ValkeyPort     int    `envconfig:"VALKEY_PORT" default:"6379"` // Default 6379
	ValkeyPassword string `envconfig:"VALKEY_PASSWORD" required:"true"`

	// Admin
	AdminMail string `envconfig:"ADMIN_MAIL" required:"true"`

	// Admin invite time to live
	AdminInviteTtl int `envconfig:"ADMIN_INVITE_TTL" required:"false" default:"600"` // Default 600 (10min)

	// Invite time to live
	InviteTtl int `envconfig:"INVITE_TTL" required:"false" default:"900"` // Default 900 (15min)

	// SMTP
	SmtpHost     string `envconfig:"SMTP_HOST" required:"true"`
	SmtpPort     int    `envconfig:"SMTP_PORT" default:"465"` // Default 465
	SmtpUser     string `envconfig:"SMTP_USER" required:"true"`
	SmtpPassword string `envconfig:"SMTP_PASSWORD" required:"true"`

	// Access token
	AccessTokenSecret   string `envconfig:"ACCESS_TOKEN_SECRET" required:"true"`
	AccessTokenLifespan int    `envconfig:"ACCESS_TOKEN_LIFESPAN" default:"900"` // Default 900 (15min)

	// Refresh token
	RefreshTokenSecret   string `envconfig:"REFRESH_TOKEN_SECRET" required:"true"`
	RefreshTokenLifespan int    `envconfig:"REFRESH_TOKEN_LIFESPAN" default:"604800"` // Default 604800 (7 days)

	// WebSocket short life access token
	WsAccessTokenSecret   string `envconfig:"WS_ACCESS_TOKEN_SECRET" required:"true"`
	WsAccessTokenLifespan int    `envconfig:"WS_ACCESS_TOKEN_LIFESPAN" default:"10"` // Default 10 (10 seconds)

	// Message TTL
	MessageTtl int `envconfig:"MESSAGE_TTL" default:"2592000"` // Default 2592000 (30 days)

	// Media TTL (should be >= MessageTtl so offline recipients can still fetch referenced media)
	MediaTtl int `envconfig:"MEDIA_TTL" default:"2592000"` // Default 2592000 (30 days)

	// Maximum allowed upload size for a single media file in bytes
	MediaMaxUploadSize int64 `envconfig:"MEDIA_MAX_UPLOAD_SIZE" default:"26214400"` // Default 26214400 (25 MiB)

	// Directory where encrypted media files are stored
	MediaDir string `envconfig:"MEDIA_DIR" default:"data/media"` // Default data/media

	// Per client IP rate limit for the expensive endpoints (register, login,
	// refresh, upload). Burst is the bucket size, refill the rate afterwards.
	RateLimitBurst     int `envconfig:"RATE_LIMIT_BURST" default:"20"`
	RateLimitPerMinute int `envconfig:"RATE_LIMIT_PER_MINUTE" default:"60"`

	// Comma separated proxies whose X-Forwarded-For header is trusted when the
	// client IP is resolved. Only private networks by default: the reverse proxy
	// runs inside them, and a public client can never inject its own address.
	TrustedProxies string `envconfig:"TRUSTED_PROXIES" default:"127.0.0.1,::1,10.0.0.0/8,172.16.0.0/12,192.168.0.0/16"`

	// Stops one account from exhausting shared storage.
	MediaMaxPerUser  int `envconfig:"MEDIA_MAX_PER_USER" default:"200"`
	MessageMaxQueued int `envconfig:"MESSAGE_MAX_QUEUED" default:"500"`

	// WebSocket messages accepted from one connection per second.
	WsMessageRate int `envconfig:"WS_MESSAGE_RATE" default:"20"`
}

// TrustedProxyList returns the configured proxies as a slice, ready for
// gin.Engine.SetTrustedProxies.
//
//	@return []string
func (c Config) TrustedProxyList() []string {
	proxies := []string{}
	for _, proxy := range strings.Split(c.TrustedProxies, ",") {
		proxy = strings.TrimSpace(proxy)
		if proxy != "" {
			proxies = append(proxies, proxy)
		}
	}
	return proxies
}

// Loads config from environmental values.
//
//	@return *Config
//	@return error
func LoadConfig() (*Config, error) {
	// Ensure data directory exists
	_, err := os.Stat("data/")
	if errors.Is(err, fs.ErrNotExist) {
		err = os.Mkdir("data/", os.ModePerm)
		if err != nil {
			log.Fatalln(err)
		}
	}

	// Firebase credentials
	_, err = os.Stat("data/firebase.json")
	if errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("firebase credentials file 'firebase.json' does not exist")
	}
	// Set environment variable for Firebase
	err = os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "data/firebase.json")
	if err != nil {
		return nil, fmt.Errorf("failed to set GOOGLE_APPLICATION_CREDENTIALS environment variable: %v", err)
	}

	// Load config
	var config Config
	err = envconfig.Process("", &config)
	if err != nil {
		return nil, err
	}

	// Ensure media directory exists
	err = os.MkdirAll(config.MediaDir, os.ModePerm)
	if err != nil {
		log.Fatalln(err)
	}

	// Validate mail
	validMail := govalidator.IsEmail(config.AdminMail)
	if !validMail {
		return nil, fmt.Errorf("invalid admin mail: %s", config.AdminMail)
	}

	return &config, nil
}
