package utils

import (
	"fmt"
	"time"

	"gopkg.in/gomail.v2"
)

// Returns a cached SMTP dialer instance from the singleton.
//
//	@return *gomail.Dialer
func GetSmtpDialer() *gomail.Dialer {
	s := GetSingleton()
	if s.smtpDialer == nil {
		s.smtpDialer = gomail.NewDialer(
			s.Config.SmtpHost,
			s.Config.SmtpPort,
			s.Config.SmtpUser,
			s.Config.SmtpPassword,
		)
	}
	return s.smtpDialer
}

// Sends mail https://pkg.go.dev/gopkg.in/gomail.v2#example-package
//
//	@param to
//	@param subject
//	@param content
//	@return error
func SendMail(to string, subject, content string) error {
	// Create message
	m := gomail.NewMessage()
	m.SetHeader("From", GetSingleton().Config.SmtpUser)
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", content)
	// Send mail
	return GetSmtpDialer().DialAndSend(m)
}

// Returns mail footer with time sent.
//
//	@return string
func GetMailFooter() string {
	return fmt.Sprintf("Whisper %s", time.Now().Format("2.1. 2006 15:04:05"))
}
