package models

type AuthResponse struct {
	AccessToken  string `json:"accessToken" validate:"required" example:"JWT_ACCESS_TOKEN"`
	RefreshToken string `json:"refreshToken" validate:"required" example:"JWT_REFRESH_TOKEN"`
}

// AuthResponse.
//
//	@param accessToken
//	@param refreshToken
//	@return *AuthResponse
func NewAuth(accessToken string, refreshToken string) *AuthResponse {
	a := new(AuthResponse)
	a.AccessToken = accessToken
	a.RefreshToken = refreshToken
	return a
}
