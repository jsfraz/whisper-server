package models

type Verify struct {
	Code string `json:"code" validate:"required,len=32" example:"tQrpaCeDwBD71217HXBG9Y35rg1ECT78"`
}
