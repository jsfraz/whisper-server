package models

import "encoding/json"

type WsResponse struct {
	Type    WsResponseType `json:"type"`
	Payload interface{}    `json:"payload"`
}

// Response to client.
//
//	@param responseType
//	@param payload
//	@return Response
func NewWsResponse(responseType WsResponseType, payload interface{}) WsResponse {
	return WsResponse{
		Type:    responseType,
		Payload: payload,
	}
}

// MarshalWsResponse marshals the response to a JSON binary message
//
//	@param response
//	@return []byte
func MarshalWsResponse(response WsResponse) ([]byte, error) {
	json, err := json.Marshal(response)
	if err != nil {
		return nil, err
	}
	return json, nil
}
