package models

import (
	"encoding/json"
	"time"
)

type Invite struct {
	Mail       string    `json:"mail" validate:"required"`
	Admin      bool      `json:"admin" validate:"required"`
	ValidUntil time.Time `json:"validUntil" validate:"required"`
}

// Return new Invite.
//
//	@param mail
//	@param admin
//	@param validUntil
//	@return *InviteData
func NewInvite(mail string, admin bool, validUntil time.Time) *Invite {
	i := new(Invite)
	i.Mail = mail
	i.Admin = admin
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

// Return Invite from JSON bytes.
//
//	@param jsonBytes
//	@return *InviteData
//	@return error
func InviteFromJson(jsonBytes []byte) (*Invite, error) {
	var i Invite
	err := json.Unmarshal(jsonBytes, &i)
	if err != nil {
		return nil, err
	}
	return &i, nil
}
