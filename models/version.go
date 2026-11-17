package models

// VersionResponse describes the server build and the oldest client it accepts,
// so a client can tell the user what to do instead of failing anonymously.
type VersionResponse struct {
	Name             string `json:"name" validate:"required" example:"whisper-server"`
	Version          string `json:"version" validate:"required" example:"0.1.0"`
	ApiVersion       string `json:"apiVersion" validate:"required" example:"1"`
	MinClientVersion string `json:"minClientVersion" validate:"required" example:"0.1.0"`
}

// NewVersionResponse.
//
//	@param version
//	@param apiVersion
//	@param minClientVersion
//	@return *VersionResponse
func NewVersionResponse(version string, apiVersion string, minClientVersion string) *VersionResponse {
	return &VersionResponse{
		Name:             "whisper-server",
		Version:          version,
		ApiVersion:       apiVersion,
		MinClientVersion: minClientVersion,
	}
}
