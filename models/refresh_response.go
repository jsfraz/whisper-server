package models

type RefreshResponse struct {
	AccessToken string `json:"accessToken" validate:"required" example:"JWT_ACCESS_TOKEN"`
	// The rotated refresh token: the previous one is spent.
	RefreshToken string `json:"refreshToken" validate:"required" example:"JWT_REFRESH_TOKEN"`
}

// Returns new RefreshResponse.
//
//	@param accessToken
//	@param refreshToken
//	@return *RefreshResponse
func NewRefreshResponse(accessToken string, refreshToken string) *RefreshResponse {
	r := new(RefreshResponse)
	r.AccessToken = accessToken
	r.RefreshToken = refreshToken
	return r
}
