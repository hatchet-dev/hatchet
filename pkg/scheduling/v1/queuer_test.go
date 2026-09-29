//go:build !e2e && !load && !rampup && !integration

package v1

import (
	"context"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	v1repo "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlchelpers"
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
// flush acks them instead of leaving them to the next poll. This drives the
// slots path end to end: a heuristic replenish rebuilds the starved action,
// advances the capacity epoch and wakes the queue through the tenant manager;
// the re-queue comes from loopQueue asking isWorkerCapacityUpdated after the
// flush (TestQueuerReplaysWakeLostWhileMissedItemInFlight covers the same
// question for a forced replenish that sends no wake at all).
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
		unackedMu:     newRWMu(&l),
		unacked:       make(map[int64]struct{}),
		unassigned:    make(map[int64]*sqlcv1.V1QueueItem),
		unassignedMu:  newMu(&l),
	}
	tm := &tenantManager{queuers: []*Queuer{q}}
	s.onCapacityRestored = func(queues []string) { tm.notifyQueuers(context.Background(), queues) }

	// released below once the wake has been observed; on an early failure the
	// cleanup releases it so the held flush goroutine is not orphaned
	release := sync.OnceFunc(func() { close(qr.flushRelease) })
	t.Cleanup(release)

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
	release()
	require.Eventually(t, func() bool { return qr.flushCount() >= 2 }, 500*time.Millisecond, time.Millisecond,
		"the missed item was not retried after the flush acked it")
	require.Less(t, time.Since(released), 900*time.Millisecond)

	retry := qr.flush(1)
	require.Len(t, retry.Assigned, 1)
	require.Equal(t, qi.ID, retry.Assigned[0].QueueItem.ID)
	require.Equal(t, workerId, retry.Assigned[0].WorkerId)
	require.Empty(t, retry.Unassigned)
}

// lostWakeQueueRepo serves one queue item on every refill until it is assigned, holds the
// first flush open until the test releases it, and exposes the two points a test has to
// order itself against: a refill that came back empty (the item was in q.unacked) and the
// item's assignment.
type lostWakeQueueRepo struct {
	fakeQueueRepository

	item *sqlcv1.V1QueueItem

	refills  atomic.Int64
	flushes  atomic.Int64
	assigned atomic.Int64

	flushStarted chan struct{}
	releaseFlush chan struct{}
	emptyRefill  chan struct{}
}

func (r *lostWakeQueueRepo) ListQueueItems(context.Context, int) ([]*sqlcv1.V1QueueItem, error) {
	r.refills.Add(1)

	if r.assigned.Load() > 0 {
		return nil, nil
	}

	return []*sqlcv1.V1QueueItem{r.item}, nil
}

// GetTaskRateLimits runs right after refillQueue with the items it returned, so an empty
// slice means the tick found nothing to assign.
func (r *lostWakeQueueRepo) GetTaskRateLimits(_ context.Context, _ *v1repo.OptimisticTx, qis []*sqlcv1.V1QueueItem) (map[int64]map[string]int32, error) {
	if len(qis) == 0 {
		select {
		case r.emptyRefill <- struct{}{}:
		default:
		}
	}

	return nil, nil
}

func (r *lostWakeQueueRepo) MarkQueueItemsProcessed(_ context.Context, ar *v1repo.AssignResults) ([]*v1repo.AssignedItem, []*v1repo.AssignedItem, error) {
	if r.flushes.Add(1) == 1 {
		close(r.flushStarted)
		<-r.releaseFlush
	}

	r.assigned.Add(int64(len(ar.Assigned)))

	return ar.Assigned, nil, nil
}

func waitFor(t *testing.T, ch <-chan struct{}, what string) {
	t.Helper()

	select {
	case <-ch:
	case <-time.After(5 * time.Second):
		t.Fatal(what)
	}
}

