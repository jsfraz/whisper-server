package models

type User struct {
	Id        uint64 `json:"id" validate:"required" gorm:"primarykey" example:"1"`
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
	u.Username = username
	u.PublicKey = publicKey
	u.Admin = admin
	return u
}
