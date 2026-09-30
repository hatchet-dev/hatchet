package streams

import (
	"context"
	"math"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// fakeStreamsRepository is an in-memory v1.StreamsRepository backing only
// ListMessagesAfterCursor, which is all topicPoller needs.
type fakeStreamsRepository struct {
	v1.StreamsRepository

	mu       sync.Mutex
	messages []*sqlcv1.V1StreamMessage

	// cursors at or below this ID are reported expired
	expiredThroughID int64
}

func (f *fakeStreamsRepository) CheckCursorRetained(_ context.Context, _ uuid.UUID, cursor v1.StreamCursor) error {
	if cursor.ID > 0 && cursor.ID <= f.expiredThroughID {
		return &v1.StreamCursorExpiredError{CursorCreatedAt: cursor.CreatedAt}
	}

	return nil
}

func newFakeStreamsRepository(tenantId uuid.UUID, namespace, topic string, n int, baseTime time.Time) *fakeStreamsRepository {
	msgs := make([]*sqlcv1.V1StreamMessage, 0, n)

	for i := 0; i < n; i++ {
		msgs = append(msgs, &sqlcv1.V1StreamMessage{
			ID:         int64(i + 1),
			InsertedAt: pgtype.Timestamptz{Time: baseTime.Add(time.Duration(i) * time.Millisecond), Valid: true},
			TenantID:   tenantId,
			Namespace:  namespace,
			Topic:      topic,
			Payload:    []byte(topic),
		})
	}

	return &fakeStreamsRepository{messages: msgs}
}

func (f *fakeStreamsRepository) ListMessagesAfterCursor(_ context.Context, _ uuid.UUID, opts v1.ListStreamMessagesOpts) ([]*sqlcv1.V1StreamMessage, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	out := make([]*sqlcv1.V1StreamMessage, 0)

	for _, m := range f.messages {
		after := v1.StreamCursor{CreatedAt: m.InsertedAt.Time, ID: m.ID}
		if after.After(opts.Cursor) {
			out = append(out, m)
		}

		if opts.Limit > 0 && len(out) == int(opts.Limit) {
			break
		}
	}

	return out, nil
}

func (f *fakeStreamsRepository) appendMessage(id int64, at time.Time) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.messages = append(f.messages, &sqlcv1.V1StreamMessage{
		ID:         id,
		InsertedAt: pgtype.Timestamptz{Time: at, Valid: true},
		Payload:    []byte("live"),
	})
}

// fakePubSub is a no-op PubSub: the pooling tests exercise the poller's
// ticker fallback exclusively, so the wake channel is never expected to fire.
type fakePubSub struct{}

func (fakePubSub) Pub(context.Context, msgqueue.Topic, *msgqueue.Message) error { return nil }
func (fakePubSub) Sub(msgqueue.Topic, msgqueue.MsgHandler) (func() error, error) {
	return func() error { return nil }, nil
}
func (fakePubSub) IsReady() bool { return true }

func testLogger() *zerolog.Logger {
	l := zerolog.Nop()
	return &l
}

// collectingListener records every message it receives, safe for concurrent
// use since the shared poller may deliver from its own goroutine.
type collectingListener struct {
	mu       sync.Mutex
	received []*contracts.StreamMessage
}

func (c *collectingListener) send(msg *contracts.StreamMessage) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.received = append(c.received, msg)
	return nil
}

func (c *collectingListener) count() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.received)
}

func (c *collectingListener) entryIDs(t *testing.T) []int64 {
	c.mu.Lock()
	defer c.mu.Unlock()

	ids := []int64{}

	for _, msg := range c.received {
		for _, e := range msg.Entries {
			cursor, err := v1.DecodeStreamCursor(e.Cursor)
			require.NoError(t, err)
			ids = append(ids, cursor.ID)
		}
	}

	return ids
}

