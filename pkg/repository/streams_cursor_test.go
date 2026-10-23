package repository

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStreamCursorRoundTrip(t *testing.T) {
	cursors := map[string]StreamCursor{
		"typical": {Namespace: "ns-a", Topic: "topic-a", CreatedAt: time.Date(2026, 9, 23, 12, 0, 0, 0, time.UTC), ID: 42},
		// the zero value means the start of retained history, not an invalid cursor
		"zero": {},
	}

	for label, original := range cursors {
		t.Run(label, func(t *testing.T) {
			encoded, err := EncodeStreamCursor(original)
			require.NoError(t, err)
			assert.NotEmpty(t, encoded)

			decoded, err := DecodeStreamCursor(encoded)
			require.NoError(t, err)

			assert.Equal(t, original.Namespace, decoded.Namespace)
			assert.Equal(t, original.Topic, decoded.Topic)
			assert.True(t, original.CreatedAt.Equal(decoded.CreatedAt))
			assert.Equal(t, original.ID, decoded.ID)

			reEncoded, err := EncodeStreamCursor(decoded)
			require.NoError(t, err)
			assert.Equal(t, encoded, reEncoded, "re-encoding must be stable")
		})
	}
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
