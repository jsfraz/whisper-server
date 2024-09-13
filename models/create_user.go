package models

type CreateUser struct {
	InviteCode string `json:"inviteCode" validate:"required,min=64,max=64" example:"HYSicju.Zg}Q~3c+>W|/'LZ<@Pel/L8hBq0sSmWQ0pj>%@x#C,(yI5:h0On^zkQ6"`
	Username   string `json:"username" validate:"required,alphanum,min=2,max=32" example:"ex4ample"`
	PublicKey  string `json:"publicKey" validate:"required" example:"RSA_PUBLIC_KEY_PEM"`
}