func TestTopicPollerRegistry_SharesOnePollerPerKey(t *testing.T) {
	tenantId := uuid.New()
	repo := newFakeStreamsRepository(tenantId, "", "topic-a", 0, time.Now())
	registry := newTopicPollerRegistry(repo, fakePubSub{}, testLogger(), 5*time.Millisecond, time.Hour)

	key := topicPollerKey{tenantId: tenantId, namespace: "", topic: "topic-a"}

	listenerA := &collectingListener{}
	unregisterA, err := registry.Join(context.Background(), key, v1.StreamCursor{}, &topicListener{
		send:   listenerA.send,
		cancel: func() {},
	})
	require.NoError(t, err)
	defer unregisterA()

	listenerB := &collectingListener{}
	unregisterB, err := registry.Join(context.Background(), key, v1.StreamCursor{}, &topicListener{
		send:   listenerB.send,
		cancel: func() {},
	})
	require.NoError(t, err)
	defer unregisterB()

	registry.mu.Lock()
	numPollers := len(registry.pollers)
	registry.mu.Unlock()

	assert.Equal(t, 1, numPollers, "two listeners on the same (tenant, namespace, topic) must share one poller")

	// publish a message "live" and let the shared ticker pick it up
	repo.appendMessage(1, time.Now())

	require.Eventually(t, func() bool {
		return listenerA.count() == 1 && listenerB.count() == 1
	}, time.Second, 5*time.Millisecond, "both listeners should receive the same fanned-out message")

	assert.Equal(t, listenerA.received[0].Entries[0].Cursor, listenerB.received[0].Entries[0].Cursor, "both listeners should see the identical message/cursor")
}

func TestTopicPollerRegistry_DifferentTopicsGetDifferentPollers(t *testing.T) {
	tenantId := uuid.New()
	repo := newFakeStreamsRepository(tenantId, "", "topic-a", 0, time.Now())
	registry := newTopicPollerRegistry(repo, fakePubSub{}, testLogger(), 5*time.Millisecond, time.Hour)

	unregisterA, err := registry.Join(context.Background(), topicPollerKey{tenantId: tenantId, topic: "topic-a"}, v1.StreamCursor{}, &topicListener{send: func(*contracts.StreamMessage) error { return nil }, cancel: func() {}})
	require.NoError(t, err)
	defer unregisterA()

	unregisterB, err := registry.Join(context.Background(), topicPollerKey{tenantId: tenantId, topic: "topic-b"}, v1.StreamCursor{}, &topicListener{send: func(*contracts.StreamMessage) error { return nil }, cancel: func() {}})
	require.NoError(t, err)
	defer unregisterB()

	registry.mu.Lock()
	numPollers := len(registry.pollers)
	registry.mu.Unlock()

	assert.Equal(t, 2, numPollers, "distinct topics must not share a poller")
}

func TestTopicPollerRegistry_RemovesPollerWhenLastListenerLeaves(t *testing.T) {
	tenantId := uuid.New()
	repo := newFakeStreamsRepository(tenantId, "", "topic-a", 0, time.Now())
	registry := newTopicPollerRegistry(repo, fakePubSub{}, testLogger(), 5*time.Millisecond, time.Hour)

	key := topicPollerKey{tenantId: tenantId, topic: "topic-a"}

	unregisterA, err := registry.Join(context.Background(), key, v1.StreamCursor{}, &topicListener{send: func(*contracts.StreamMessage) error { return nil }, cancel: func() {}})
	require.NoError(t, err)

	unregisterB, err := registry.Join(context.Background(), key, v1.StreamCursor{}, &topicListener{send: func(*contracts.StreamMessage) error { return nil }, cancel: func() {}})
	require.NoError(t, err)

	unregisterA()

	registry.mu.Lock()
	_, stillPresent := registry.pollers[key]
	registry.mu.Unlock()
	assert.True(t, stillPresent, "poller must survive while at least one listener remains")

	unregisterB()

	registry.mu.Lock()
	_, stillPresent = registry.pollers[key]
	registry.mu.Unlock()
	assert.False(t, stillPresent, "poller must be removed once its last listener leaves")
}

func TestTopicPollerRegistry_JoinCatchesUpBeforeAttaching(t *testing.T) {
	tenantId := uuid.New()
	base := time.Now()
	repo := newFakeStreamsRepository(tenantId, "", "topic-a", 0, base)
	// A large tail interval means the poller will not tick on its own during
	// this test -- any delivery must come from join()'s own catch-up-on-join
	// path, which is exactly what this test is verifying.
	registry := newTopicPollerRegistry(repo, fakePubSub{}, testLogger(), time.Hour, time.Hour)

	key := topicPollerKey{tenantId: tenantId, topic: "topic-a"}

	// first listener starts the poller at "now" (nothing persisted yet)
	listenerA := &collectingListener{}
	unregisterA, err := registry.Join(context.Background(), key, v1.StreamCursor{}, &topicListener{send: listenerA.send, cancel: func() {}})
	require.NoError(t, err)
	defer unregisterA()

	// a message lands directly in the fake repo (as if another Subscribe
	// call's own catch-up had already seen it) without the shared poller
	// ever ticking
	repo.appendMessage(1, base.Add(time.Millisecond))

	// a second listener joins whose own catch-up cursor is already past
	// where the (idle, not-yet-ticked) poller sits -- it must pull everyone,
	// including listener A, forward before attaching
	listenerB := &collectingListener{}
	startCursor := v1.StreamCursor{CreatedAt: base.Add(time.Millisecond), ID: 1}
	unregisterB, err := registry.Join(context.Background(), key, startCursor, &topicListener{send: listenerB.send, cancel: func() {}})
	require.NoError(t, err)
	defer unregisterB()

	assert.Equal(t, 1, listenerA.count(), "the pre-existing listener must receive the message the new joiner had already caught up to")
	assert.Equal(t, 0, listenerB.count(), "the new listener already saw this message during its own catch-up and must not receive it again")
}

