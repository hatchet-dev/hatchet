package streams

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	tasktypes "github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes/v1"
	hatcheterrors "github.com/hatchet-dev/hatchet/pkg/errors"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
)

// fakeStreamsRepo is an in-memory v1.StreamsRepository modeling exactly the
// compare-and-swap semantics of InsertOrderedStreamMessage/
// ForceInsertOrderedStreamMessage (see pkg/repository/sqlcv1/streams.sql),
// which is all the controller needs.
type fakeStreamsRepo struct {
	v1.StreamsRepository

	mu      sync.Mutex
	cursors map[string]int64 // -1 means "no row yet"
	ordered []v1.CreateOrderedStreamMessageOpts
	forced  []v1.CreateOrderedStreamMessageOpts
}

func newFakeStreamsRepo() *fakeStreamsRepo {
	return &fakeStreamsRepo{cursors: make(map[string]int64)}
}

func cursorKey(tenantId uuid.UUID, opts v1.CreateOrderedStreamMessageOpts) string {
	return tenantId.String() + "|" + opts.Namespace + "|" + opts.Topic + "|" + opts.ProducerID
}

func (f *fakeStreamsRepo) InsertOrderedStreamMessage(_ context.Context, tenantId uuid.UUID, opts v1.CreateOrderedStreamMessageOpts) (v1.OrderedStreamMessageResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	k := cursorKey(tenantId, opts)

	current, ok := f.cursors[k]
	if !ok {
		current = -1
	}

	if opts.ProducerSeq != current+1 {
		return v1.OrderedStreamMessageResult{Inserted: false, CurrentSeq: current}, nil
	}

	f.cursors[k] = opts.ProducerSeq
	f.ordered = append(f.ordered, opts)

	return v1.OrderedStreamMessageResult{Inserted: true, CurrentSeq: opts.ProducerSeq}, nil
}

func (f *fakeStreamsRepo) ForceInsertOrderedStreamMessage(_ context.Context, tenantId uuid.UUID, opts v1.CreateOrderedStreamMessageOpts) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	k := cursorKey(tenantId, opts)

	if current, ok := f.cursors[k]; !ok || opts.ProducerSeq > current {
		f.cursors[k] = opts.ProducerSeq
	}

	f.forced = append(f.forced, opts)

	return nil
}

type fakeRepository struct {
	v1.Repository
	streams *fakeStreamsRepo
}

func (f *fakeRepository) Streams() v1.StreamsRepository { return f.streams }

// fakeMessageQueue records every SendMessage call, which is how the
// controller re-publishes a producer-sequenced message that arrived ahead of
// its predecessor.
type fakeMessageQueue struct {
	msgqueue.MessageQueue

	mu   sync.Mutex
	sent []*msgqueue.Message
}

func (f *fakeMessageQueue) SendMessage(_ context.Context, _ msgqueue.Queue, msg *msgqueue.Message) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.sent = append(f.sent, msg)

	return nil
}

func (f *fakeMessageQueue) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()

	return len(f.sent)
}

// fakePubSub records every wake published.
type fakePubSub struct {
	msgqueue.PubSub

	mu       sync.Mutex
	messages []*msgqueue.Message
}

func (f *fakePubSub) Pub(_ context.Context, topic msgqueue.Topic, msg *msgqueue.Message) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if topic != msgqueue.StreamWakeTopic() {
		return fmt.Errorf("unexpected topic %s", topic.Name())
	}

	f.messages = append(f.messages, msg)

	return nil
}

func (f *fakePubSub) wakes(t *testing.T) []msgqueue.StreamWake {
	f.mu.Lock()
	defer f.mu.Unlock()

	var wakes []msgqueue.StreamWake

	for _, msg := range f.messages {
		for _, payload := range msg.Payloads {
			var w msgqueue.StreamWake
			require.NoError(t, json.Unmarshal(payload, &w))
			wakes = append(wakes, w)
		}
	}

	return wakes
}

func newTestController(repo *fakeStreamsRepo, mq *fakeMessageQueue) *ControllerImpl {
	return newTestControllerWithPubSub(repo, mq, &fakePubSub{})
}

func newTestControllerWithPubSub(repo *fakeStreamsRepo, mq *fakeMessageQueue, pubsub *fakePubSub) *ControllerImpl {
	l := zerolog.Nop()

	return &ControllerImpl{
		mq:     mq,
		pubsub: pubsub,
		repo:   &fakeRepository{streams: repo},
		l:      &l,
		a:      hatcheterrors.NewWrapped(hatcheterrors.NoOpAlerter{}),
	}
}

