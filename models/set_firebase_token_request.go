package models

type SetFirebaseTokenRequest struct {
	Token string `json:"token" validate:"required"`
}
