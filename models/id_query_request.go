package models

type IdQueryRequest struct {
	Id uint64 `query:"id" validate:"required"`
}
