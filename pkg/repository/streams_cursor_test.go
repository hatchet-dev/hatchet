package repository

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStreamCursorRoundTrip(t *testing.T) {
	original := StreamCursor{
		Namespace: "ns-a",
		Topic:     "topic-a",
		CreatedAt: time.Date(2026, 9, 23, 12, 0, 0, 0, time.UTC),
		ID:        42,
	}

	encoded, err := EncodeStreamCursor(original)
	require.NoError(t, err)
	assert.NotEmpty(t, encoded)

	decoded, err := DecodeStreamCursor(encoded)
	require.NoError(t, err)

	assert.Equal(t, original.Namespace, decoded.Namespace)
	assert.Equal(t, original.Topic, decoded.Topic)
	assert.True(t, original.CreatedAt.Equal(decoded.CreatedAt))
	assert.Equal(t, original.ID, decoded.ID)

	// re-encoding the decoded cursor should be stable
	reEncoded, err := EncodeStreamCursor(decoded)
	require.NoError(t, err)
	assert.Equal(t, encoded, reEncoded)
}

func TestStreamCursorDecodeMalformed(t *testing.T) {
	cases := []string{
		"",
		"not-a-cursor",
		"v2:abc123",              // unrecognized version
		"v1:not-valid-base64!!!", // invalid base64
		"v1:" + "bm90IGpzb24=",   // valid base64, but not valid JSON payload ("not json")
	}

	for _, c := range cases {
		_, err := DecodeStreamCursor(c)
		assert.Error(t, err, "expected an error decoding %q", c)
	}
}

func TestStreamCursorZeroValueIsValidStartOfHistory(t *testing.T) {
	// the zero value of StreamCursor (epoch time, id 0) is used to mean
	// "the beginning of retained history" -- confirm it round-trips like any
	// other cursor rather than being treated as a special/invalid case.
	var zero StreamCursor

	encoded, err := EncodeStreamCursor(zero)
	require.NoError(t, err)

	decoded, err := DecodeStreamCursor(encoded)
	require.NoError(t, err)

	assert.True(t, zero.CreatedAt.Equal(decoded.CreatedAt))
	assert.Equal(t, zero.ID, decoded.ID)
}
