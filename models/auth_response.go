package models

type AuthResponse struct {
	AccessToken  string `json:"accessToken" validate:"required" example:"JWT_ACCESS_TOKEN"`
	RefreshToken string `json:"refreshToken" validate:"required" example:"JWT_REFRESH_TOKEN"`
	// User         User   `json:"user" validate:"required"`
	/*
		EncryptedPrivateKey string `json:"encryptedPrivateKey" validate:"required" example:"ENCRYPTED_RSA_PRIVATE_KEY_PEM"`
		EncryptedMasterKey  string `json:"encryptedMasterKey" validate:"required" example:"ENCRYPTED_MASTER_KEY"`
	*/
}

// Returns new AuthResponse.
//
//	@param accessToken
//	@param refreshToken
//	@param user
//	@return *AuthResponse
func NewAuth(accessToken string, refreshToken string) *AuthResponse {
	a := new(AuthResponse)
	a.AccessToken = accessToken
	a.RefreshToken = refreshToken
	// a.User = user
	/*
		a.EncryptedPrivateKey = user.EncryptedPrivateKey
		a.EncryptedMasterKey = user.EncryptedMasterKey
	*/
	return a
}
