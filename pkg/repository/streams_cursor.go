package repository

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// lets the format change without breaking cursors clients already hold
const streamCursorVersion = "v1"

// StreamCursor is a position in a topic. It carries its topic so a cursor
// alone can resume a subscription.
type StreamCursor struct {
	Namespace string    `json:"namespace"`
	Topic     string    `json:"topic"`
	CreatedAt time.Time `json:"created_at"`
	ID        int64     `json:"id"`
}

// After compares by ID only: CreatedAt is transaction start time, not ordered by ID.
func (c StreamCursor) After(other StreamCursor) bool {
	return c.ID > other.ID
}

func EncodeStreamCursor(c StreamCursor) (string, error) {
	body, err := json.Marshal(c)
	if err != nil {
		return "", fmt.Errorf("could not encode stream cursor: %w", err)
	}

	return streamCursorVersion + ":" + base64.RawURLEncoding.EncodeToString(body), nil
}

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
