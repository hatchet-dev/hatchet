package repository

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// lets the format change, e.g. to point at object storage, without breaking refs in stored messages
const streamPayloadRefVersion = "p1"

// StreamPayloadRef identifies an uploaded payload. CreatedAt is its partition
// key, so a lookup reads one partition.
type StreamPayloadRef struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
}

func EncodeStreamPayloadRef(r StreamPayloadRef) (string, error) {
	body, err := json.Marshal(r)
	if err != nil {
		return "", fmt.Errorf("could not encode stream payload ref: %w", err)
	}

	return streamPayloadRefVersion + ":" + base64.RawURLEncoding.EncodeToString(body), nil
}

func DecodeStreamPayloadRef(s string) (StreamPayloadRef, error) {
	version, encoded, ok := strings.Cut(s, ":")
	if !ok || version != streamPayloadRefVersion {
		return StreamPayloadRef{}, fmt.Errorf("invalid or unrecognized stream payload ref")
	}

	body, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return StreamPayloadRef{}, fmt.Errorf("invalid stream payload ref encoding: %w", err)
	}

	var ref StreamPayloadRef
	if err := json.Unmarshal(body, &ref); err != nil {
		return StreamPayloadRef{}, fmt.Errorf("invalid stream payload ref payload: %w", err)
	}

	return ref, nil
}
