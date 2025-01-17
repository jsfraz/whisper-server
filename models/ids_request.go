package models

type IdsRequest struct {
	Ids []uint64 `json:"ids" validate:"required"`
}