func TestTopicPollerRegistry_JoinReplaysHistoryToOnlyTheNewListener(t *testing.T) {
	tenantId := uuid.New()
	repo := newFakeStreamsRepository(tenantId, "", "topic-a", 3, time.Now())
	registry := newTopicPollerRegistry(repo, fakePubSub{}, testLogger(), time.Hour, time.Hour)

	key := topicPollerKey{tenantId: tenantId, topic: "topic-a"}

	listenerA := &collectingListener{}
	unregisterA, err := registry.Join(context.Background(), key, v1.StreamCursor{ID: 3}, &topicListener{send: listenerA.send, cancel: func() {}})
	require.NoError(t, err)
	defer unregisterA()

	listenerB := &collectingListener{}
	unregisterB, err := registry.Join(context.Background(), key, v1.StreamCursor{}, &topicListener{send: listenerB.send, cancel: func() {}})
	require.NoError(t, err)
	defer unregisterB()

	assert.Empty(t, listenerA.entryIDs(t))
	assert.Equal(t, []int64{1, 2, 3}, listenerB.entryIDs(t))
}

func TestTopicPollerRegistry_JoinRejectsExpiredCursor(t *testing.T) {
	tenantId := uuid.New()
	repo := newFakeStreamsRepository(tenantId, "", "topic-a", 3, time.Now())
	repo.expiredThroughID = 2
	registry := newTopicPollerRegistry(repo, fakePubSub{}, testLogger(), time.Hour, time.Hour)

	key := topicPollerKey{tenantId: tenantId, topic: "topic-a"}

	listener := &collectingListener{}
	_, err := registry.Join(context.Background(), key, v1.StreamCursor{ID: 1}, &topicListener{send: listener.send, cancel: func() {}})

	var expired *v1.StreamCursorExpiredError
	require.ErrorAs(t, err, &expired)
	assert.Equal(t, 0, listener.count(), "an expired cursor must not replay anything")
	assert.Empty(t, registry.pollers, "an expired cursor must not start a poller")
}

// A listener whose own catch-up ended behind the shared poller must still get
// the rows the poller had already fanned out to earlier listeners.
func TestTopicPollerRegistry_JoinBehindPollerBackfillsListener(t *testing.T) {
	tenantId := uuid.New()
	base := time.Now()
	repo := newFakeStreamsRepository(tenantId, "", "topic-a", 0, base)
	registry := newTopicPollerRegistry(repo, fakePubSub{}, testLogger(), time.Hour, time.Hour)

	key := topicPollerKey{tenantId: tenantId, topic: "topic-a"}

	listenerA := &collectingListener{}
	unregisterA, err := registry.Join(context.Background(), key, v1.StreamCursor{}, &topicListener{send: listenerA.send, cancel: func() {}})
	require.NoError(t, err)
	defer unregisterA()

	repo.appendMessage(1, base.Add(time.Millisecond))
	repo.appendMessage(2, base.Add(2*time.Millisecond))

	// pulls the shared poller forward to id=2
	listenerB := &collectingListener{}
	unregisterB, err := registry.Join(context.Background(), key, v1.StreamCursor{ID: 2}, &topicListener{send: listenerB.send, cancel: func() {}})
	require.NoError(t, err)
	defer unregisterB()

	// its own catch-up only reached id=1 before the poller advanced
	listenerC := &collectingListener{}
	unregisterC, err := registry.Join(context.Background(), key, v1.StreamCursor{ID: 1}, &topicListener{send: listenerC.send, cancel: func() {}})
	require.NoError(t, err)
	defer unregisterC()

	assert.Equal(t, []int64{1, 2}, listenerA.entryIDs(t))
	assert.Empty(t, listenerB.entryIDs(t))
	assert.Equal(t, []int64{2}, listenerC.entryIDs(t))
}

