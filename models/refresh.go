package models

type Refresh struct {
	RefreshToken string `json:"refreshToken" validate:"required" example:"JWT_REFRESH_TOKEN"`
}
