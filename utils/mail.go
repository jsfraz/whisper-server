package utils

import (
	"os"
	"strconv"

	"gopkg.in/gomail.v2"
)

// Sends mail https://pkg.go.dev/gopkg.in/gomail.v2#example-package

func SendMail(mailData MailData, to string, username string, text2 string) error {
	m := gomail.NewMessage()
	m.SetHeader("From", os.Getenv("MAIL_USER"))
	m.SetHeader("To", to)
	m.SetHeader("Subject", mailData.Subject)
	m.SetBody("text/html", mailData.ToHtml(GetSingleton().MailTemlplate, username, text2))

	port, _ := strconv.Atoi(os.Getenv("SMTP_PORT"))
	d := gomail.NewDialer(os.Getenv("SMTP_SERVER"), port, os.Getenv("MAIL_USER"), os.Getenv("MAIL_PASSWORD"))

	return d.DialAndSend(m)
}
