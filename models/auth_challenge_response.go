package models

// AuthChallengeResponse carries the one-time value the client embeds in the
// RSA-signed login JWT.
type AuthChallengeResponse struct {
	Challenge string `json:"challenge" validate:"required"`
	// Validity of the challenge in seconds.
	ExpiresIn int `json:"expiresIn" validate:"required"`
}

// Return new AuthChallengeResponse.
//
//	@param challenge
//	@param expiresIn
//	@return *AuthChallengeResponse
func NewAuthChallengeResponse(challenge string, expiresIn int) *AuthChallengeResponse {
	return &AuthChallengeResponse{Challenge: challenge, ExpiresIn: expiresIn}
}
