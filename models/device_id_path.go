package models

// DeviceIdPath addresses one device in the device endpoints.
type DeviceIdPath struct {
	Id string `path:"id" validate:"required,uuid"`
}