// TestQueuerReplaysWakeLostWhileMissedItemInFlight covers the 50 ms mode of the
// low-latency harness at 10 slots: an item misses capacity and, while its flush is
// still running (the item sits in q.unacked, invisible to refillQueue), a replenish
// rebuilds the pools with a free slot. Whether the wake-up for that capacity was
// consumed by a tick that could not see the item, or was never sent (the periodic
// replenish does not notify), nothing would wake the queuer once the item came back.
// The post-flush path wakes it when the capacity epoch moved during the flight
// (isWorkerCapacityUpdated), and only then.
func TestQueuerReplaysWakeLostWhileMissedItemInFlight(t *testing.T) {
	for _, tc := range []struct {
		name             string
		rebuild          bool
		wakeDuringFlight bool
	}{
		{name: "pools rebuilt and the wake-up consumed during the flight: replayed", rebuild: true, wakeDuringFlight: true},
		{name: "pools rebuilt with no wake-up during the flight: replayed", rebuild: true},
		{name: "no rebuild: no extra tick", wakeDuringFlight: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tenantId := uuid.New()
			workerId := uuid.New()

			// a replenish finds one free default slot on the worker
			s := newTestScheduler(t, tenantId, &mockAssignmentRepo{
				listActionsForWorkersFn: func(context.Context, uuid.UUID, []uuid.UUID) ([]*sqlcv1.ListActionsForWorkersRow, error) {
					return []*sqlcv1.ListActionsForWorkersRow{{WorkerId: workerId, ActionId: sqlchelpers.TextFromStr("A")}}, nil
				},
				listAvailableSlotsForWorkersFn: func(context.Context, uuid.UUID, sqlcv1.ListAvailableSlotsForWorkersParams) ([]*sqlcv1.ListAvailableSlotsForWorkersRow, error) {
					return []*sqlcv1.ListAvailableSlotsForWorkersRow{{ID: workerId, AvailableSlots: 1}}, nil
				},
			})
			s.setWorkers([]*v1repo.ListActiveWorkersResult{testWorker(workerId)})

			// until then the worker's only default slot is in use, so the item misses
			w := &worker{ListActiveWorkersResult: testWorker(workerId)}
			used := newSlot(w, v1repo.SlotTypeDefault)
			used.used = true
			seedActionPools(t, s, "A", used)

			repo := &lostWakeQueueRepo{
				item:         testQI(tenantId, "A", 1),
				flushStarted: make(chan struct{}),
				releaseFlush: make(chan struct{}),
				emptyRefill:  make(chan struct{}, 16),
			}

			l := zerolog.Nop()
			q := &Queuer{
				repo:          repo,
				tenantId:      tenantId,
				queueName:     "q",
				l:             &l,
				s:             s,
				limit:         100,
				resultsCh:     make(chan *QueueResults, 16),
				notifyQueueCh: make(chan map[string]string, 1),
				unackedMu:     newRWMu(&l),
				unacked:       make(map[int64]struct{}),
				unassigned:    make(map[int64]*sqlcv1.V1QueueItem),
				unassignedMu:  newMu(&l),
			}

			ctx, cancel := context.WithCancel(context.Background())
			loopDone := make(chan struct{})
			go func() {
				defer close(loopDone)
				q.loopQueue(ctx)
			}()

			var releaseOnce sync.Once
			release := func() { releaseOnce.Do(func() { close(repo.releaseFlush) }) }
			t.Cleanup(func() {
				release()
				cancel()
				<-loopDone
			})

			// tick 1: the item misses and its flush is held open
			q.queue(ctx)
			waitFor(t, repo.flushStarted, "the missed item's flush did not start")

			// capacity comes back while the item is in flight
			if tc.rebuild {
				require.NoError(t, s.replenish(ctx, true))
			}

			// the wake-up for the new capacity finds nothing: tick 2 refills while the
			// item is still unacked, so it is filtered out
			if tc.wakeDuringFlight {
				q.queue(ctx)
				waitFor(t, repo.emptyRefill, "tick 2 did not run an empty refill")
			}

			release()

			if tc.rebuild {
				// no other wake-up exists and the poll floor is 1 s, so an assignment
				// within 500 ms can only come from the replayed wake-up
				require.Eventually(t, func() bool { return repo.assigned.Load() == 1 }, 500*time.Millisecond, time.Millisecond,
					"the wake-up lost while the item was in flight must be replayed once it is back")
			} else {
				require.Eventually(t, func() bool {
					q.unassignedMu.Lock()
					defer q.unassignedMu.Unlock()
					_, ok := q.unassigned[repo.item.ID]
					return ok
				}, 5*time.Second, time.Millisecond, "the flush must return the item to q.unassigned")

				refills := repo.refills.Load()
				require.Never(t, func() bool { return repo.refills.Load() != refills }, 300*time.Millisecond, 10*time.Millisecond,
					"without a rebuild there is no capacity to retry against and nothing to replay")
				require.Zero(t, repo.assigned.Load())
			}
		})
	}
}

// TestQueuerQueueCoalescesIntoBufferedWake pins the rule the replay relies on: a wake-up is
// dropped only while another is buffered and not yet consumed, never while one is merely
// being delivered, and the caller never blocks.
func TestQueuerQueueCoalescesIntoBufferedWake(t *testing.T) {
	l := zerolog.Nop()
	q := &Queuer{l: &l, notifyQueueCh: make(chan map[string]string, 1)}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	q.queue(ctx)
	q.queue(ctx)
	require.Len(t, q.notifyQueueCh, 1, "a second wake-up coalesces into the buffered one")

	<-q.notifyQueueCh
	q.queue(ctx)
	require.Len(t, q.notifyQueueCh, 1, "once the buffered wake-up is consumed the next one is delivered, canceled context or not")
}
