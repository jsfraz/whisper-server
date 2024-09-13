package models

import (
	"encoding/json"
	"time"
)

type Invite struct {
	Url        string    `json:"url"`
	Code       string    `json:"code"`
	ValidUntil time.Time `json:"validUntil"`
}

// Return new Invite
//
//	@param url
//	@param code
//	@param validUntil
//	@return *Invite
func NewInvite(url string, code string, validUntil time.Time) *Invite {
	i := new(Invite)
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
func (i Invite) MarshalBinary() ([]byte, error) {
	return json.Marshal(i)
}
