package models

// WsSyncRequest asks for every archived message newer than the client's cursor.
// A fresh device starts at zero and walks the history in batches.
type WsSyncRequest struct {
	SinceSeq int64 `json:"sinceSeq" validate:"gte=0"`
}

// WsSynced answers a sync request. Peers carries the accounts referenced by the
// batch so a new device can render names without one request per message.
type WsSynced struct {
	Messages      []PrivateMessage `json:"messages" validate:"required"`
	Peers         []User           `json:"peers" validate:"required"`
	SyncedThrough int64            `json:"syncedThrough" validate:"required"`
	HasMore       bool             `json:"hasMore" validate:"required"`
}
