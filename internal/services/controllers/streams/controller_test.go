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
	// when each cursor last advanced
	advancedAt map[string]time.Time
	ordered    []v1.CreateOrderedStreamMessageOpts
	forced     []v1.CreateOrderedStreamMessageOpts
}

func newFakeStreamsRepo() *fakeStreamsRepo {
	return &fakeStreamsRepo{cursors: make(map[string]int64), advancedAt: make(map[string]time.Time)}
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
		return v1.OrderedStreamMessageResult{Inserted: false, CurrentSeq: current, CurrentSeqAdvancedAt: f.advancedAt[k]}, nil
	}

	f.cursors[k] = opts.ProducerSeq
	f.advancedAt[k] = time.Now()
	f.ordered = append(f.ordered, opts)

	return v1.OrderedStreamMessageResult{Inserted: true, CurrentSeq: opts.ProducerSeq, CurrentSeqAdvancedAt: f.advancedAt[k]}, nil
}

func (f *fakeStreamsRepo) ForceInsertOrderedStreamMessage(_ context.Context, tenantId uuid.UUID, opts v1.CreateOrderedStreamMessageOpts) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	k := cursorKey(tenantId, opts)

	if current, ok := f.cursors[k]; !ok || opts.ProducerSeq > current {
		f.cursors[k] = opts.ProducerSeq
		f.advancedAt[k] = time.Now()
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

func TestInsertOrderedStreamMessage_GivesUpAfterMaxWaitAndForceInserts(t *testing.T) {
	repo := newFakeStreamsRepo()
	mq := &fakeMessageQueue{}
	c := newTestController(repo, mq)
	tenantId := uuid.New()

	// seq=1 with no predecessor, but old enough to exceed maxProducerGapWait
	msg := &tasktypes.StreamMessagePayload{
		Topic: "t", ProducerID: "p1", ProducerSeq: 1, CreatedAt: time.Now().Add(-maxProducerGapWait - time.Second),
	}
	applied, err := c.insertOrderedStreamMessage(context.Background(), tenantId, msg)
	requireApplied(t, true, applied, err)

	assert.Empty(t, repo.ordered, "a forced insert does not go through the ordered path")
	require.Len(t, repo.forced, 1, "the gap has persisted too long and must be force-inserted")
	assert.Equal(t, int64(1), repo.forced[0].ProducerSeq)
	assert.Equal(t, 0, mq.count(), "giving up must not also re-publish the message")
}

// Under a queue backlog every message is old on arrival; a gap must still be
// waited out while the watermark keeps advancing, or the force insert jumps
// the watermark and the late predecessor is then dropped as a duplicate.
func TestInsertOrderedStreamMessage_BacklogGapIsHeldWhileWatermarkAdvances(t *testing.T) {
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

	assert.Empty(t, repo.forced, "the watermark just advanced, so the gap isn't stalled")
	assert.Equal(t, 1, mq.count(), "the early message must be requeued")

	applied, err = c.insertOrderedStreamMessage(context.Background(), tenantId, msg(1))
	requireApplied(t, true, applied, err)
}

func TestInsertOrderedStreamMessage_ForceInsertsOnceWatermarkStalls(t *testing.T) {
	repo := newFakeStreamsRepo()
	mq := &fakeMessageQueue{}
	c := newTestController(repo, mq)
	tenantId := uuid.New()

	first := &tasktypes.StreamMessagePayload{Topic: "t", ProducerID: "p1", ProducerSeq: 0, CreatedAt: time.Now()}
	applied, err := c.insertOrderedStreamMessage(context.Background(), tenantId, first)
	requireApplied(t, true, applied, err)

	// seq 1 never arrives
	repo.advancedAt[cursorKey(tenantId, v1.CreateOrderedStreamMessageOpts{Topic: "t", ProducerID: "p1"})] = time.Now().Add(-maxProducerGapWait - time.Second)

	applied, err = c.insertOrderedStreamMessage(context.Background(), tenantId, &tasktypes.StreamMessagePayload{Topic: "t", ProducerID: "p1", ProducerSeq: 2, CreatedAt: time.Now()})
	requireApplied(t, true, applied, err)

	require.Len(t, repo.forced, 1)
	assert.Equal(t, 0, mq.count())
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
