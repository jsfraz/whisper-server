package models

import (
	"encoding/json"
	"time"
)

type InviteData struct {
	Url        string    `json:"url" validate:"required"`
	Code       string    `json:"code" validate:"required"`
	ValidUntil time.Time `json:"validUntil" validate:"required"`
}

// Return new InviteData
//
//	@param url
//	@param code
//	@param validUntil
//	@return *Invite
func NewInviteData(url string, code string, validUntil time.Time) *InviteData {
	i := new(InviteData)
	i.Url = url
	i.Code = code
	i.ValidUntil = validUntil
	return i
}

// Return JSON bytes.
//
//	@receiver i
//	@return []byte
//	@return error
func (i InviteData) MarshalBinary() ([]byte, error) {
	return json.Marshal(i)
}
