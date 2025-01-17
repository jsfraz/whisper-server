package models

type Register struct {
	InviteCode string `json:"inviteCode" validate:"required,min=64,max=64" example:"INVITE_CODE"`
	Username   string `json:"username" validate:"required,alphanum,min=2,max=32" example:"ex4ample"`
	PublicKey  string `json:"publicKey" validate:"required" example:"RSA_PUBLIC_KEY_PEM"`
}
