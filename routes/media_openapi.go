package routes

import (
	"github.com/wI2L/fizz/openapi"
)

// errorSchema is the JSON body returned by handlers on error.
func errorSchema() *openapi.SchemaOrRef {
	return &openapi.SchemaOrRef{Schema: &openapi.Schema{
		Type:                 "object",
		AdditionalProperties: &openapi.SchemaOrRef{Schema: &openapi.Schema{}},
	}}
}

// errorResponse mirrors the responses produced by utils.CreateOperationOption.
func errorResponse(description string) *openapi.ResponseOrRef {
	return &openapi.ResponseOrRef{Response: &openapi.Response{
		Description: description,
		Content: map[string]*openapi.MediaTypeOrRef{
			"application/json": {MediaType: &openapi.MediaType{
				Schema:  errorSchema(),
				Example: map[string]interface{}{"error": "..."},
			}},
		},
	}}
}

// addMediaOperations documents the media endpoints in the OpenAPI document.
//
// Upload and download are raw Gin handlers (multipart body, streamed ciphertext),
// so fizz cannot derive their specification from a tonic handler.
//
// shortcut: media operations are hand-written OpenAPI; derive them from the
// handlers once fizz supports multipart/form-data and binary responses.
func addMediaOperations(gen *openapi.Generator) {
	doc := gen.API()
	if doc.Components.Schemas == nil {
		doc.Components.Schemas = map[string]*openapi.SchemaOrRef{}
	}
	doc.Components.Schemas["ModelsMediaUploadResponse"] = &openapi.SchemaOrRef{Schema: &openapi.Schema{
		Type: "object",
		Properties: map[string]*openapi.SchemaOrRef{
			"id": {Schema: &openapi.Schema{Type: "string"}},
		},
		Required: []string{"id"},
	}}

	security := []*openapi.SecurityRequirement{{"bearerAuth": []string{}}}

	upload := &openapi.Operation{
		Tags:        []string{"Media"},
		Summary:     "Upload encrypted media file",
		Description: "Uploads an end-to-end encrypted media file bound to the receiver. The server stores only the ciphertext. Returns the media id.",
		ID:          "UploadMedia",
		Security:    security,
		RequestBody: &openapi.RequestBody{
			Required: true,
			Content: map[string]*openapi.MediaType{
				"multipart/form-data": {Schema: &openapi.SchemaOrRef{Schema: &openapi.Schema{
					Type: "object",
					Properties: map[string]*openapi.SchemaOrRef{
						"file": {Schema: &openapi.Schema{
							Type:        "string",
							Format:      "binary",
							Description: "The encrypted media blob (nonce || mac || ciphertext). Its field name must be file.",
						}},
						"receiverId": {Schema: &openapi.Schema{
							Type:        "integer",
							Format:      "int64",
							Description: "Id of the user the media is sent to.",
						}},
					},
					Required: []string{"file", "receiverId"},
				}}},
			},
		},
		Responses: openapi.Responses{
			"200": {Response: &openapi.Response{
				Description: "OK",
				Content: map[string]*openapi.MediaTypeOrRef{
					"application/json": {MediaType: &openapi.MediaType{
						Schema: &openapi.SchemaOrRef{Reference: &openapi.Reference{
							Ref: "#/components/schemas/ModelsMediaUploadResponse",
						}},
					}},
				},
			}},
			"400": errorResponse("Bad Request"),
			"401": errorResponse("Unauthorized"),
			"413": errorResponse("Request Entity Too Large"),
			"500": errorResponse("Internal Server Error"),
		},
	}

	download := &openapi.Operation{
		Tags:        []string{"Media"},
		Summary:     "Download encrypted media file by id",
		Description: "Downloads the encrypted media file. Only the intended receiver may download it. The file is retained until the client confirms the download or its TTL expires.",
		ID:          "DownloadMedia",
		Security:    security,
		Parameters: []*openapi.ParameterOrRef{{Parameter: &openapi.Parameter{
			Name:     "id",
			In:       "path",
			Required: true,
			Schema:   &openapi.SchemaOrRef{Schema: &openapi.Schema{Type: "string"}},
		}}},
		Responses: openapi.Responses{
			"200": {Response: &openapi.Response{
				Description: "Encrypted media file",
				Content: map[string]*openapi.MediaTypeOrRef{
					"application/octet-stream": {MediaType: &openapi.MediaType{
						Schema: &openapi.SchemaOrRef{Schema: &openapi.Schema{
							Type:   "string",
							Format: "binary",
						}},
					}},
				},
			}},
			"401": errorResponse("Unauthorized"),
			"403": errorResponse("Forbidden"),
			"404": errorResponse("Not Found"),
			"500": errorResponse("Internal Server Error"),
		},
	}

	item := doc.Paths["/api/media"]
	if item == nil {
		item = &openapi.PathItem{}
		doc.Paths["/api/media"] = item
	}
	item.POST = upload

	item = doc.Paths["/api/media/{id}"]
	if item == nil {
		item = &openapi.PathItem{}
		doc.Paths["/api/media/{id}"] = item
	}
	item.GET = download
}
