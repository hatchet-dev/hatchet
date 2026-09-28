package streams

import (
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

func TestResolveSubscribeAddressAndCursor_NoTopicNoCursor(t *testing.T) {
	_, _, _, err := resolveSubscribeAddressAndCursor(&contracts.SubscribeStreamRequest{})
	assert.Error(t, err, "topic is required unless cursor is supplied")
}

func TestResolveSubscribeAddressAndCursor_TopicWithNoCursorDefaultsToBeginning(t *testing.T) {
	namespace, topic, cursor, err := resolveSubscribeAddressAndCursor(&contracts.SubscribeStreamRequest{Namespace: "ns-a", Topic: "topic-a"})
	require.NoError(t, err)

	assert.Equal(t, "ns-a", namespace)
	assert.Equal(t, "topic-a", topic)
	assert.Equal(t, v1.StreamCursor{Namespace: "ns-a", Topic: "topic-a", ID: 0}, cursor)
}

func TestResolveSubscribeAddressAndCursor_CursorAloneAddressesTheTopic(t *testing.T) {
	// Round(0) strips the monotonic clock reading time.Now() attaches, which a
	// real cursor never carries once it has been through a JSON round-trip --
	// without this, assert.Equal below would spuriously fail comparing an
	// in-process time.Time against its own post-JSON-round-trip copy.
	cursor := v1.StreamCursor{Namespace: "ns-a", Topic: "topic-a", CreatedAt: time.Now().Round(0), ID: 7}
	raw := encodedCursor(t, cursor)

	namespace, topic, resolved, err := resolveSubscribeAddressAndCursor(&contracts.SubscribeStreamRequest{Cursor: &raw})
	require.NoError(t, err)

	assert.Equal(t, "ns-a", namespace)
	assert.Equal(t, "topic-a", topic)
	assert.Equal(t, cursor, resolved)
}

func TestResolveSubscribeAddressAndCursor_TopicWithMatchingCursor(t *testing.T) {
	cursor := v1.StreamCursor{Namespace: "ns-a", Topic: "topic-a", CreatedAt: time.Now().Round(0), ID: 7}
	raw := encodedCursor(t, cursor)

	namespace, topic, resolved, err := resolveSubscribeAddressAndCursor(&contracts.SubscribeStreamRequest{
		Namespace: "ns-a",
		Topic:     "topic-a",
		Cursor:    &raw,
	})
	require.NoError(t, err)

	assert.Equal(t, "ns-a", namespace)
	assert.Equal(t, "topic-a", topic)
	assert.Equal(t, cursor, resolved)
}

func TestResolveSubscribeAddressAndCursor_RejectsMismatchedTopic(t *testing.T) {
	cursor := v1.StreamCursor{Namespace: "ns-a", Topic: "topic-a", CreatedAt: time.Now(), ID: 7}
	raw := encodedCursor(t, cursor)

	_, _, _, err := resolveSubscribeAddressAndCursor(&contracts.SubscribeStreamRequest{
		Namespace: "ns-a",
		Topic:     "topic-b", // different topic than the cursor's own
		Cursor:    &raw,
	})
	assert.Error(t, err, "a cursor from a different topic must be rejected, not silently applied")
}

func TestResolveSubscribeAddressAndCursor_RejectsMismatchedNamespace(t *testing.T) {
	cursor := v1.StreamCursor{Namespace: "ns-a", Topic: "topic-a", CreatedAt: time.Now(), ID: 7}
	raw := encodedCursor(t, cursor)

	_, _, _, err := resolveSubscribeAddressAndCursor(&contracts.SubscribeStreamRequest{
		Namespace: "ns-b", // different namespace than the cursor's own
		Topic:     "topic-a",
		Cursor:    &raw,
	})
	assert.Error(t, err, "a cursor from a different namespace must be rejected, not silently applied")
}
