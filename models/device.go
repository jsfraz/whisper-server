package models

import "time"

// A device is one installation of the app signed into an account. Every token
// carries its id, so revoking the device locks it out immediately.
type Device struct {
	Id         string     `json:"id" validate:"required" gorm:"primarykey"`
	UserId     string     `json:"-" validate:"required" gorm:"index"`
	Label      string     `json:"label" validate:"required" example:"Pixel 8"`
	Platform   string     `json:"platform" validate:"required" example:"android"`
	CreatedAt  time.Time  `json:"createdAt" validate:"required"`
	LastSeenAt time.Time  `json:"lastSeenAt" validate:"required"`
	RevokedAt  *time.Time `json:"revokedAt"`
	// Set by the list handler for the device that made the request.
	Current bool `json:"current" gorm:"-"`
}

// DevicesResponse is the device list of the authenticated account.
type DevicesResponse struct {
	Devices []Device `json:"devices" validate:"required"`
}

// RetentionRequest carries the per-user archive retention choice.
type RetentionRequest struct {
	Days int `json:"days" validate:"required" example:"30"`
}

// RetentionResponse is the current archive retention choice.
type RetentionResponse struct {
	Days int `json:"days" validate:"required" example:"30"`
}

// NewRetentionResponse.
//
//	@param days
//	@return *RetentionResponse
func NewRetentionResponse(days int) *RetentionResponse {
	return &RetentionResponse{Days: days}
}
