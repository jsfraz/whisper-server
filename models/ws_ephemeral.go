package models

import "time"

// Ephemeral frames. The server relays them to the live devices of one peer and
// keeps nothing: no row, no timestamp, no archive, no push. A client that never
// sends them therefore leaves nothing to learn.

// WsTypingRequest is one "I am typing" hint for one peer.
type WsTypingRequest struct {
	PeerId string `json:"peerId" validate:"required,uuid"`
}

// WsPresenceRequest announces the caller's own state to the peers it names. It
// is sent when presence is enabled and again, with Online false, when it is
// turned off or the app goes away.
type WsPresenceRequest struct {
	PeerIds []string `json:"peerIds" validate:"required,min=1,max=200,dive,uuid"`
	Online  bool     `json:"online"`
}

// WsLiveLocationRequest carries one end-to-end encrypted position. The server
// sees a ciphertext, never coordinates.
type WsLiveLocationRequest struct {
	PeerId  string `json:"peerId" validate:"required,uuid"`
	Payload string `json:"payload" validate:"required,max=4096"`
}

// TypingFrame tells a peer that someone is typing.
type TypingFrame struct {
	// V versions the frame; everything from this release on is version 1.
	V      int    `json:"v"`
	PeerId string `json:"peerId"`
}

// PresenceFrame tells a peer that someone is online, or that it just went away
// together with the instant the server saw it.
type PresenceFrame struct {
	V      int    `json:"v"`
	PeerId string `json:"peerId"`
	Online bool   `json:"online"`
	// LastSeenAt is stamped by the server, so a client clock cannot place its
	// own last seen wherever it likes.
	LastSeenAt *time.Time `json:"lastSeenAt,omitempty"`
}

// LiveLocationFrame relays one encrypted position to a peer.
type LiveLocationFrame struct {
	V       int    `json:"v"`
	PeerId  string `json:"peerId"`
	Payload string `json:"payload"`
}
