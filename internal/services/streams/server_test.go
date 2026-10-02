package streams

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
)

func encodedCursor(t *testing.T, c v1.StreamCursor) string {
	t.Helper()

	s, err := v1.EncodeStreamCursor(c)
	require.NoError(t, err)

	return s
}

func TestResolveSubscribeAddressAndCursor(t *testing.T) {
	tooLong := strings.Repeat("a", v1.MaxStreamNameLength+1)
	ownCursor := encodedCursor(t, v1.StreamCursor{Namespace: "ns-a", Topic: "topic-a", CreatedAt: time.Now(), ID: 7})
	// a cursor is client-supplied, so the address decoded from it is checked too
	tooLongCursor := encodedCursor(t, v1.StreamCursor{Namespace: tooLong, Topic: "orders", ID: 1})

	t.Run("a topic without a cursor starts at the beginning", func(t *testing.T) {
		namespace, topic, cursor, err := resolveSubscribeAddressAndCursor(&contracts.SubscribeStreamRequest{Namespace: "ns-a", Topic: "topic-a"})
		require.NoError(t, err)

		assert.Equal(t, "ns-a", namespace)
		assert.Equal(t, "topic-a", topic)
		assert.Equal(t, v1.StreamCursor{Namespace: "ns-a", Topic: "topic-a", ID: 0}, cursor)
	})

	rejected := map[string]struct {
		req     *contracts.SubscribeStreamRequest
		wantErr string
	}{
		"no topic and no cursor":            {req: &contracts.SubscribeStreamRequest{}},
		"a cursor from another topic":       {req: &contracts.SubscribeStreamRequest{Namespace: "ns-a", Topic: "topic-b", Cursor: &ownCursor}},
		"a cursor from another namespace":   {req: &contracts.SubscribeStreamRequest{Namespace: "ns-b", Topic: "topic-a", Cursor: &ownCursor}},
		"a topic that's too long":           {req: &contracts.SubscribeStreamRequest{Topic: tooLong}, wantErr: "topic is 256 characters long"},
		"a cursor whose address is invalid": {req: &contracts.SubscribeStreamRequest{Cursor: &tooLongCursor}, wantErr: "namespace is 256 characters long"},
	}

	for label, tc := range rejected {
		t.Run("rejects "+label, func(t *testing.T) {
			_, _, _, err := resolveSubscribeAddressAndCursor(tc.req)
			require.Error(t, err)

			if tc.wantErr != "" {
				assert.ErrorContains(t, err, tc.wantErr)
			}
		})
	}
}
