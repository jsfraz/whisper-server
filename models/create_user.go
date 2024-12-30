package models

type CreateUser struct {
	Mail string `json:"mail" validate:"email,required" example:"user@example.com"`
}