func requireApplied(t *testing.T, want bool, applied bool, err error) {
	t.Helper()
	require.NoError(t, err)
	assert.Equal(t, want, applied)
}

func TestInsertOrderedStreamMessage_InOrderInsertsSequentially(t *testing.T) {
	repo := newFakeStreamsRepo()
	c := newTestController(repo, &fakeMessageQueue{})
	tenantId := uuid.New()

	for seq := int64(0); seq < 3; seq++ {
		msg := &tasktypes.StreamMessagePayload{
			Topic: "t", ProducerID: "p1", ProducerSeq: seq, CreatedAt: time.Now(),
		}
		applied, err := c.insertOrderedStreamMessage(context.Background(), tenantId, msg)
		requireApplied(t, true, applied, err)
	}

	require.Len(t, repo.ordered, 3)
	for i, opts := range repo.ordered {
		assert.Equal(t, int64(i), opts.ProducerSeq)
	}
}

func TestInsertOrderedStreamMessage_GapIsHeldAndRepublished(t *testing.T) {
	repo := newFakeStreamsRepo()
	mq := &fakeMessageQueue{}
	c := newTestController(repo, mq)
	tenantId := uuid.New()

	// seq=1 arrives before its predecessor (seq=0) -- should not be applied
	msg := &tasktypes.StreamMessagePayload{
		Topic: "t", ProducerID: "p1", ProducerSeq: 1, CreatedAt: time.Now(),
	}
	applied, err := c.insertOrderedStreamMessage(context.Background(), tenantId, msg)
	requireApplied(t, false, applied, err)

	assert.Empty(t, repo.ordered, "the out-of-order message must not be inserted")
	assert.Empty(t, repo.forced, "the gap has not persisted long enough to force-insert")
	assert.Equal(t, 1, mq.count(), "the held-back message must be re-published for another attempt")
	assert.False(t, mq.sent[0].ImmediatelyExpire, "an expiring requeue is dead-lettered through the DLQ backoff under a backlog")
}

func TestInsertOrderedStreamMessage_StaleDuplicateIsDroppedNotRepublished(t *testing.T) {
	repo := newFakeStreamsRepo()
	mq := &fakeMessageQueue{}
	c := newTestController(repo, mq)
	tenantId := uuid.New()

	first := &tasktypes.StreamMessagePayload{Topic: "t", ProducerID: "p1", ProducerSeq: 0, CreatedAt: time.Now()}
	applied, err := c.insertOrderedStreamMessage(context.Background(), tenantId, first)
	requireApplied(t, true, applied, err)
	require.Len(t, repo.ordered, 1)

	// a redelivery of the same, already-applied message
	applied, err = c.insertOrderedStreamMessage(context.Background(), tenantId, first)
	requireApplied(t, false, applied, err)

	assert.Len(t, repo.ordered, 1, "the duplicate must not be inserted a second time")
	assert.Equal(t, 0, mq.count(), "a stale duplicate must be dropped, not retried forever")
}

func TestInsertOrderedStreamMessage_FirstAttemptIsNeverForceInsertedRegardlessOfAge(t *testing.T) {
	repo := newFakeStreamsRepo()
	mq := &fakeMessageQueue{}
	c := newTestController(repo, mq)
	tenantId := uuid.New()

	// seq=1 with no predecessor, published a long time ago (e.g. drained from
	// a queue backlog) -- but this is the controller's first time seeing it as
	// a gap, so it must be given a chance to resolve via retry, not
	// force-inserted on the spot. CreatedAt age must play no part in this
	// decision (see GapFirstDetectedAt on StreamMessagePayload).
	msg := &tasktypes.StreamMessagePayload{
		Topic: "t", ProducerID: "p1", ProducerSeq: 1, CreatedAt: time.Now().Add(-maxProducerGapWait - time.Hour),
	}
	applied, err := c.insertOrderedStreamMessage(context.Background(), tenantId, msg)
	requireApplied(t, false, applied, err)

	assert.Empty(t, repo.forced, "a first-seen gap must be requeued, never force-inserted immediately")
	assert.Equal(t, 1, mq.count())
}

