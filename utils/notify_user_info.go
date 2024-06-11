package utils

type NotifyUserInfo struct {
	Username         string `json:"username"`
	Mail             string `json:"mail"`
	VerificationCode string `json:"verification_code"`
}
