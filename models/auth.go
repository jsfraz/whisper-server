package models

type Auth struct {
	Token string `json:"token" validate:"required" example:"JWT_TOKEN"`
}
