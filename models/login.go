package models

type Login struct {
	Username string `json:"username" validate:"required" example:"ex4ample"`
	Password string `json:"password" validate:"required" example:"str0ng_p455w0rd"`
}
