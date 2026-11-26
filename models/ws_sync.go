package models

// WsSyncRequest asks for every archived message newer than the client's cursor.
// A fresh device starts at zero and walks the history in batches.
//
// SinceEventSeq is absent on clients that predate the mutation log: without it
// the server sends no events, so an old build cannot get stuck looping over a
// cursor it does not know.
type WsSyncRequest struct {
	SinceSeq      int64  `json:"sinceSeq" validate:"gte=0"`
	SinceEventSeq *int64 `json:"sinceEventSeq" validate:"omitempty,gte=0"`
}

// WsSynced answers a sync request. Peers carries the accounts referenced by the
// batch so a new device can render names without one request per message.
// Events replays edits, deletions, reactions and pins; Cleared replays this
// account's whole-chat cuts.
type WsSynced struct {
	Messages      []PrivateMessage      `json:"messages" validate:"required"`
	Peers         []User                `json:"peers" validate:"required"`
	Events        []MessageEvent        `json:"events" validate:"required"`
	Cleared       []ConversationCleared `json:"cleared" validate:"required"`
	Reads         []ReadReceipt         `json:"reads" validate:"required"`
	SyncedThrough int64                 `json:"syncedThrough" validate:"required"`
	EventsThrough int64                 `json:"eventsThrough" validate:"required"`
	HasMore       bool                  `json:"hasMore" validate:"required"`
}
