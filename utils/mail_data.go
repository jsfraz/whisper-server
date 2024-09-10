package utils

import (
	"encoding/json"

	"github.com/aymerick/raymond"
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
func (m *MailData) ToHtml(template string, username string, text2 string) (*string, error) {
	content, err := raymond.Render(template,
		map[string]string{
			"subject":  m.Subject,
			"username": username,
			"text1":    m.Text1,
			"text2":    text2,
			"footer":   m.Footer,
		})
	if err != nil {
		return nil, err
	}
	return &content, nil
}
