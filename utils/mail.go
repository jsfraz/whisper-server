package utils

import (
	"fmt"
	"time"

	"gopkg.in/gomail.v2"
)

// Sends mail https://pkg.go.dev/gopkg.in/gomail.v2#example-package
//
//	@param mailData
//	@param to
//	@param username
//	@param text2
//	@return error
func SendMail(to string, subject, content string) error {
	// Create message
	m := gomail.NewMessage()
	m.SetHeader("From", GetSingleton().Config.SmtpUser)
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", content)
	// Send mail
	d := gomail.NewDialer(
		GetSingleton().Config.SmtpHost,
		GetSingleton().Config.SmtpPort,
		GetSingleton().Config.SmtpUser,
		GetSingleton().Config.SmtpPassword,
	)
	return d.DialAndSend(m)
}

func GetMailFooter() string {
	return fmt.Sprintf("Whisper %s", time.Now().Format("2.1. 2006 15:04"))
}
