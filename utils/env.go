package utils

import (
	"fmt"
	"net/mail"
	"os"
	"regexp"
)

var numberPattern string = `^\d+$`

// Checks ENVs. Exits program if some ENV is not set.
func CheckEnvs() {
	// Mail
	ok := true
	_, err := mail.ParseAddress(os.Getenv("MAIL_USER"))
	if err != nil {
		ok = false
		fmt.Println("Invalid mail.")
	}
	if os.Getenv("MAIL_PASSWORD") == "" {
		ok = false
		fmt.Println("Invalid mail password.")
	}
	if os.Getenv("SMTP_SERVER") == "" {
		ok = false
		fmt.Println("Invalid SMTP address.")
	}
	matchPort, _ := regexp.MatchString(numberPattern, os.Getenv("SMTP_PORT"))
	if !matchPort {
		ok = false
		fmt.Println("Empty or invalid SMTP port.")
	}
	if os.Getenv("ACCESS_TOKEN_SECRET") == "" {
		ok = false
		fmt.Println("Invalid access token secret.")
	}
	if os.Getenv("REFRESH_TOKEN_SECRET") == "" {
		ok = false
		fmt.Println("Invalid refresh token secret.")
	}
	matchAccess, _ := regexp.MatchString(numberPattern, os.Getenv("ACCESS_TOKEN_LIFESPAN"))
	if !matchAccess {
		ok = false
		fmt.Println("Invalid access token lifespan.")
	}
	matchRefresh, _ := regexp.MatchString(numberPattern, os.Getenv("ACCESS_TOKEN_LIFESPAN"))
	if !matchRefresh {
		ok = false
		fmt.Println("Invalid access token lifespan.")
	}
	if !ok {
		fmt.Println("Check your environment variables. Shutting down...")
		os.Exit(1)
	}
}
