package models

// HistoryQuery is one page of a conversation, newest first.
//
//	@param peerId
//	@param before
//	@param limit
type HistoryQuery struct {
	PeerId string `query:"peerId" validate:"required,uuid" description:"The other participant of the conversation"`
	Before string `query:"before" description:"Opaque cursor from the previous page"`
	Limit  int    `query:"limit" description:"Page size, 1-200 (default 50)"`
}

// HistoryPage is one page of the ciphertext archive plus the peers it mentions.
type HistoryPage struct {
	Messages   []PrivateMessage `json:"messages" validate:"required"`
	Peers      []User           `json:"peers" validate:"required"`
	NextCursor string           `json:"nextCursor"`
}
