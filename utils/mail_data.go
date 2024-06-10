package utils

import (
	"encoding/json"
	"strings"
)

type MailData struct {
	Subject string `json:"subject"`
	Text1   string `json:"text1"`
	Footer  string `json:"footer"`
}

// New Mail from mail text JSON.
//
//	@param jsonStr
//	@return *MailData
func NewMailData(jsonStr string) *MailData {
	var m MailData
	err := json.Unmarshal([]byte(jsonStr), &m)
	if err != nil {
		panic(err)
	}
	return &m
}

// Creates a HTML mail string.
//
//	@receiver m
//	@param template
//	@param username
//	@param text2
//	@return string
func (m MailData) ToHtml(template string, username string, text2 string) string {
	s := template
	s = strings.Replace(s, "@SUBJECT", m.Subject, 1)
	s = strings.Replace(s, "@USERNAME", username, 1)
	s = strings.Replace(s, "@TEXT1", m.Text1, 1)
	s = strings.Replace(s, "@TEXT2", text2, 1)
	s = strings.Replace(s, "@FOOTER", m.Footer, 1)
	return s
}
