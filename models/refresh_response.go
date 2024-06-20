package models

type RefreshResponse struct {
	AccessToken string `json:"accessToken" validate:"required" example:"JWT_ACCESS_TOKEN"`
}

// Returns new RefreshResponse.
//
//	@param accessToken
//	@return *RefreshResponse
func NewRefreshResponse(accessToken string) *RefreshResponse {
	r := new(RefreshResponse)
	r.AccessToken = accessToken
	return r
}
