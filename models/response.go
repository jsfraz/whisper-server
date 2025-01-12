package models

import "encoding/json"

type Response struct {
	Type    ResponseType `json:"type"`
	Payload interface{}  `json:"payload"`
}

// Response to client.
//
//	@param responseType
//	@param payload
//	@return Response
func NewResponse(responseType ResponseType, payload interface{}) Response {
	return Response{
		Type:    responseType,
		Payload: payload,
	}
}

// MarshalResponse marshals the response to a JSON binary message
//
//	@param response
//	@return []byte
func MarshalResponse(response Response) ([]byte, error) {
	json, err := json.Marshal(response)
	if err != nil {
		return nil, err
	}
	return json, nil
}
