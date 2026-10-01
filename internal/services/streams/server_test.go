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

func TestResolveSubscribeAddressAndCursor_RejectsInvalidAddress(t *testing.T) {
	tooLong := strings.Repeat("a", v1.MaxStreamNameLength+1)

	_, _, _, err := resolveSubscribeAddressAndCursor(&contracts.SubscribeStreamRequest{Topic: tooLong})
	assert.ErrorContains(t, err, "topic is 256 characters long")

	// a cursor is client-supplied, so the address decoded from it is checked too
	cursor, err := v1.EncodeStreamCursor(v1.StreamCursor{Namespace: tooLong, Topic: "orders", ID: 1})
	require.NoError(t, err)

	_, _, _, err = resolveSubscribeAddressAndCursor(&contracts.SubscribeStreamRequest{Cursor: &cursor})
	assert.ErrorContains(t, err, "namespace is 256 characters long")
}
