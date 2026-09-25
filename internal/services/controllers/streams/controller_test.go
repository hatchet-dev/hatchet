package streams

import (
	"context"
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

func newTestController(repo *fakeStreamsRepo, mq *fakeMessageQueue) *ControllerImpl {
	l := zerolog.Nop()

	return &ControllerImpl{
		mq:   mq,
		repo: &fakeRepository{streams: repo},
		l:    &l,
		a:    hatcheterrors.NewWrapped(hatcheterrors.NoOpAlerter{}),
	}
}

func TestInsertOrderedStreamMessage_InOrderInsertsSequentially(t *testing.T) {
	repo := newFakeStreamsRepo()
	c := newTestController(repo, &fakeMessageQueue{})
	tenantId := uuid.New()

	for seq := int64(0); seq < 3; seq++ {
		msg := &tasktypes.StreamMessagePayload{
			Topic: "t", ProducerID: "p1", ProducerSeq: seq, CreatedAt: time.Now(),
		}
		require.NoError(t, c.insertOrderedStreamMessage(context.Background(), tenantId, msg))
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
	require.NoError(t, c.insertOrderedStreamMessage(context.Background(), tenantId, msg))

	assert.Empty(t, repo.ordered, "the out-of-order message must not be inserted")
	assert.Empty(t, repo.forced, "the gap has not persisted long enough to force-insert")
	assert.Equal(t, 1, mq.count(), "the held-back message must be re-published for another attempt")
}

func TestInsertOrderedStreamMessage_StaleDuplicateIsDroppedNotRepublished(t *testing.T) {
	repo := newFakeStreamsRepo()
	mq := &fakeMessageQueue{}
	c := newTestController(repo, mq)
	tenantId := uuid.New()

	first := &tasktypes.StreamMessagePayload{Topic: "t", ProducerID: "p1", ProducerSeq: 0, CreatedAt: time.Now()}
	require.NoError(t, c.insertOrderedStreamMessage(context.Background(), tenantId, first))
	require.Len(t, repo.ordered, 1)

	// a redelivery of the same, already-applied message
	require.NoError(t, c.insertOrderedStreamMessage(context.Background(), tenantId, first))

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
	require.NoError(t, c.insertOrderedStreamMessage(context.Background(), tenantId, msg))

	assert.Empty(t, repo.ordered, "a forced insert does not go through the ordered path")
	require.Len(t, repo.forced, 1, "the gap has persisted too long and must be force-inserted")
	assert.Equal(t, int64(1), repo.forced[0].ProducerSeq)
	assert.Equal(t, 0, mq.count(), "giving up must not also re-publish the message")
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
