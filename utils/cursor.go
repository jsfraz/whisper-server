package utils

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Cursor is the position of one message in a conversation: the timestamp plus
// the id, so two messages with the same timestamp still page deterministically.
type Cursor struct {
	SentAt time.Time
	Id     string
}

// EncodeCursor makes an opaque cursor for the client. It carries no secret, but
// the client must not build one itself.
//
//	@param sentAt
//	@param id
//	@return string
func EncodeCursor(sentAt time.Time, id string) string {
	raw := fmt.Sprintf("%d|%s", sentAt.UTC().UnixNano(), id)
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

// DecodeCursor parses a cursor from a client.
//
//	@param value
//	@return *Cursor
//	@return error
func DecodeCursor(value string) (*Cursor, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return nil, fmt.Errorf("invalid cursor")
	}
	parts := strings.SplitN(string(decoded), "|", 2)
	if len(parts) != 2 || parts[1] == "" {
		return nil, fmt.Errorf("invalid cursor")
	}
	nanos, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid cursor")
	}
	return &Cursor{SentAt: time.Unix(0, nanos).UTC(), Id: parts[1]}, nil
}
