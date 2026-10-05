package streams

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
)

// batchingStreamsRepository records each batch and stores every message whose
// seq is even, reporting odd ones as gaps.
type batchingStreamsRepository struct {
	v1.StreamsRepository

	mu      sync.Mutex
	batches [][]v1.TenantStreamMessage
	err     error
	// holds the first batch until released, so later publishes queue up
	firstBatch chan struct{}
}

func (f *batchingStreamsRepository) InsertOrderedStreamMessages(_ context.Context, msgs []v1.TenantStreamMessage) ([]v1.OrderedStreamMessageResult, error) {
	f.mu.Lock()
	first := len(f.batches) == 0
	f.batches = append(f.batches, msgs)
	f.mu.Unlock()

	if first && f.firstBatch != nil {
		<-f.firstBatch
	}

	if f.err != nil {
		return nil, f.err
	}

	results := make([]v1.OrderedStreamMessageResult, len(msgs))

	for i, m := range msgs {
		results[i] = v1.OrderedStreamMessageResult{Inserted: m.Opts.ProducerSeq%2 == 0, CurrentSeq: m.Opts.ProducerSeq - 2}
	}

	return results, nil
}

func (f *batchingStreamsRepository) batchSizes() []int {
	f.mu.Lock()
	defer f.mu.Unlock()

	sizes := make([]int, len(f.batches))
	for i, b := range f.batches {
		sizes[i] = len(b)
	}

	return sizes
}

func publishMsg(tenantId uuid.UUID, topic string, seq int64) v1.TenantStreamMessage {
	return v1.TenantStreamMessage{TenantID: tenantId, Opts: v1.CreateOrderedStreamMessageOpts{
		Topic: topic, Payload: []byte("m"), ProducerID: topic, ProducerSeq: seq,
	}}
}

func TestPublishBatcher_BatchesQueuedPublishesAndRoutesResults(t *testing.T) {
	repo := &batchingStreamsRepository{firstBatch: make(chan struct{})}
	release := sync.OnceFunc(func() { close(repo.firstBatch) })
	// one worker, so publishes queued behind its first commit must share the next
	b := newPublishBatcher(repo, fakePubSub{}, testLogger(), 1)
	defer b.stop()
	defer release()

	tenantId := uuid.New()
	const total = 40

	var wg sync.WaitGroup
	results := make([]v1.OrderedStreamMessageResult, total)

	wg.Go(func() {
		var err error
		results[0], err = b.publish(context.Background(), publishMsg(tenantId, "topic-0", 0))
		assert.NoError(t, err)
	})

	require.Eventually(t, func() bool { return len(repo.batchSizes()) == 1 }, time.Second, time.Millisecond)

	for i := 1; i < total; i++ {
		wg.Go(func() {
			var err error
			results[i], err = b.publish(context.Background(), publishMsg(tenantId, fmt.Sprintf("topic-%d", i), int64(i)))
			assert.NoError(t, err)
		})
	}

	require.Eventually(t, func() bool { return len(b.pending) == total-1 }, time.Second, time.Millisecond)
	release()
	wg.Wait()

	for i, res := range results {
		assert.Equal(t, i%2 == 0, res.Inserted, "publish %d got another publish's result", i)
	}

	assert.Equal(t, []int{1, total - 1}, repo.batchSizes())
}

func TestPublishBatcher_WakesOnlyStoredTopicsPerTenant(t *testing.T) {
	repo := &batchingStreamsRepository{}
	pubsub := &recordingPubSub{}
	b := newPublishBatcher(repo, pubsub, testLogger(), 1)
	defer b.stop()

	tenantId := uuid.New()

	b.commit([]*pendingPublish{
		{msg: publishMsg(tenantId, "stored", 0), done: make(chan publishResult, 1)},
		{msg: publishMsg(tenantId, "stored", 2), done: make(chan publishResult, 1)},
		{msg: publishMsg(tenantId, "gap", 1), done: make(chan publishResult, 1)},
	})

	require.Len(t, pubsub.messages, 1)
	assert.Equal(t, tenantId, pubsub.messages[0].TenantID)
	assert.Len(t, pubsub.messages[0].Payloads, 1, "one wake for the stored topic, none for the gap")
}

func TestPublishBatcher_FailedCommitFailsEveryPublishInIt(t *testing.T) {
	repo := &batchingStreamsRepository{err: errors.New("database unavailable")}
	b := newPublishBatcher(repo, fakePubSub{}, testLogger(), 1)
	defer b.stop()

	_, err := b.publish(context.Background(), publishMsg(uuid.New(), "t", 0))
	assert.ErrorContains(t, err, "database unavailable")
}

type recordingPubSub struct {
	fakePubSub

	mu       sync.Mutex
	messages []*msgqueue.Message
}

func (r *recordingPubSub) Pub(_ context.Context, _ msgqueue.Topic, msg *msgqueue.Message) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.messages = append(r.messages, msg)

	return nil
}
