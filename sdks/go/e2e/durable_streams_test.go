//go:build e2e

package e2e

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/hatchet-dev/hatchet/sdks/go/features"
)

// collectStreamEvents reads count events, then stops iterating, which cancels the subscription.
func collectStreamEvents(t *testing.T, ctx context.Context, topic string, count int, opts ...features.StreamCallOpt) []features.StreamEvent {
	t.Helper()

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var events []features.StreamEvent
	for event, err := range sharedClient.Streams().Events(ctx, topic, opts...) {
		require.NoError(t, err)
		events = append(events, event)
		if len(events) >= count {
			break
		}
	}

	require.Len(t, events, count)

	return events
}

func payloads(events []features.StreamEvent) []string {
	out := make([]string, len(events))
	for i, e := range events {
		out[i] = string(e.Payload)
	}
	return out
}

func TestDurableStreamsReadsFromTheBeginningThenLive(t *testing.T) {
	ctx := newTestContext(t)
	topic := "durable_streams_from_beginning_" + uniqueID()
	streams := sharedClient.Streams()

	require.NoError(t, streams.Publish(ctx, topic, []byte("published-before-consume")))

	go func() {
		time.Sleep(200 * time.Millisecond)
		for _, m := range []string{"live-1", "live-2"} {
			assert.NoError(t, streams.Publish(ctx, topic, []byte(m)))
		}
	}()

	events := collectStreamEvents(t, ctx, topic, 3)
	assert.Equal(t, []string{"published-before-consume", "live-1", "live-2"}, payloads(events))
}

func TestDurableStreamsResumesFromCursor(t *testing.T) {
	ctx := newTestContext(t)
	topic := "durable_streams_resume_cursor_" + uniqueID()
	streams := sharedClient.Streams()

	for _, m := range []string{"one", "two", "three", "four"} {
		require.NoError(t, streams.Publish(ctx, topic, []byte(m)))
	}

	firstTwo := collectStreamEvents(t, ctx, topic, 2)
	assert.Equal(t, []string{"one", "two"}, payloads(firstTwo))

	rest := collectStreamEvents(t, ctx, topic, 2, features.WithStreamCursor(firstTwo[1].Cursor))
	assert.Equal(t, []string{"three", "four"}, payloads(rest))
}

func TestDurableStreamsResetStartsNewReadersAfterIt(t *testing.T) {
	ctx := newTestContext(t)
	topic := "durable_streams_reset_" + uniqueID()
	streams := sharedClient.Streams()

	for _, m := range []string{"one", "two", "three"} {
		require.NoError(t, streams.Publish(ctx, topic, []byte(m)))
	}

	first := collectStreamEvents(t, ctx, topic, 1)
	require.NoError(t, streams.Reset(ctx, topic, features.WithStreamCursor(first[0].Cursor)))
	assert.Equal(t, []string{"two", "three"}, payloads(collectStreamEvents(t, ctx, topic, 2)))

	require.NoError(t, streams.Reset(ctx, topic))
	require.NoError(t, streams.Publish(ctx, topic, []byte("after-reset")))
	assert.Equal(t, []string{"after-reset"}, payloads(collectStreamEvents(t, ctx, topic, 1)))
}

func TestDurableStreamsIsolatesNamespaces(t *testing.T) {
	ctx := newTestContext(t)
	topic := "durable_streams_namespace_" + uniqueID()
	streams := sharedClient.Streams()

	require.NoError(t, streams.Publish(ctx, topic, []byte("from-a"), features.WithStreamNamespace("ns-a")))
	require.NoError(t, streams.Publish(ctx, topic, []byte("from-b"), features.WithStreamNamespace("ns-b")))

	events := collectStreamEvents(t, ctx, topic, 1, features.WithStreamNamespace("ns-b"))
	assert.Equal(t, []string{"from-b"}, payloads(events))
}

func TestDurableStreamsTopicMetadata(t *testing.T) {
	ctx := newTestContext(t)
	topic := "durable_streams_metadata_" + uniqueID()
	streams := sharedClient.Streams()

	for _, m := range []string{"first", "second", "third"} {
		require.NoError(t, streams.Publish(ctx, topic, []byte(m)))
	}

	md, err := streams.TopicMetadata(ctx, topic)
	require.NoError(t, err)
	assert.Equal(t, topic, md.Topic)
	assert.Equal(t, int64(3), md.MessageCount)
	require.NotNil(t, md.LastPublishedAt)

	events := collectStreamEvents(t, ctx, topic, 3)
	require.NotNil(t, md.LatestCursor)
	assert.Equal(t, events[2].Cursor, *md.LatestCursor)

	_, err = streams.TopicMetadata(ctx, "durable_streams_missing_"+uniqueID())
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestDurableStreamsLargePayload(t *testing.T) {
	ctx := newTestContext(t)
	topic := "durable_streams_large_payload_" + uniqueID()
	streams := sharedClient.Streams()

	large := bytes.Repeat([]byte("x"), 5*1024*1024)

	require.NoError(t, streams.Publish(ctx, topic, large))
	require.NoError(t, streams.Publish(ctx, topic, []byte("small")))

	events := collectStreamEvents(t, ctx, topic, 2)
	assert.Equal(t, large, events[0].Payload)
	assert.Equal(t, "small", string(events[1].Payload))
}
