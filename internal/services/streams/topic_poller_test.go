package streams

import (
	"context"
	"fmt"
	"maps"
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

	// runs once, after the next read, to interleave work with a reader
	afterNextRead func()
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
	out := f.listLocked(opts)
	hook := f.afterNextRead
	f.afterNextRead = nil
	f.mu.Unlock()

	if hook != nil {
		hook()
	}

	return out, nil
}

func (f *fakeStreamsRepository) listLocked(opts v1.ListStreamMessagesOpts) []*sqlcv1.V1StreamMessage {
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

	return out
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

func TestTopicPollerRegistry_OnePollerPerTopicForAsLongAsItHasListeners(t *testing.T) {
	tenantId := uuid.New()
	repo := newFakeStreamsRepository(tenantId, "", "topic-a", 0, time.Now())
	registry := newTopicPollerRegistry(repo, fakePubSub{}, testLogger(), 5*time.Millisecond, time.Hour)

	keyA := topicPollerKey{tenantId: tenantId, topic: "topic-a"}
	keyB := topicPollerKey{tenantId: tenantId, topic: "topic-b"}

	join := func(key topicPollerKey, l *collectingListener) func() {
		unregister, err := registry.Join(context.Background(), key, v1.StreamCursor{}, &topicListener{send: l.send, cancel: func() {}})
		require.NoError(t, err)
		return unregister
	}

	pollers := func() map[topicPollerKey]*topicPoller {
		registry.mu.Lock()
		defer registry.mu.Unlock()
		return maps.Clone(registry.pollers)
	}

	listenerA1, listenerA2 := &collectingListener{}, &collectingListener{}
	leaveA1 := join(keyA, listenerA1)
	leaveA2 := join(keyA, listenerA2)
	leaveB := join(keyB, &collectingListener{})

	assert.Len(t, pollers(), 2, "listeners on one topic share a poller; topics don't")

	repo.appendMessage(1, time.Now())

	require.Eventually(t, func() bool {
		return listenerA1.count() == 1 && listenerA2.count() == 1
	}, time.Second, 5*time.Millisecond, "the shared poller fans each message out to every listener")

	leaveA1()
	assert.Contains(t, pollers(), keyA, "a poller survives while it has a listener")

	leaveA2()
	assert.NotContains(t, pollers(), keyA, "a poller is removed with its last listener")
	assert.Contains(t, pollers(), keyB)

	leaveB()
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

// A listener whose unlocked replay ends behind the shared poller, because the
// poller moved on meanwhile, must still get the rows it fanned out without it.
func TestTopicPollerRegistry_JoinBehindPollerBackfillsListener(t *testing.T) {
	tenantId := uuid.New()
	base := time.Now()
	repo := newFakeStreamsRepository(tenantId, "", "topic-a", 1, base)
	registry := newTopicPollerRegistry(repo, fakePubSub{}, testLogger(), time.Hour, time.Hour)

	key := topicPollerKey{tenantId: tenantId, topic: "topic-a"}

	join := func(cursor v1.StreamCursor, l *collectingListener) {
		unregister, err := registry.Join(context.Background(), key, cursor, &topicListener{send: l.send, cancel: func() {}})
		require.NoError(t, err)
		t.Cleanup(unregister)
	}

	listenerA, listenerB, listenerC := &collectingListener{}, &collectingListener{}, &collectingListener{}
	join(v1.StreamCursor{}, listenerA)

	// while C replays, message 2 lands and B's join pulls the poller past it
	repo.afterNextRead = func() {
		repo.appendMessage(2, base.Add(time.Second))
		join(v1.StreamCursor{ID: 2}, listenerB)
	}

	join(v1.StreamCursor{ID: 1}, listenerC)

	assert.Equal(t, []int64{1, 2}, listenerA.entryIDs(t))
	assert.Empty(t, listenerB.entryIDs(t))
	assert.Equal(t, []int64{2}, listenerC.entryIDs(t))
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

func TestSendRange(t *testing.T) {
	total := subscribeCatchUpBatchSize*2 + 7

	cases := map[string]struct {
		from    v1.StreamCursor
		maxID   int64
		wantIDs []int64
		wantEnd int64
	}{
		"pages through every row in order":  {maxID: math.MaxInt64, wantIDs: idRange(1, total), wantEnd: int64(total)},
		"stops at maxID":                    {from: v1.StreamCursor{ID: 3}, maxID: 6, wantIDs: []int64{4, 5, 6}, wantEnd: 6},
		"returns from when nothing is sent": {from: v1.StreamCursor{ID: int64(total)}, maxID: math.MaxInt64, wantIDs: []int64{}, wantEnd: int64(total)},
	}

	for label, tc := range cases {
		t.Run(label, func(t *testing.T) {
			tenantId := uuid.New()
			repo := newFakeStreamsRepository(tenantId, "", "topic-a", total, time.Now())
			key := topicPollerKey{tenantId: tenantId, topic: "topic-a"}

			listener := &collectingListener{}
			last, err := sendRange(context.Background(), repo, key, tc.from, tc.maxID, listener.send)
			require.NoError(t, err)

			assert.Equal(t, tc.wantIDs, listener.entryIDs(t))
			assert.Equal(t, tc.wantEnd, last.ID)
		})
	}
}

func idRange(from, to int) []int64 {
	ids := make([]int64, 0, to-from+1)
	for i := from; i <= to; i++ {
		ids = append(ids, int64(i))
	}
	return ids
}

// capturingPubSub records Sub calls and keeps the wake handler so tests can
// deliver wakes directly.
type capturingPubSub struct {
	fakePubSub

	mu      sync.Mutex
	subs    []msgqueue.Topic
	handler msgqueue.MsgHandler
}

func (c *capturingPubSub) Sub(topic msgqueue.Topic, handler msgqueue.MsgHandler) (func() error, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.subs = append(c.subs, topic)
	c.handler = handler

	return func() error { return nil }, nil
}

func (c *capturingPubSub) deliver(t *testing.T, tenantId uuid.UUID, wakes ...msgqueue.StreamWake) {
	t.Helper()

	msg, err := msgqueue.NewTenantMessage(tenantId, msgqueue.MsgIDStreamMessage, true, false, wakes...)
	require.NoError(t, err)

	c.mu.Lock()
	handler := c.handler
	c.mu.Unlock()

	require.NoError(t, handler(msg))
}

func TestTopicPollerRegistry_AllTopicsShareOneWakeSubscription(t *testing.T) {
	tenantId := uuid.New()
	repo := newFakeStreamsRepository(tenantId, "", "topic", 0, time.Now())
	pubsub := &capturingPubSub{}
	registry := newTopicPollerRegistry(repo, pubsub, testLogger(), time.Hour, time.Hour)

	for i := range 25 {
		key := topicPollerKey{tenantId: tenantId, topic: fmt.Sprintf("topic-%d", i)}
		unregister, err := registry.Join(context.Background(), key, v1.StreamCursor{}, &topicListener{send: (&collectingListener{}).send, cancel: func() {}})
		require.NoError(t, err)
		defer unregister()
	}

	assert.Equal(t, []msgqueue.Topic{msgqueue.StreamWakeTopic()}, pubsub.subs)
}

func TestTopicPollerRegistry_WakeReachesOnlyItsTopic(t *testing.T) {
	tenantId := uuid.New()
	base := time.Now()
	repo := newFakeStreamsRepository(tenantId, "ns", "topic-a", 0, base)
	pubsub := &capturingPubSub{}
	// no ticks during the test, so any delivery must come from a wake
	registry := newTopicPollerRegistry(repo, pubsub, testLogger(), time.Hour, time.Hour)

	listener := &collectingListener{}
	unregister, err := registry.Join(context.Background(), topicPollerKey{tenantId: tenantId, namespace: "ns", topic: "topic-a"}, v1.StreamCursor{}, &topicListener{send: listener.send, cancel: func() {}})
	require.NoError(t, err)
	defer unregister()

	repo.appendMessage(1, base.Add(time.Millisecond))

	pubsub.deliver(t, tenantId, msgqueue.StreamWake{Namespace: "ns", Topic: "topic-b"}, msgqueue.StreamWake{Topic: "topic-a"})
	pubsub.deliver(t, uuid.New(), msgqueue.StreamWake{Namespace: "ns", Topic: "topic-a"})

	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, 0, listener.count(), "wakes for other topics, namespaces, or tenants must not poll this one")

	pubsub.deliver(t, tenantId, msgqueue.StreamWake{Namespace: "ns", Topic: "topic-a"})

	require.Eventually(t, func() bool { return listener.count() == 1 }, time.Second, 5*time.Millisecond)
}
