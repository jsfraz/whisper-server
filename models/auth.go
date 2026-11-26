package models

type Auth struct {
	Token string `json:"token" validate:"required" example:"JWT_TOKEN"`
	// DeviceId is generated once per installation and stays stable across
	// logins, so a revocation can target it. The client also signs it as the
	// "did" claim of the login JWT.
	DeviceId string `json:"deviceId" validate:"required,uuid"`
	// Human readable device name shown in the device list.
	Label string `json:"label" validate:"required,min=1,max=64" example:"Pixel 8"`
	// Platform of the installation.
	Platform string `json:"platform" validate:"required,min=1,max=32" example:"android"`
}

type AuthResponse struct {
	AccessToken  string `json:"accessToken" validate:"required" example:"JWT_ACCESS_TOKEN"`
	RefreshToken string `json:"refreshToken" validate:"required" example:"JWT_REFRESH_TOKEN"`
	// ServerVersion lets the client check it is talking to a server it knows,
	// without a second request.
	ServerVersion string `json:"serverVersion" validate:"required" example:"0.1.0"`
}

// AuthResponse.
//
//	@param accessToken
//	@param refreshToken
//	@param serverVersion
//	@return *AuthResponse
func NewAuthResponse(accessToken string, refreshToken string, serverVersion string) *AuthResponse {
	a := new(AuthResponse)
	a.AccessToken = accessToken
	a.RefreshToken = refreshToken
	a.ServerVersion = serverVersion
	return a
}