func TestInsertOrderedStreamMessage_GivesUpOnceGapFirstDetectedAtExceedsMaxWait(t *testing.T) {
	repo := newFakeStreamsRepo()
	mq := &fakeMessageQueue{}
	c := newTestController(repo, mq)
	tenantId := uuid.New()

	// this is what a real requeued message looks like on its Nth retry: the
	// controller itself stamped GapFirstDetectedAt on the first attempt, and
	// it has now persisted past maxProducerGapWait
	msg := &tasktypes.StreamMessagePayload{
		Topic: "t", ProducerID: "p1", ProducerSeq: 1, CreatedAt: time.Now(),
		GapFirstDetectedAt: time.Now().Add(-maxProducerGapWait - time.Second),
	}
	applied, err := c.insertOrderedStreamMessage(context.Background(), tenantId, msg)
	requireApplied(t, true, applied, err)

	assert.Empty(t, repo.ordered, "a forced insert does not go through the ordered path")
	require.Len(t, repo.forced, 1, "the gap has persisted too long and must be force-inserted")
	assert.Equal(t, int64(1), repo.forced[0].ProducerSeq)
	assert.Equal(t, 0, mq.count(), "giving up must not also re-publish the message")
}

// Under a queue backlog every message is old (by CreatedAt) on arrival; a
// first-seen gap must still be held for a retry rather than force-inserted
// just because it happens to be stale, or the force-insert jumps the
// watermark and the late predecessor is then dropped as a duplicate.
func TestInsertOrderedStreamMessage_BacklogGapIsHeldOnFirstAttempt(t *testing.T) {
	repo := newFakeStreamsRepo()
	mq := &fakeMessageQueue{}
	c := newTestController(repo, mq)
	tenantId := uuid.New()
	published := time.Now().Add(-maxProducerGapWait - time.Minute)

	msg := func(seq int64) *tasktypes.StreamMessagePayload {
		return &tasktypes.StreamMessagePayload{Topic: "t", ProducerID: "p1", ProducerSeq: seq, CreatedAt: published}
	}

	applied, err := c.insertOrderedStreamMessage(context.Background(), tenantId, msg(0))
	requireApplied(t, true, applied, err)

	applied, err = c.insertOrderedStreamMessage(context.Background(), tenantId, msg(2))
	requireApplied(t, false, applied, err)

	assert.Empty(t, repo.forced, "a first-seen gap is held regardless of how old the message's CreatedAt is")
	assert.Equal(t, 1, mq.count(), "the early message must be requeued")

	applied, err = c.insertOrderedStreamMessage(context.Background(), tenantId, msg(1))
	requireApplied(t, true, applied, err)
}

// Regression test for a real reordering/data-loss bug: a producer that pauses
// for longer than maxProducerGapWait (normal for e.g. an agent between LLM
// calls) then publishes two messages back to back, handled out of order by
// two concurrent controller flushes. The later one must be held as a gap, not
// force-inserted immediately just because the producer's watermark happens to
// be old from the pause -- a premature force-insert would jump the watermark
// past the earlier message, which then gets dropped as a stale duplicate on
// arrival.
func TestInsertOrderedStreamMessage_ProducerPauseDoesNotCausePrematureForceInsertOrLoss(t *testing.T) {
	repo := newFakeStreamsRepo()
	mq := &fakeMessageQueue{}
	c := newTestController(repo, mq)
	tenantId := uuid.New()

	first := &tasktypes.StreamMessagePayload{Topic: "t", ProducerID: "p1", ProducerSeq: 0, CreatedAt: time.Now()}
	applied, err := c.insertOrderedStreamMessage(context.Background(), tenantId, first)
	requireApplied(t, true, applied, err)

	// the producer now pauses for longer than maxProducerGapWait; nothing
	// else touches this cursor in the meantime (no watermark-staleness
	// bookkeeping exists to falsely trip here)

	// N+2 is processed before N+1 by a concurrent flush
	nPlus2 := &tasktypes.StreamMessagePayload{Topic: "t", ProducerID: "p1", ProducerSeq: 2, CreatedAt: time.Now()}
	applied, err = c.insertOrderedStreamMessage(context.Background(), tenantId, nPlus2)
	requireApplied(t, false, applied, err)

	require.Empty(t, repo.forced, "must not force-insert ahead of a message that hasn't even had one retry yet")
	require.Equal(t, 1, mq.count(), "N+2 must be held and requeued")

	// N+1 now arrives and must apply normally, not get dropped as stale
	nPlus1 := &tasktypes.StreamMessagePayload{Topic: "t", ProducerID: "p1", ProducerSeq: 1, CreatedAt: time.Now()}
	applied, err = c.insertOrderedStreamMessage(context.Background(), tenantId, nPlus1)
	requireApplied(t, true, applied, err)

	// N+2's retry (carrying the GapFirstDetectedAt the controller stamped on
	// its first attempt) now closes cleanly via the ordinary CAS path
	retried := mq.sent[0]
	require.Len(t, retried.Payloads, 1)

	var retriedPayload tasktypes.StreamMessagePayload
	require.NoError(t, json.Unmarshal(retried.Payloads[0], &retriedPayload))
	require.False(t, retriedPayload.GapFirstDetectedAt.IsZero(), "the requeued message must carry when its gap was first detected")

	applied, err = c.insertOrderedStreamMessage(context.Background(), tenantId, &retriedPayload)
	requireApplied(t, true, applied, err)

	require.Empty(t, repo.forced, "the gap closed on retry; it must never have been force-inserted")
}

