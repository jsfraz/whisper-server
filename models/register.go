package models

type Register struct {
	Username  string `json:"username" validate:"required,alphanum,min=2,max=32" example:"ex4ample"`
	Mail      string `json:"mail" validate:"required,email" example:"user@example.com"`
	PublicKey string `json:"publicKey" validate:"required" example:"RSA_PUBLIC_KEY_PEM"`
}