func TestTopicPoller_BatchesMultipleMessagesIntoOneStreamMessage(t *testing.T) {
	tenantId := uuid.New()
	base := time.Now()
	// pre-populate several messages before the poller's first tick, so a
	// single ListMessagesAfterCursor call sees all of them at once -- this is
	// exactly the catch-up-from-an-old-cursor scenario batching optimizes for.
	repo := newFakeStreamsRepository(tenantId, "", "topic-a", 5, base)
	registry := newTopicPollerRegistry(repo, fakePubSub{}, testLogger(), 5*time.Millisecond, time.Hour)

	key := topicPollerKey{tenantId: tenantId, topic: "topic-a"}

	listener := &collectingListener{}
	unregister, err := registry.Join(context.Background(), key, v1.StreamCursor{}, &topicListener{send: listener.send, cancel: func() {}})
	require.NoError(t, err)
	defer unregister()

	require.Eventually(t, func() bool {
		return listener.count() > 0
	}, time.Second, 5*time.Millisecond)

	// give any further ticks time to run (they should find nothing new)
	// before asserting exactly one frame was ever sent
	time.Sleep(20 * time.Millisecond)

	require.Len(t, listener.received, 1, "messages arriving together should be batched into a single StreamMessage frame")
	assert.Len(t, listener.received[0].Entries, 5, "the single frame should carry every entry")
}

func TestTopicPoller_HangsUpIdleListenersAndStopsItself(t *testing.T) {
	tenantId := uuid.New()
	repo := newFakeStreamsRepository(tenantId, "", "topic-a", 0, time.Now())
	// a short tick interval combined with a short idle timeout so the hangup
	// fires quickly without needing to fabricate time
	registry := newTopicPollerRegistry(repo, fakePubSub{}, testLogger(), 5*time.Millisecond, 20*time.Millisecond)

	key := topicPollerKey{tenantId: tenantId, topic: "topic-a"}

	listener := &collectingListener{}
	// mirrors server.go's Subscribe handler: cancel is a plain context
	// CancelFunc, and a separate goroutine (the RPC handler, here simulated)
	// is the one blocked on ctx.Done() that reacts by calling unregister().
	ctx, cancel := context.WithCancel(context.Background())
	unregister, err := registry.Join(context.Background(), key, v1.StreamCursor{}, &topicListener{
		send:   listener.send,
		cancel: cancel,
	})
	require.NoError(t, err)

	done := make(chan struct{})
	go func() {
		<-ctx.Done()
		unregister()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("listener was never canceled by idle hangup")
	}

	require.Equal(t, 1, listener.count(), "listener should receive exactly one hangup message")
	assert.True(t, listener.received[0].Hangup, "the final message must be marked hangup")

	registry.mu.Lock()
	_, stillPresent := registry.pollers[key]
	registry.mu.Unlock()
	assert.False(t, stillPresent, "poller must remove itself from the registry after hanging up its only listener")
}

func TestSendRange_PagesThroughEveryRowInOrder(t *testing.T) {
	tenantId := uuid.New()
	total := subscribeCatchUpBatchSize*2 + 7
	repo := newFakeStreamsRepository(tenantId, "", "topic-a", total, time.Now())
	key := topicPollerKey{tenantId: tenantId, topic: "topic-a"}

	listener := &collectingListener{}
	last, err := sendRange(context.Background(), repo, key, v1.StreamCursor{}, math.MaxInt64, listener.send)
	require.NoError(t, err)

	ids := listener.entryIDs(t)
	require.Len(t, ids, total)
	for i, id := range ids {
		assert.Equal(t, int64(i+1), id)
	}
	assert.Equal(t, int64(total), last.ID)
}

func TestSendRange_StopsAtMaxID(t *testing.T) {
	tenantId := uuid.New()
	repo := newFakeStreamsRepository(tenantId, "", "topic-a", 10, time.Now())
	key := topicPollerKey{tenantId: tenantId, topic: "topic-a"}

	listener := &collectingListener{}
	last, err := sendRange(context.Background(), repo, key, v1.StreamCursor{ID: 3}, 6, listener.send)
	require.NoError(t, err)

	assert.Equal(t, []int64{4, 5, 6}, listener.entryIDs(t))
	assert.Equal(t, int64(6), last.ID)
}

func TestSendRange_ReturnsStartCursorWhenNothingToSend(t *testing.T) {
	tenantId := uuid.New()
	repo := newFakeStreamsRepository(tenantId, "", "topic-a", 3, time.Now())
	key := topicPollerKey{tenantId: tenantId, topic: "topic-a"}
	from := v1.StreamCursor{ID: 3}

	sent := 0
	last, err := sendRange(context.Background(), repo, key, from, math.MaxInt64, func(*contracts.StreamMessage) error {
		sent++
		return nil
	})
	require.NoError(t, err)

	assert.Equal(t, 0, sent)
	assert.Equal(t, from, last)
}