func TestHandleStreamMessages_InsertsEveryMessageInOneBatch(t *testing.T) {
	repo := newFakeStreamsRepo()
	c := newTestController(repo, &fakeMessageQueue{})
	tenantId := uuid.New()

	// a single MQSubBuffer flush can mix messages from different topics and
	// producers for the same tenant; each still needs its own compare-and-swap.
	msgs := []*tasktypes.StreamMessagePayload{
		{Topic: "t1", ProducerID: "p1", ProducerSeq: 0, Payload: []byte("p1-0"), CreatedAt: time.Now()},
		{Topic: "t2", ProducerID: "p2", ProducerSeq: 0, Payload: []byte("p2-0"), CreatedAt: time.Now()},
	}

	payloads := make([][]byte, 0, len(msgs))
	for _, m := range msgs {
		b, err := msgqueue.NewTenantMessage(tenantId, msgqueue.MsgIDStreamMessage, true, true, *m)
		require.NoError(t, err)
		payloads = append(payloads, b.Payloads...)
	}

	require.NoError(t, c.handleStreamMessages(context.Background(), tenantId, payloads))

	require.Len(t, repo.ordered, 2)
	assert.ElementsMatch(t, [][]byte{[]byte("p1-0"), []byte("p2-0")}, []([]byte){repo.ordered[0].Payload, repo.ordered[1].Payload})
}

func TestHandleStreamMessages_WakesEachWrittenTopicOnceAfterInsert(t *testing.T) {
	repo := newFakeStreamsRepo()
	pubsub := &fakePubSub{}
	c := newTestControllerWithPubSub(repo, &fakeMessageQueue{}, pubsub)
	tenantId := uuid.New()

	msgs := []*tasktypes.StreamMessagePayload{
		{Topic: "t1", ProducerID: "p1", ProducerSeq: 0, CreatedAt: time.Now()},
		{Topic: "t1", ProducerID: "p1", ProducerSeq: 1, CreatedAt: time.Now()},
		{Topic: "t2", ProducerID: "p2", ProducerSeq: 0, CreatedAt: time.Now()},
		// a gap on t3: held back and requeued, so nothing was written to wake for
		{Topic: "t3", ProducerID: "p3", ProducerSeq: 5, CreatedAt: time.Now()},
	}

	payloads := make([][]byte, 0, len(msgs))
	for _, m := range msgs {
		b, err := msgqueue.NewTenantMessage(tenantId, msgqueue.MsgIDStreamMessage, true, true, *m)
		require.NoError(t, err)
		payloads = append(payloads, b.Payloads...)
	}

	require.NoError(t, c.handleStreamMessages(context.Background(), tenantId, payloads))

	assert.ElementsMatch(t, []msgqueue.StreamWake{{Topic: "t1"}, {Topic: "t2"}}, pubsub.wakes(t))
	require.Len(t, pubsub.messages, 1, "one batch's wakes share a message")
	assert.Equal(t, tenantId, pubsub.messages[0].TenantID)
}

func TestWakeTopics_SplitsLargeBatchesUnderTheSizeBudget(t *testing.T) {
	pubsub := &fakePubSub{}
	c := newTestControllerWithPubSub(newFakeStreamsRepo(), &fakeMessageQueue{}, pubsub)

	wakes := make([]msgqueue.StreamWake, 100)
	for i := range wakes {
		wakes[i] = msgqueue.StreamWake{Namespace: "ns", Topic: fmt.Sprintf("%0250d", i)}
	}

	c.wakeTopics(context.Background(), uuid.New(), wakes)

	require.Greater(t, len(pubsub.messages), 1)

	for _, msg := range pubsub.messages {
		body, err := json.Marshal(msg)
		require.NoError(t, err)
		assert.Less(t, len(body), 8000, "each wake must fit in one pg_notify")
	}

	assert.ElementsMatch(t, wakes, pubsub.wakes(t))
}
