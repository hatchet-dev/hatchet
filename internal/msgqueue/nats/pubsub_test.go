//go:build integration

package nats

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	natsgo "github.com/nats-io/nats.go"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	prommetrics "github.com/hatchet-dev/hatchet/pkg/integrations/metrics/prometheus"
)

const testNATSURL = "nats://127.0.0.1:4222"

func newTestPubSub(t *testing.T, opts ...PubSubOpt) *PubSub {
	t.Helper()

	opts = append([]PubSubOpt{WithPubSubURL(testNATSURL)}, opts...)
	cleanup, ps, err := NewPubSub(opts...)
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

func counterValue(t *testing.T, c prometheus.Counter) float64 {
	t.Helper()

	m := &dto.Metric{}
	require.NoError(t, c.Write(m))

	return m.GetCounter().GetValue()
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

// handlerSlots is how many handler calls a subscription runs at once, so a test
// can block every one of them.
func handlerSlots(topic msgqueue.Topic) int {
	return handlerLimit(topic.Kind())
}

func TestPubSubSkipsStaleMessagesBehindBacklog(t *testing.T) {
	tests := []struct {
		name     string
		topic    msgqueue.Topic
		staleAge time.Duration
		freshAge time.Duration
	}{
		{"scheduler partition", msgqueue.SchedulerPartitionTopic(uuid.NewString()), 6 * time.Second, time.Second},
		{"tenant stream", msgqueue.TenantTopic(uuid.New()), 31 * time.Second, 6 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			ps := newTestPubSub(t)
			skipped := prommetrics.PubSubStaleSkipped.WithLabelValues("nats", string(tt.topic.Kind()))
			skippedBefore := counterValue(t, skipped)

			plugs := handlerSlots(tt.topic)
			backlog := 2 * minBacklogForStaleSkip

			started := make(chan struct{}, plugs)
			unblock := make(chan struct{})
			var unblockOnce sync.Once
			release := func() { unblockOnce.Do(func() { close(unblock) }) }
			received := make(chan string, plugs+backlog+3)

			cleanupSub, err := ps.Sub(tt.topic, func(m *msgqueue.Message) error {
				if m.ID == "plug" {
					started <- struct{}{}
					<-unblock
				}
				received <- m.ID
				return nil
			})
			require.NoError(t, err)
			t.Cleanup(func() { _ = cleanupSub() })
			// registered after the sub cleanup so it runs first
			t.Cleanup(release)

			publish := func(id string, publishedAt time.Time) {
				msg, err := msgqueue.NewTenantMessage(uuid.New(), id, true, false, map[string]interface{}{"key": "value"})
				require.NoError(t, err)
				msg.PublishedAt = publishedAt
				require.NoError(t, ps.Pub(ctx, tt.topic, msg))
			}

			// block every handler so the messages below queue up behind them
			for range plugs {
				publish("plug", time.Now())
			}
			for range plugs {
				select {
				case <-started:
				case <-ctx.Done():
					t.Fatal("plug handlers did not start")
				}
			}

			now := time.Now()
			publish("stale", now.Add(-tt.staleAge))
			publish("fresh", now.Add(-tt.freshAge))
			publish("unstamped", time.Time{})
			for range backlog {
				publish("filler", time.Now())
			}
			require.NoError(t, ps.nc.Flush())

			release()

			counts := map[string]int{}
			for range plugs + backlog + 2 {
				select {
				case id := <-received:
					counts[id]++
				case <-ctx.Done():
					t.Fatalf("timed out waiting for deliveries: got %v", counts)
				}
			}

			assert.Equal(t, map[string]int{"plug": plugs, "fresh": 1, "unstamped": 1, "filler": backlog}, counts)
			assert.Equal(t, skippedBefore+1, counterValue(t, skipped))

			select {
			case id := <-received:
				t.Fatalf("unexpected delivery of %q", id)
			case <-time.After(200 * time.Millisecond):
			}
		})
	}
}

// A stale stamp on a subscription with no backlog can only come from clock
// skew between pods, so the message is delivered.
func TestPubSubDeliversStaleMessagesWithoutBacklog(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ps := newTestPubSub(t)
	topic := msgqueue.SchedulerPartitionTopic(uuid.NewString())
	skipped := prommetrics.PubSubStaleSkipped.WithLabelValues("nats", string(topic.Kind()))
	skippedBefore := counterValue(t, skipped)

	received := make(chan *msgqueue.Message, 1)

	cleanupSub, err := ps.Sub(topic, func(m *msgqueue.Message) error {
		received <- m
		return nil
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = cleanupSub() })

	msg, err := msgqueue.NewTenantMessage(uuid.New(), "skewed", true, false, map[string]interface{}{"key": "value"})
	require.NoError(t, err)
	msg.PublishedAt = time.Now().Add(-time.Minute)
	require.NoError(t, ps.Pub(ctx, topic, msg))

	got := receiveN(t, ctx, received, 1)
	assert.Equal(t, "skewed", got[0].ID)
	assert.Equal(t, skippedBefore, counterValue(t, skipped))
}

func TestPubSubLargePayload(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	ps := newTestPubSub(t)

	tenantId := uuid.New()
	topic := msgqueue.TenantTopic(tenantId)

	// 8MiB of random bytes lands at roughly 14MiB on the wire, because the
	// []byte is base64-encoded twice: once by NewTenantMessage into the payload
	// JSON, then again when the enclosing Message is marshaled. Much larger and
	// a lone payload exceeds max_payload, which Pub cannot chunk its way out of.
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

// An oversized multi-payload message must be split recursively until each chunk
// fits under max_payload, with every payload delivered exactly once.
func TestPubSubOversizedMessageIsChunked(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	ps := newTestPubSub(t)

	tenantId := uuid.New()
	topic := msgqueue.TenantTopic(tenantId)

	// 16 payloads of 1MiB each is ~28MiB on the wire (see the double-encoding
	// note above), so Pub must split before anything can be delivered.
	const numPayloads = 16

	payloads := make([]map[string]interface{}, numPayloads)
	for i := range payloads {
		b := make([]byte, 1024*1024)
		_, err := rand.Read(b)
		require.NoError(t, err)
		payloads[i] = map[string]interface{}{"data": b}
	}

	msg, err := msgqueue.NewTenantMessage(tenantId, "task-stream-event", true, false, payloads...)
	require.NoError(t, err)
	require.Len(t, msg.Payloads, numPayloads)

	want := make([][]byte, numPayloads)
	for i, p := range msg.Payloads {
		want[i] = bytes.Clone(p)
	}

	received := make(chan *msgqueue.Message, numPayloads)

	cleanupSub, err := ps.Sub(topic, func(m *msgqueue.Message) error {
		received <- m
		return nil
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = cleanupSub() })

	require.NoError(t, ps.Pub(ctx, topic, msg))

	var got [][]byte
	var numChunks int

	for len(got) < numPayloads {
		select {
		case m := <-received:
			numChunks++
			assert.Equal(t, msg.ID, m.ID, "chunks must preserve message metadata")
			got = append(got, m.Payloads...)
		case <-ctx.Done():
			t.Fatalf("timed out waiting for chunks: got %d of %d payloads across %d chunks", len(got), numPayloads, numChunks)
		}
	}

	assert.Greater(t, numChunks, 1, "message should have been split into multiple chunks")
	assert.ElementsMatch(t, want, got, "every payload should arrive exactly once; handlers run concurrently, so chunks may arrive in any order")
}

func TestPubSubHandlersRunConcurrently(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ps := newTestPubSub(t)
	topic := msgqueue.SchedulerPartitionTopic(uuid.NewString())

	const n = 5

	started := make(chan struct{}, n)
	unblock := make(chan struct{})

	cleanupSub, err := ps.Sub(topic, func(m *msgqueue.Message) error {
		started <- struct{}{}
		<-unblock
		return nil
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = cleanupSub() })
	defer close(unblock)

	for range n {
		msg, err := msgqueue.NewTenantMessage(uuid.New(), "check-tenant-queue", true, false, map[string]interface{}{"key": "value"})
		require.NoError(t, err)
		require.NoError(t, ps.Pub(ctx, topic, msg))
	}

	// every handler is blocked, so all n can only have started concurrently
	for i := range n {
		select {
		case <-started:
		case <-ctx.Done():
			t.Fatalf("only %d of %d handlers started while the others were blocked", i, n)
		}
	}
}

func TestPubSubFullPoolDefersInsteadOfDropping(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ps := newTestPubSub(t)
	topic := msgqueue.TenantTopic(uuid.New())

	limit := handlerLimit(topic.Kind())
	total := limit + 10

	var running, handled atomic.Int64
	unblock := make(chan struct{})
	var unblockOnce sync.Once
	release := func() { unblockOnce.Do(func() { close(unblock) }) }

	cleanupSub, err := ps.Sub(topic, func(m *msgqueue.Message) error {
		running.Add(1)
		<-unblock
		running.Add(-1)
		handled.Add(1)
		return nil
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = cleanupSub() })
	// registered after the sub cleanup so it runs first: cleanup waits for the
	// blocked handlers if an assertion below fails
	t.Cleanup(release)

	for range total {
		msg, err := msgqueue.NewTenantMessage(uuid.New(), "task-completed", true, false, map[string]interface{}{"key": "value"})
		require.NoError(t, err)
		require.NoError(t, ps.Pub(ctx, topic, msg))
	}

	require.Eventually(t, func() bool { return running.Load() == int64(limit) }, 5*time.Second, time.Millisecond)

	time.Sleep(100 * time.Millisecond)
	assert.EqualValues(t, limit, running.Load(), "no more than the limit run at once")
	assert.EqualValues(t, 0, handled.Load())

	release()

	require.Eventually(t, func() bool { return handled.Load() == int64(total) }, 5*time.Second, time.Millisecond,
		"messages beyond the limit are handled once slots free up")
}

func TestPubSubCleanupWaitsForRunningHandlers(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ps := newTestPubSub(t)
	topic := msgqueue.TenantTopic(uuid.New())

	started := make(chan struct{}, 1)
	unblock := make(chan struct{})
	var finished, calls atomic.Int64

	cleanupSub, err := ps.Sub(topic, func(m *msgqueue.Message) error {
		calls.Add(1)
		started <- struct{}{}
		<-unblock
		finished.Add(1)
		return nil
	})
	require.NoError(t, err)

	publish := func() {
		msg, err := msgqueue.NewTenantMessage(uuid.New(), "task-completed", true, false, map[string]interface{}{"key": "value"})
		require.NoError(t, err)
		require.NoError(t, ps.Pub(ctx, topic, msg))
	}

	publish()

	select {
	case <-started:
	case <-ctx.Done():
		t.Fatal("handler did not start")
	}

	cleanupDone := make(chan error, 1)
	go func() { cleanupDone <- cleanupSub() }()

	select {
	case <-cleanupDone:
		t.Fatal("cleanup returned while a handler was still running")
	case <-time.After(100 * time.Millisecond):
	}

	close(unblock)

	select {
	case err := <-cleanupDone:
		require.NoError(t, err)
	case <-ctx.Done():
		t.Fatal("cleanup did not return after the handler finished")
	}

	assert.EqualValues(t, 1, finished.Load())

	publish()
	time.Sleep(200 * time.Millisecond)
	assert.EqualValues(t, 1, calls.Load(), "no handler runs after cleanup returns")
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

func TestPubSubCustomSubjectPrefix(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	prefix := "custom.prefix"
	ps := newTestPubSub(t, WithPubSubSubjectPrefix(prefix))

	tenantId := uuid.New()
	topic := msgqueue.TenantTopic(tenantId)
	expectedSubject := prefix + "." + topic.Name()

	msg, err := msgqueue.NewTenantMessage(tenantId, "task-completed", true, false, map[string]interface{}{"key": "value"})
	require.NoError(t, err)

	receivedPS := make(chan *msgqueue.Message, 1)
	cleanupSub, err := ps.Sub(topic, func(m *msgqueue.Message) error {
		receivedPS <- m
		return nil
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = cleanupSub() })

	nc, err := natsgo.Connect(testNATSURL)
	require.NoError(t, err)
	t.Cleanup(nc.Close)

	rawReceived := make(chan *natsgo.Msg, 1)
	rawSub, err := nc.Subscribe(expectedSubject, func(m *natsgo.Msg) {
		rawReceived <- m
	})
	require.NoError(t, err)
	require.NoError(t, nc.Flush())
	t.Cleanup(func() { _ = rawSub.Unsubscribe() })

	require.NoError(t, ps.Pub(ctx, topic, msg))

	got := receiveN(t, ctx, receivedPS, 1)
	assert.Equal(t, msg.ID, got[0].ID)

	select {
	case <-rawReceived:
	case <-ctx.Done():
		t.Fatal("timed out waiting for raw NATS delivery on custom prefix subject")
	}
}
