package repository

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// streamCursorVersion prefixes every encoded cursor so the internal format can
// change later without breaking clients holding an older cursor -- the cursor
// is opaque to SDK callers by design.
const streamCursorVersion = "v1"

// StreamCursor identifies a position in a durable stream topic's message
// history. It is never exposed to callers directly -- see EncodeStreamCursor
// / DecodeStreamCursor.
//
// Namespace/Topic are embedded (not just CreatedAt/ID) so a cursor is fully
// self-describing: a caller only needs to persist the cursor string to resume
// later, without separately tracking which topic it belongs to. This also
// closes a footgun that would otherwise exist silently -- id is a single
// sequence shared by every topic in the tenant (see v1_stream_message), so
// nothing at the query level stops a cursor from topic A being applied to
// topic B; embedding the topic lets the caller (internal/services/streams)
// detect and reject that mismatch instead of quietly resuming the wrong
// topic from an unrelated position.
type StreamCursor struct {
	Namespace string    `json:"namespace"`
	Topic     string    `json:"topic"`
	CreatedAt time.Time `json:"created_at"`
	ID        int64     `json:"id"`
}

// After reports whether c is strictly after other in stream order
// (CreatedAt, then ID as a tiebreaker) -- the same ordering
// ListStreamMessagesAfterCursor uses. Namespace/Topic don't participate in
// ordering, only identification.
func (c StreamCursor) After(other StreamCursor) bool {
	if c.CreatedAt.After(other.CreatedAt) {
		return true
	}

	return c.CreatedAt.Equal(other.CreatedAt) && c.ID > other.ID
}

// EncodeStreamCursor produces the opaque cursor string returned to callers.
func EncodeStreamCursor(c StreamCursor) (string, error) {
	body, err := json.Marshal(c)
	if err != nil {
		return "", fmt.Errorf("could not encode stream cursor: %w", err)
	}

	return streamCursorVersion + ":" + base64.RawURLEncoding.EncodeToString(body), nil
}

// DecodeStreamCursor parses a cursor string previously returned by
// EncodeStreamCursor. It returns a clear error for malformed or
// unrecognized-version cursors rather than panicking.
func DecodeStreamCursor(s string) (StreamCursor, error) {
	version, encoded, ok := strings.Cut(s, ":")
	if !ok || version != streamCursorVersion {
		return StreamCursor{}, fmt.Errorf("invalid or unrecognized stream cursor")
	}

	body, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return StreamCursor{}, fmt.Errorf("invalid stream cursor encoding: %w", err)
	}

	var cursor StreamCursor
	if err := json.Unmarshal(body, &cursor); err != nil {
		return StreamCursor{}, fmt.Errorf("invalid stream cursor payload: %w", err)
	}

	return cursor, nil
}
