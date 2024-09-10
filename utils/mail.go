package utils

import (
	"gopkg.in/gomail.v2"
)

// Sends mail https://pkg.go.dev/gopkg.in/gomail.v2#example-package
//
//	@param mailData
//	@param to
//	@param username
//	@param text2
//	@return error
func SendMail(mailData MailData, to string, username string, text2 string) error {
	// Render template
	content, err := mailData.ToHtml(GetSingleton().MailTemlplate, username, text2)
	if err != nil {
		return err
	}
	// Create message
	m := gomail.NewMessage()
	m.SetHeader("From", GetSingleton().Config.SmtpUser)
	m.SetHeader("To", to)
	m.SetHeader("Subject", mailData.Subject)
	m.SetBody("text/html", *content)
	// Send mail
	d := gomail.NewDialer(GetSingleton().Config.SmtpHost, GetSingleton().Config.SmtpPort, GetSingleton().Config.SmtpUser, GetSingleton().Config.SmtpPassword)
	return d.DialAndSend(m)
}
