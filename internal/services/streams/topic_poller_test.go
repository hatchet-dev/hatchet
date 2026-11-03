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
// what topicPoller needs.
type fakeStreamsRepository struct {
	v1.StreamsRepository

	mu       sync.Mutex
	messages []*sqlcv1.V1StreamMessage

	lastOffset     int64
	topicCreatedAt time.Time
	reads          int

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

	return &fakeStreamsRepository{messages: msgs, lastOffset: int64(n), topicCreatedAt: baseTime}
}

func (f *fakeStreamsRepository) GetTopicInfo(_ context.Context, _ uuid.UUID, namespace, topic string) (v1.StreamTopicInfo, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	return v1.StreamTopicInfo{Namespace: namespace, Topic: topic, LastOffset: f.lastOffset, CreatedAt: f.topicCreatedAt}, nil
}

// recreateTopic stands in for the idle sweep deleting the topic and a publish recreating it
func (f *fakeStreamsRepository) recreateTopic() {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.messages = nil
	f.lastOffset = 0
	f.topicCreatedAt = time.Now()
}

func (f *fakeStreamsRepository) readCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.reads
}

// reserveOffset stands in for a rejected publish, which reserves an offset it never stores
func (f *fakeStreamsRepository) reserveOffset() {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.lastOffset++
}

func (f *fakeStreamsRepository) ListMessagesAfterCursor(_ context.Context, _ uuid.UUID, opts v1.ListStreamMessagesOpts) ([]*sqlcv1.V1StreamMessage, error) {
	f.mu.Lock()
	f.reads++
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
		// appendMessage leaves Topic empty, delivering to every topic
		if m.Topic != "" && m.Topic != opts.Topic {
			continue
		}

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

	f.lastOffset = max(f.lastOffset, id)
	f.messages = append(f.messages, &sqlcv1.V1StreamMessage{
		ID:         id,
		InsertedAt: pgtype.Timestamptz{Time: at, Valid: true},
		Payload:    []byte("live"),
	})
}

