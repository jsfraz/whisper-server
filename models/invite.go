package models

import "encoding/json"

type Invite struct {
	Url  string `json:"url"`
	Code string `json:"code"`
}

// Return new Invite
//
//	@param url
//	@param code
//	@return *Invite
func NewInvite(url string, code string) *Invite {
	i := new(Invite)
	i.Url = url
	i.Code = code
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
