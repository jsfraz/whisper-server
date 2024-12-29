package models

type Auth struct {
	UserId      uint64 `json:"userId" validate:"required" example:"1"`
	Nonce       string `json:"nonce" validate:"required" example:"BASE64_NONCE"`
	SignedNonce string `json:"signedNonce" validate:"required" example:"BASE64_RSA_SIGNED_NONCE"`
}