func (f *fakeStreamsRepository) appendTopicMessage(topic string, id int64) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.lastOffset = max(f.lastOffset, id)
	f.messages = append(f.messages, &sqlcv1.V1StreamMessage{
		ID:         id,
		InsertedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
		Topic:      topic,
		Payload:    []byte(topic),
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

// hangup returns the cursor of the hangup frame received, if any.
func (c *collectingListener) hangup(t *testing.T) (v1.StreamCursor, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, msg := range c.received {
		if msg.Hangup {
			cursor, err := v1.DecodeStreamCursor(msg.Cursor)
			require.NoError(t, err)
			return cursor, true
		}
	}

	return v1.StreamCursor{}, false
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

func TestTopicPoller_ReadsMessagesOnlyAfterAnOffsetIsReserved(t *testing.T) {
	tenantId := uuid.New()
	repo := newFakeStreamsRepository(tenantId, "", "topic", 0, time.Now())
	registry := newTopicPollerRegistry(repo, fakePubSub{}, testLogger(), 5*time.Millisecond, time.Hour)
	listener := &collectingListener{}

	unregister, err := registry.Join(context.Background(), topicPollerKey{tenantId: tenantId, topic: "topic"}, v1.StreamCursor{}, &topicListener{send: listener.send, cancel: func() {}})
	require.NoError(t, err)
	defer unregister()

	idleReads := func() int {
		before := repo.readCount()
		time.Sleep(50 * time.Millisecond)
		return repo.readCount() - before
	}

	assert.Zero(t, idleReads(), "ticks on an idle topic skip the message read")

	repo.appendMessage(1, time.Now())
	require.Eventually(t, func() bool { return listener.count() == 1 }, time.Second, 5*time.Millisecond)
	assert.Zero(t, idleReads(), "once delivered, the topic is idle again")

	readsBefore := repo.readCount()
	repo.reserveOffset()
	require.Eventually(t, func() bool { return repo.readCount() > readsBefore }, time.Second, 5*time.Millisecond)
	assert.Zero(t, idleReads(), "an offset reserved but never stored is read once, not every tick")
	assert.Equal(t, 1, listener.count())
}

// A send blocks once the reader's HTTP/2 flow-control window is full, as here.
func TestTopicPollerRegistry_StalledListenerAffectsOnlyItself(t *testing.T) {
	tenantId := uuid.New()
	repo := newFakeStreamsRepository(tenantId, "", "stalled", 0, time.Now())
	registry := newTopicPollerRegistry(repo, fakePubSub{}, testLogger(), 5*time.Millisecond, time.Hour)

	stalledKey := topicPollerKey{tenantId: tenantId, topic: "stalled"}
	otherKey := topicPollerKey{tenantId: tenantId, topic: "other"}

	release := make(chan struct{})
	defer close(release)

	stalled := make(chan struct{})
	var stallOnce sync.Once
	stalledSend := func(*contracts.StreamMessage) error {
		stallOnce.Do(func() { close(stalled) })
		<-release
		return nil
	}

	unregisterStalled, err := registry.Join(context.Background(), stalledKey, v1.StreamCursor{}, &topicListener{send: stalledSend, cancel: func() {}})
	require.NoError(t, err)
	t.Cleanup(unregisterStalled)

	repo.appendTopicMessage("stalled", 1)

	select {
	case <-stalled:
	case <-time.After(time.Second):
		t.Fatal("the poller never delivered to the stalled listener")
	}

	type joined struct {
		unregister func()
		err        error
	}

	joinAsync := func(key topicPollerKey, l *collectingListener) <-chan joined {
		done := make(chan joined, 1)
		go func() {
			unregister, err := registry.Join(context.Background(), key, v1.StreamCursor{}, &topicListener{send: l.send, cancel: func() {}})
			done <- joined{unregister, err}
		}()
		return done
	}

	healthySameTopic := &collectingListener{}
	sameTopicJoined := joinAsync(stalledKey, healthySameTopic)

	// lets the same-topic Join reach the poller before the unrelated one starts
	time.Sleep(50 * time.Millisecond)

	other := &collectingListener{}

	select {
	case j := <-joinAsync(otherKey, other):
		require.NoError(t, j.err)
		t.Cleanup(j.unregister)
	case <-time.After(time.Second):
		t.Fatal("a Join on an unrelated topic blocked behind the stalled listener")
	}

	repo.appendTopicMessage("other", 2)

	require.Eventually(t, func() bool { return other.count() == 1 }, time.Second, 5*time.Millisecond,
		"live delivery on an unrelated topic is unaffected")

	select {
	case j := <-sameTopicJoined:
		require.NoError(t, j.err)
		t.Cleanup(j.unregister)
	case <-time.After(time.Second):
		t.Fatal("a healthy Join on the stalled listener's topic blocked behind it")
	}
}

func TestTopicPoller_DropsAListenerThatFallsTooFarBehind(t *testing.T) {
	tenantId := uuid.New()
	repo := newFakeStreamsRepository(tenantId, "", "topic", 0, time.Now())
	registry := newTopicPollerRegistry(repo, fakePubSub{}, testLogger(), 5*time.Millisecond, time.Hour)
	key := topicPollerKey{tenantId: tenantId, topic: "topic"}

	release := make(chan struct{})
	defer close(release)

	dropped := make(chan struct{})
	var dropOnce sync.Once

	unregisterStalled, err := registry.Join(context.Background(), key, v1.StreamCursor{}, &topicListener{
		send:   func(*contracts.StreamMessage) error { <-release; return nil },
		cancel: func() { dropOnce.Do(func() { close(dropped) }) },
	})
	require.NoError(t, err)
	t.Cleanup(unregisterStalled)

	healthy := &collectingListener{}
	unregisterHealthy, err := registry.Join(context.Background(), key, v1.StreamCursor{}, &topicListener{send: healthy.send, cancel: func() {}})
	require.NoError(t, err)
	t.Cleanup(unregisterHealthy)

	// one frame per poll, so the stalled queue fills one frame at a time
	frames := listenerQueueSize + 2
	for id := 1; id <= frames; id++ {
		repo.appendMessage(int64(id), time.Now())
		require.Eventually(t, func() bool { return healthy.count() == id }, time.Second, time.Millisecond)
	}

	select {
	case <-dropped:
	case <-time.After(time.Second):
		t.Fatal("a listener whose queue overflowed was never dropped")
	}

	assert.Equal(t, frames, healthy.count(), "the healthy listener on the same topic received every frame")
}

func TestTopicPoller_RetiresWithItsLastListenerForGood(t *testing.T) {
	tenantId := uuid.New()
	repo := newFakeStreamsRepository(tenantId, "", "topic", 0, time.Now())
	registry := newTopicPollerRegistry(repo, fakePubSub{}, testLogger(), 5*time.Millisecond, time.Hour)
	key := topicPollerKey{tenantId: tenantId, topic: "topic"}

	first := &collectingListener{}
	leaveFirst, err := registry.Join(context.Background(), key, v1.StreamCursor{}, &topicListener{send: first.send, cancel: func() {}})
	require.NoError(t, err)

	repo.appendMessage(1, time.Now())
	require.Eventually(t, func() bool { return first.count() == 1 }, time.Second, 5*time.Millisecond)

	registry.mu.Lock()
	retired := registry.pollers[key]
	registry.mu.Unlock()

	leaveFirst()

	_, err = retired.join(context.Background(), time.Time{}, v1.StreamCursor{}, &topicListener{send: (&collectingListener{}).send, cancel: func() {}})
	assert.ErrorIs(t, err, errPollerRetired, "a poller never restarts, so no state outlives its listeners")

	second := &collectingListener{}
	leaveSecond, err := registry.Join(context.Background(), key, v1.StreamCursor{}, &topicListener{send: second.send, cancel: func() {}})
	require.NoError(t, err)
	t.Cleanup(leaveSecond)

	registry.mu.Lock()
	assert.NotSame(t, retired, registry.pollers[key], "the next Join starts a new poller")
	registry.mu.Unlock()

	repo.appendMessage(2, time.Now())
	require.Eventually(t, func() bool { return len(second.entryIDs(t)) == 2 }, time.Second, 5*time.Millisecond)
	assert.Equal(t, []int64{1, 2}, second.entryIDs(t))
}

func TestTopicPoller_FollowsARecreatedTopic(t *testing.T) {
	tenantId := uuid.New()
	repo := newFakeStreamsRepository(tenantId, "", "topic", 0, time.Now())
	registry := newTopicPollerRegistry(repo, fakePubSub{}, testLogger(), 5*time.Millisecond, time.Hour)

	listener := &collectingListener{}
	unregister, err := registry.Join(context.Background(), topicPollerKey{tenantId: tenantId, topic: "topic"}, v1.StreamCursor{}, &topicListener{send: listener.send, cancel: func() {}})
	require.NoError(t, err)
	t.Cleanup(unregister)

	for id := int64(1); id <= 3; id++ {
		repo.appendMessage(id, time.Now())
	}

	require.Eventually(t, func() bool { return len(listener.entryIDs(t)) == 3 }, time.Second, 5*time.Millisecond)

	repo.recreateTopic()
	repo.appendMessage(1, time.Now())

	var resumeAt v1.StreamCursor
	require.Eventually(t, func() bool {
		var ok bool
		resumeAt, ok = listener.hangup(t)
		return ok
	}, time.Second, 5*time.Millisecond, "the poller on the old topic retires")

	assert.Equal(t, v1.StreamCursor{Topic: "topic"}, resumeAt, "listeners resume at the new topic's start, a cursor that can't expire")
	assert.Equal(t, []int64{1, 2, 3}, listener.entryIDs(t))

	// the client resubscribing from the hangup's cursor
	resubscribed := &collectingListener{}
	unregisterResubscribed, err := registry.Join(context.Background(), topicPollerKey{tenantId: tenantId, topic: "topic"}, resumeAt, &topicListener{send: resubscribed.send, cancel: func() {}})
	require.NoError(t, err)
	t.Cleanup(unregisterResubscribed)

	assert.Equal(t, []int64{1}, resubscribed.entryIDs(t))
}

func TestTopicPollerRegistry_JoinAfterARecreateThePollerMissedGetsEachMessageOnce(t *testing.T) {
	tenantId := uuid.New()
	repo := newFakeStreamsRepository(tenantId, "", "topic", 3, time.Now().Add(-time.Hour))
	registry := newTopicPollerRegistry(repo, fakePubSub{}, testLogger(), time.Hour, time.Hour)
	key := topicPollerKey{tenantId: tenantId, topic: "topic"}

	existing := &collectingListener{}
	unregisterExisting, err := registry.Join(context.Background(), key, v1.StreamCursor{}, &topicListener{send: existing.send, cancel: func() {}})
	require.NoError(t, err)
	t.Cleanup(unregisterExisting)
	require.Equal(t, []int64{1, 2, 3}, existing.entryIDs(t))

	registry.mu.Lock()
	old := registry.pollers[key]
	registry.mu.Unlock()

	// the poller ticks only on wakes here, so it hasn't seen the recreate when the second subscriber joins
	repo.recreateTopic()
	repo.appendMessage(1, time.Now())
	repo.appendMessage(2, time.Now())

	joined := &collectingListener{}
	unregisterJoined, err := registry.Join(context.Background(), key, v1.StreamCursor{}, &topicListener{send: joined.send, cancel: func() {}})
	require.NoError(t, err)
	t.Cleanup(unregisterJoined)

	require.Eventually(t, func() bool {
		_, ok := existing.hangup(t)
		return ok
	}, time.Second, 5*time.Millisecond, "the join retires the poller on the old topic")

	registry.mu.Lock()
	p := registry.pollers[key]
	registry.mu.Unlock()
	require.NotSame(t, old, p, "the joining subscriber starts a new poller")

	readsBefore := repo.readCount()
	p.wakeUp()
	require.Eventually(t, func() bool { return repo.readCount() > readsBefore }, time.Second, time.Millisecond)

	// the queue goroutines get a moment to deliver anything the wake fanned out
	time.Sleep(50 * time.Millisecond)

	assert.Equal(t, []int64{1, 2}, joined.entryIDs(t), "the joining subscriber gets each recreated message once")
	assert.Equal(t, []int64{1, 2, 3}, existing.entryIDs(t), "the existing listener gets the new topic by resubscribing, not in place")
}

func TestTopicPoller_JoinWithACursorFromBeforeARecreateStartsAtTheNewTopic(t *testing.T) {
	tenantId := uuid.New()
	// the idle sweep only deletes topics with nothing published within retention
	base := time.Now().Add(-time.Hour)
	repo := newFakeStreamsRepository(tenantId, "", "topic", 3, base)
	key := topicPollerKey{tenantId: tenantId, topic: "topic"}
	p := newTopicPoller(repo, testLogger(), key, 5*time.Millisecond, time.Hour, func(*topicPoller) {})

	// a replay that ended at the old topic's last message
	oldCursor := v1.StreamCursor{Topic: "topic", CreatedAt: base.Add(2 * time.Millisecond), ID: 3}

	repo.recreateTopic()
	repo.appendMessage(1, time.Now())

	info, err := repo.GetTopicInfo(context.Background(), tenantId, "", "topic")
	require.NoError(t, err)

	listener := &collectingListener{}
	id, err := p.join(context.Background(), info.CreatedAt, oldCursor, &topicListener{send: listener.send, cancel: func() {}})
	require.NoError(t, err)
	t.Cleanup(func() { p.leave(id) })

	require.Eventually(t, func() bool { return len(listener.entryIDs(t)) == 1 }, time.Second, 5*time.Millisecond,
		"offsets below the old cursor belong to the new topic")
	assert.Equal(t, []int64{1}, listener.entryIDs(t))
}

func TestTopicPollerRegistry_UnregisterIsIdempotent(t *testing.T) {
	tenantId := uuid.New()
	repo := newFakeStreamsRepository(tenantId, "", "topic", 0, time.Now())
	registry := newTopicPollerRegistry(repo, fakePubSub{}, testLogger(), time.Hour, time.Hour)
	key := topicPollerKey{tenantId: tenantId, topic: "topic"}

	join := func() func() {
		unregister, err := registry.Join(context.Background(), key, v1.StreamCursor{}, &topicListener{send: (&collectingListener{}).send, cancel: func() {}})
		require.NoError(t, err)
		return unregister
	}

	leaveFirst := join()
	leaveSecond := join()
	t.Cleanup(leaveSecond)

	leaveFirst()
	leaveFirst()

	registry.mu.Lock()
	_, present := registry.pollers[key]
	registry.mu.Unlock()
	assert.True(t, present, "a repeated unregister doesn't retire the poller another listener is on")
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

	require.Eventually(t, func() bool { return listenerA.count() == 1 }, time.Second, 5*time.Millisecond,
		"the pre-existing listener must receive the message the new joiner had already caught up to")
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

	require.Eventually(t, func() bool { return len(listenerA.entryIDs(t)) == 2 && len(listenerC.entryIDs(t)) == 1 }, time.Second, 5*time.Millisecond)
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
