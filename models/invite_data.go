package models

import "encoding/json"

type InviteData struct {
	Mail  string `json:"mail"`
	Admin bool   `json:"admin"`
}

// Return new InviteData.
//
//	@param mail
//	@param admin
//	@return *InviteData
func NewInviteData(mail string, admin bool) *InviteData {
	i := new(InviteData)
	i.Mail = mail
	i.Admin = admin
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

// Return InviteData from JSON bytes.
//
//	@param jsonBytes
//	@return *InviteData
//	@return error
func InviteDataFromJson(jsonBytes []byte) (*InviteData, error) {
	var i InviteData
	err := json.Unmarshal(jsonBytes, &i)
	if err != nil {
		return nil, err
	}
	return &i, nil
}
