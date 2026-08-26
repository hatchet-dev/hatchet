//go:build !e2e && !load && !rampup && !integration

package v1

import (
	"context"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	v1repo "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// recordingQueueRepo wraps fakeQueueRepository and records RequeueRateLimitedItems calls.
type recordingQueueRepo struct {
	fakeQueueRepository

	mu           sync.Mutex
	requeueCalls int
}

func (r *recordingQueueRepo) RequeueRateLimitedItems(context.Context, uuid.UUID, string) ([]*sqlcv1.RequeueRateLimitedQueueItemsRow, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.requeueCalls++
	return nil, nil
}

func (r *recordingQueueRepo) callCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.requeueCalls
}

// TestQueuerRequeuesRateLimitedItemsOnFreshProcess covers the deadlock where
// tasks fill a concurrency slot, get parked in v1_rate_limited_queue_items, then
// a new queuer (scheduler restart / new lease) starts with an empty live queue.
// It must still requeue the parked rows; otherwise they hold filled concurrency
// slots forever.
func TestQueuerRequeuesRateLimitedItemsOnFreshProcess(t *testing.T) {
	repo := &recordingQueueRepo{}
	l := zerolog.Nop()

	q := &Queuer{
		repo:      repo,
		tenantId:  uuid.New(),
		queueName: "default",
		l:         &l,
	}

	q.requeueRateLimitedItems(context.Background())

	require.Equal(t, 1, repo.callCount(),
		"a fresh queuer must requeue parked rate-limited items even when the live queue is empty")
}

func TestQueuerRequeuesRateLimitedItemsWhenAlreadyObserved(t *testing.T) {
	repo := &recordingQueueRepo{}
	l := zerolog.Nop()

	q := &Queuer{
		repo:      repo,
		tenantId:  uuid.New(),
		queueName: "default",
		l:         &l,
	}

	q.requeueRateLimitedItems(context.Background())

	require.Equal(t, 1, repo.callCount())
}

// Ensure recordingQueueRepo still satisfies QueueRepository.
var _ v1repo.QueueRepository = (*recordingQueueRepo)(nil)
