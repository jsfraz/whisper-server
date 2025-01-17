package models

type WsAuthResponse struct {
	AccessToken string `json:"accessToken" validate:"required" example:"JWT_ACCESS_TOKEN"`
}

// WsAuthResponse.
//
//	@param accessToken
//	@return *WsAuthResponse
func NewWsAuthResponse(accessToken string) *WsAuthResponse {
	w := new(WsAuthResponse)
	w.AccessToken = accessToken
	return w
}
