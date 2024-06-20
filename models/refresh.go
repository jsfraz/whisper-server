package models

type Refresh struct {
	RefreshToken string `query:"refreshToken" validate:"required" example:"JWT_REFRESH_TOKEN"`
}
