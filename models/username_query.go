package models

type UsernameQuery struct {
	Username string `query:"username" validate:"required"`
}
