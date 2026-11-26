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
	doc.Components.Schemas["ModelsMediaUploadSessionResponse"] = &openapi.SchemaOrRef{Schema: &openapi.Schema{
		Type: "object",
		Properties: map[string]*openapi.SchemaOrRef{
			"id":        {Schema: &openapi.Schema{Type: "string"}},
			"received":  {Schema: &openapi.Schema{Type: "integer", Format: "int64", Description: "Bytes the server holds."}},
			"totalSize": {Schema: &openapi.Schema{Type: "integer", Format: "int64", Description: "Ciphertext length the client declared."}},
		},
		Required: []string{"id", "received", "totalSize"},
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
							Type:        "string",
							Format:      "uuid",
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

	addMediaUploadOperations(gen)
}

// addMediaUploadOperations documents the chunked upload: one raw Gin handler
// (the chunk append) plus the response schemas fizz cannot derive from handlers
// that write JSON themselves.
func addMediaUploadOperations(gen *openapi.Generator) {
	doc := gen.API()
	security := []*openapi.SecurityRequirement{{"bearerAuth": []string{}}}
	sessionSchema := &openapi.SchemaOrRef{Reference: &openapi.Reference{
		Ref: "#/components/schemas/ModelsMediaUploadSessionResponse",
	}}
	sessionBody := func() map[string]*openapi.MediaTypeOrRef {
		return map[string]*openapi.MediaTypeOrRef{
			"application/json": {MediaType: &openapi.MediaType{
				Schema: sessionSchema,
			}},
		}
	}

	if item := doc.Paths["/api/media/uploads"]; item != nil && item.POST != nil {
		item.POST.Responses["200"] = &openapi.ResponseOrRef{Response: &openapi.Response{
			Description: "The upload session, with the offset the server holds.",
			Content:     sessionBody(),
		}}
	}

	chunk := &openapi.Operation{
		Tags:        []string{"Media"},
		Summary:     "Upload one chunk",
		Description: "Appends one chunk of the encrypted media file at the Upload-Offset header. A chunk at a different offset is refused with 409 and the offset the server holds.",
		ID:          "UploadMediaChunk",
		Security:    security,
		Parameters: []*openapi.ParameterOrRef{
			{Parameter: &openapi.Parameter{
				Name:     "id",
				In:       "path",
				Required: true,
				Schema:   &openapi.SchemaOrRef{Schema: &openapi.Schema{Type: "string"}},
			}},
			{Parameter: &openapi.Parameter{
				Name:        "Upload-Offset",
				In:          "header",
				Required:    true,
				Description: "Byte offset this chunk starts at; must equal the received count.",
				Schema:      &openapi.SchemaOrRef{Schema: &openapi.Schema{Type: "integer", Format: "int64"}},
			}},
		},
		RequestBody: &openapi.RequestBody{
			Required: true,
			Content: map[string]*openapi.MediaType{
				"multipart/form-data": {Schema: &openapi.SchemaOrRef{Schema: &openapi.Schema{
					Type: "object",
					Properties: map[string]*openapi.SchemaOrRef{
						"chunk": {Schema: &openapi.Schema{
							Type:        "string",
							Format:      "binary",
							Description: "One slice of the encrypted media blob. Its field name must be chunk.",
						}},
					},
					Required: []string{"chunk"},
				}}},
			},
		},
		Responses: openapi.Responses{
			"200": {Response: &openapi.Response{
				Description: "The upload session, with the new offset.",
				Content:     sessionBody(),
			}},
			"400": errorResponse("Bad Request"),
			"401": errorResponse("Unauthorized"),
			"403": errorResponse("Forbidden"),
			"404": errorResponse("Not Found"),
			"409": errorResponse("Offset Mismatch"),
			"413": errorResponse("Request Entity Too Large"),
			"500": errorResponse("Internal Server Error"),
		},
	}

	item := doc.Paths["/api/media/uploads/{id}"]
	if item == nil {
		item = &openapi.PathItem{}
		doc.Paths["/api/media/uploads/{id}"] = item
	}
	item.PUT = chunk
	if item.GET != nil {
		item.GET.Responses["200"] = &openapi.ResponseOrRef{Response: &openapi.Response{
			Description: "The upload session, with the offset the server holds.",
			Content:     sessionBody(),
		}}
	}

	if item := doc.Paths["/api/media/uploads/{id}/complete"]; item != nil && item.POST != nil {
		item.POST.Responses["200"] = &openapi.ResponseOrRef{Response: &openapi.Response{
			Description: "The media file is complete and downloadable.",
			Content: map[string]*openapi.MediaTypeOrRef{
				"application/json": {MediaType: &openapi.MediaType{
					Schema: &openapi.SchemaOrRef{Reference: &openapi.Reference{
						Ref: "#/components/schemas/ModelsMediaUploadResponse",
					}},
				}},
			},
		}}
	}
}
