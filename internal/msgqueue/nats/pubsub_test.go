//go:build integration

package nats

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/rand"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
)

const testNATSURL = "nats://127.0.0.1:4222"

func newTestPubSub(t *testing.T) *PubSub {
	t.Helper()

	cleanup, ps, err := NewPubSub(WithPubSubURL(testNATSURL))
	require.NoError(t, err)
	require.NotNil(t, ps)

	t.Cleanup(func() {
		if err := cleanup(); err != nil {
			t.Errorf("error cleaning up pubsub: %v", err)
		}
	})

	return ps
}

func receiveN(t *testing.T, ctx context.Context, ch <-chan *msgqueue.Message, n int) []*msgqueue.Message {
	t.Helper()

	out := make([]*msgqueue.Message, 0, n)
	for len(out) < n {
		select {
		case msg := <-ch:
			out = append(out, msg)
		case <-ctx.Done():
			t.Fatalf("timed out waiting for pubsub delivery: got %d of %d", len(out), n)
		}
	}
	return out
}

func TestPubSubTenantFanout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ps := newTestPubSub(t)

	tenantId := uuid.New()
	topic := msgqueue.TenantTopic(tenantId)

	msg, err := msgqueue.NewTenantMessage(tenantId, "task-completed", true, false, map[string]interface{}{"key": "value"})
	require.NoError(t, err)

	received := make(chan *msgqueue.Message, 2)

	handler := func(m *msgqueue.Message) error {
		received <- m
		return nil
	}

	cleanupSub1, err := ps.Sub(topic, handler)
	require.NoError(t, err)

	cleanupSub2, err := ps.Sub(topic, handler)
	require.NoError(t, err)

	require.NoError(t, ps.Pub(ctx, topic, msg))

	got := receiveN(t, ctx, received, 2)
	for _, m := range got {
		assert.Equal(t, msg.ID, m.ID)
	}

	require.NoError(t, cleanupSub1())
	require.NoError(t, cleanupSub2())
}

func TestPubSubSchedulerTopicRoundtrip(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ps := newTestPubSub(t)

	topic := msgqueue.SchedulerPartitionTopic(uuid.NewString())

	msg, err := msgqueue.NewTenantMessage(uuid.New(), "check-tenant-queue", true, false, map[string]interface{}{"key": "value"})
	require.NoError(t, err)

	received := make(chan *msgqueue.Message, 1)

	cleanupSub, err := ps.Sub(topic, func(m *msgqueue.Message) error {
		received <- m
		return nil
	})
	require.NoError(t, err)

	require.NoError(t, ps.Pub(ctx, topic, msg))

	got := receiveN(t, ctx, received, 1)
	assert.Equal(t, msg.ID, got[0].ID)

	require.NoError(t, cleanupSub())
}

func TestPubSubLargePayload(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	ps := newTestPubSub(t)

	tenantId := uuid.New()
	topic := msgqueue.TenantTopic(tenantId)

	payload := make([]byte, 8*1024*1024)
	_, err := rand.Read(payload)
	require.NoError(t, err)

	msg, err := msgqueue.NewTenantMessage(tenantId, "task-stream-event", true, false, map[string]interface{}{"data": payload})
	require.NoError(t, err)

	originalPayload := make([]byte, len(msg.Payloads[0]))
	copy(originalPayload, msg.Payloads[0])

	received := make(chan *msgqueue.Message, 1)

	cleanupSub, err := ps.Sub(topic, func(m *msgqueue.Message) error {
		received <- m
		return nil
	})
	require.NoError(t, err)

	require.NoError(t, ps.Pub(ctx, topic, msg))

	got := receiveN(t, ctx, received, 1)
	assert.Equal(t, msg.ID, got[0].ID)
	require.Len(t, got[0].Payloads, 1)
	assert.Equal(t, originalPayload, got[0].Payloads[0])

	require.NoError(t, cleanupSub())
}

func TestPubSubCompressedRoundtrip(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ps := newTestPubSub(t)

	tenantId := uuid.New()
	topic := msgqueue.TenantTopic(tenantId)

	plain := []byte("hello-compressed-payload-for-nats-pubsub")
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	_, err := zw.Write(plain)
	require.NoError(t, err)
	require.NoError(t, zw.Close())

	msg := &msgqueue.Message{
		ID:         "task-stream-event",
		TenantID:   tenantId,
		Payloads:   [][]byte{buf.Bytes()},
		Compressed: true,
	}

	received := make(chan *msgqueue.Message, 1)

	cleanupSub, err := ps.Sub(topic, func(m *msgqueue.Message) error {
		received <- m
		return nil
	})
	require.NoError(t, err)

	require.NoError(t, ps.Pub(ctx, topic, msg))

	got := receiveN(t, ctx, received, 1)
	assert.Equal(t, msg.ID, got[0].ID)
	require.Len(t, got[0].Payloads, 1)
	assert.Equal(t, plain, got[0].Payloads[0], "payload should be transparently decompressed")

	require.NoError(t, cleanupSub())
}
