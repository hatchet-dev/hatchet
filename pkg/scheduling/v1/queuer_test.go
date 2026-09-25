//go:build !e2e && !load && !rampup && !integration

package v1

import (
	"context"
	"slices"
	"sync"
	"testing"
	"time"

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

// Ensure recordingQueueRepo still satisfies QueueRepository.
var _ v1repo.QueueRepository = (*recordingQueueRepo)(nil)

// blockingFlushQueueRepo serves a queue whose items leave it when assigned,
// blocks the first flush until released and records every flush.
type blockingFlushQueueRepo struct {
	fakeQueueRepository

	flushStarted chan struct{}
	flushRelease chan struct{}

	mu        sync.Mutex
	items     []*sqlcv1.V1QueueItem
	listCalls int
	flushes   []*v1repo.AssignResults
}

func (r *blockingFlushQueueRepo) ListQueueItems(context.Context, int) ([]*sqlcv1.V1QueueItem, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.listCalls++
	return append([]*sqlcv1.V1QueueItem(nil), r.items...), nil
}

func (r *blockingFlushQueueRepo) MarkQueueItemsProcessed(_ context.Context, res *v1repo.AssignResults) ([]*v1repo.AssignedItem, []*v1repo.AssignedItem, error) {
	r.mu.Lock()
	r.flushes = append(r.flushes, res)
	n := len(r.flushes)

	for _, assigned := range res.Assigned {
		r.items = slices.DeleteFunc(r.items, func(qi *sqlcv1.V1QueueItem) bool { return qi.ID == assigned.QueueItem.ID })
	}
	r.mu.Unlock()

	if n == 1 {
		close(r.flushStarted)
		<-r.flushRelease
	}

	return res.Assigned, nil, nil
}

func (r *blockingFlushQueueRepo) listCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.listCalls
}

func (r *blockingFlushQueueRepo) flushCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.flushes)
}

func (r *blockingFlushQueueRepo) flush(i int) *v1repo.AssignResults {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.flushes[i]
}

// A capacity wake that arrives while a batch's misses are still being flushed
// finds them unacked and assigns nothing; the batch must re-queue once the
// flush acks them instead of leaving them to the next poll.
func TestQueuer_CapacityRestoredDuringFlushRequeuesMisses(t *testing.T) {
	tenantId := uuid.New()
	workerId := uuid.New()

	s := newTestScheduler(t, tenantId, starvedActionRepo(map[uuid.UUID]map[string]int32{
		workerId: {v1repo.SlotTypeDefault: 1, v1repo.SlotTypeDurable: 10},
	}))
	s.setWorkers([]*v1repo.ListActiveWorkersResult{testWorker(workerId)})

	// drained default pool, idle durable pool: only the starved marker makes
	// a heuristic replenish rebuild this action
	w := &worker{ListActiveWorkersResult: testWorker(workerId)}
	used := newSlot(w, v1repo.SlotTypeDefault)
	used.used = true
	slots := []*slot{used}
	for i := 0; i < 10; i++ {
		slots = append(slots, newSlot(w, v1repo.SlotTypeDurable))
	}
	a := seedActionPools(t, s, "A", slots...)
	onLoop(t, s, func() {
		a.lastReplenishedSlotCount = 11
		a.lastReplenishedWorkerCount = 1
	})

	qi := testQI(tenantId, "A", 1)
	qr := &blockingFlushQueueRepo{
		items:        []*sqlcv1.V1QueueItem{qi},
		flushStarted: make(chan struct{}),
		flushRelease: make(chan struct{}),
	}

	l := zerolog.Nop()
	q := &Queuer{
		repo:          qr,
		tenantId:      tenantId,
		queueName:     qi.Queue,
		l:             &l,
		s:             s,
		limit:         100,
		resultsCh:     make(chan *QueueResults, 16),
		notifyQueueCh: make(chan map[string]string, 1),
		queueMu:       newMu(&l),
		unackedMu:     newRWMu(&l),
		unacked:       make(map[int64]struct{}),
		unassigned:    make(map[int64]*sqlcv1.V1QueueItem),
		unassignedMu:  newMu(&l),
	}
	tm := &tenantManager{queuers: []*Queuer{q}}
	s.onCapacityRestored = func(queues []string) { tm.notifyQueuers(context.Background(), queues) }

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go q.loopQueue(ctx)

	// first tick: the item misses and its flush blocks with the item unacked
	q.queue(ctx)
	select {
	case <-qr.flushStarted:
	case <-time.After(5 * time.Second):
		t.Fatal("the miss was never flushed")
	}
	require.Equal(t, 1, qr.listCount())

	// the rebuild restores the default slot and wakes the queue while the
	// flush is pending; that tick refills but filters out the unacked item
	require.NoError(t, s.replenish(ctx, false))
	require.Eventually(t, func() bool { return qr.listCount() == 2 }, 5*time.Second, time.Millisecond)
	require.Equal(t, 1, qr.flushCount(), "the early wake must not have assigned anything")

	// once the flush acks the miss, the batch re-queues and the retry assigns
	// the item well inside the 1 s poll interval
	released := time.Now()
	close(qr.flushRelease)
	require.Eventually(t, func() bool { return qr.flushCount() >= 2 }, 500*time.Millisecond, time.Millisecond,
		"the missed item was not retried after the flush acked it")
	require.Less(t, time.Since(released), 900*time.Millisecond)

	retry := qr.flush(1)
	require.Len(t, retry.Assigned, 1)
	require.Equal(t, qi.ID, retry.Assigned[0].QueueItem.ID)
	require.Equal(t, workerId, retry.Assigned[0].WorkerId)
	require.Empty(t, retry.Unassigned)
}
