package models

import "github.com/google/uuid"

type User struct {
	Id        string `json:"id" validate:"required" gorm:"primarykey" example:"2f8c1c1e-6c55-4d8f-9d3f-6f6b8a0c2d11"`
	Username  string `json:"username" validate:"required" example:"ex4ample"`
	PublicKey string `json:"publicKey" validate:"required" example:"RSA_PUBLIC_KEY_PEM"`
	Admin     bool   `json:"admin" validate:"required" example:"false"`
}

// Return new user.
//
//	@param username
//	@param mail
//	@param publicKey
//	@param admin
//	@return *User
func NewUser(username string, publicKey string, admin bool) *User {
	u := new(User)
	u.Id = uuid.NewString()
	u.Username = username
	u.PublicKey = publicKey
	u.Admin = admin
	return u
}
