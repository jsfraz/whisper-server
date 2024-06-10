package utils

import (
	"fmt"
	"net/mail"
	"os"
	"regexp"
)

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
	matchPort, _ := regexp.MatchString(`^\d+$`, os.Getenv("SMTP_PORT"))
	if !matchPort {
		ok = false
		fmt.Println("Empty or invalid SMTP port.")
	}
	if !ok {
		fmt.Println("Check your environment variables. Shutting down...")
		os.Exit(1)
	}
}
