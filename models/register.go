package models

type Register struct {
	Username string `json:"username" validate:"required,alphanum,min=2,max=32" example:"ex4ample"`
	Mail     string `json:"mail" validate:"required,email" example:"user@example.com"`
	Password string `json:"password" validate:"required,min=8,max=64" example:"str0ng_p455w0rd"`
}
