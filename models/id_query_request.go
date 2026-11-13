package models

type IdQueryRequest struct {
	Id string `query:"id" validate:"required,uuid"`
}
