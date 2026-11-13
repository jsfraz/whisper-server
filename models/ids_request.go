package models

type IdsRequest struct {
	Ids []string `json:"ids" validate:"required,dive,uuid"`
}
