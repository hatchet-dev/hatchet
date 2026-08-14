//go:build !e2e && !load && !rampup && !integration

package v1

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	repo "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

type mockAssignmentRepo struct {
	listActionsForWorkersFn                func(ctx context.Context, tenantId uuid.UUID, workerIds []uuid.UUID) ([]*sqlcv1.ListActionsForWorkersRow, error)
	listAvailableSlotsForWorkersFn         func(ctx context.Context, tenantId uuid.UUID, params sqlcv1.ListAvailableSlotsForWorkersParams) ([]*sqlcv1.ListAvailableSlotsForWorkersRow, error)
	listAvailableSlotsForWorkersAndTypesFn func(ctx context.Context, tenantId uuid.UUID, params sqlcv1.ListAvailableSlotsForWorkersAndTypesParams) ([]*sqlcv1.ListAvailableSlotsForWorkersAndTypesRow, error)
	listWorkerSlotConfigsFn                func(ctx context.Context, tenantId uuid.UUID, workerIds []uuid.UUID) ([]*sqlcv1.ListWorkerSlotConfigsRow, error)
}

func (m *mockAssignmentRepo) ListActionsForWorkers(ctx context.Context, tenantId uuid.UUID, workerIds []uuid.UUID) ([]*sqlcv1.ListActionsForWorkersRow, error) {
	if m.listActionsForWorkersFn == nil {
		return nil, fmt.Errorf("ListActionsForWorkers not configured")
	}

	return m.listActionsForWorkersFn(ctx, tenantId, workerIds)
}

func (m *mockAssignmentRepo) ListAvailableSlotsForWorkers(ctx context.Context, tenantId uuid.UUID, params sqlcv1.ListAvailableSlotsForWorkersParams) ([]*sqlcv1.ListAvailableSlotsForWorkersRow, error) {
	if m.listAvailableSlotsForWorkersFn == nil {
		return nil, fmt.Errorf("ListAvailableSlotsForWorkers not configured")
	}

	return m.listAvailableSlotsForWorkersFn(ctx, tenantId, params)
}

func (m *mockAssignmentRepo) ListAvailableSlotsForWorkersAndTypes(ctx context.Context, tenantId uuid.UUID, params sqlcv1.ListAvailableSlotsForWorkersAndTypesParams) ([]*sqlcv1.ListAvailableSlotsForWorkersAndTypesRow, error) {
	if m.listAvailableSlotsForWorkersAndTypesFn != nil {
		return m.listAvailableSlotsForWorkersAndTypesFn(ctx, tenantId, params)
	}

	// Backwards-compat fallback: emulate the multi-type query by calling the per-type query.
	if m.listAvailableSlotsForWorkersFn != nil {
		out := make([]*sqlcv1.ListAvailableSlotsForWorkersAndTypesRow, 0)

		for _, slotType := range params.Slottypes {
			rows, err := m.listAvailableSlotsForWorkersFn(ctx, tenantId, sqlcv1.ListAvailableSlotsForWorkersParams{
				Tenantid:  params.Tenantid,
				Workerids: params.Workerids,
				Slottype:  slotType,
			})
			if err != nil {
				return nil, err
			}

			for _, row := range rows {
				out = append(out, &sqlcv1.ListAvailableSlotsForWorkersAndTypesRow{
					ID:             row.ID,
					SlotType:       slotType,
					AvailableSlots: row.AvailableSlots,
				})
			}
		}

		return out, nil
	}

	return nil, fmt.Errorf("ListAvailableSlotsForWorkersAndTypes not configured")
}

func (m *mockAssignmentRepo) ListWorkerSlotConfigs(ctx context.Context, tenantId uuid.UUID, workerIds []uuid.UUID) ([]*sqlcv1.ListWorkerSlotConfigsRow, error) {
	if m.listWorkerSlotConfigsFn == nil {
		// Default: all workers have the default slot type.
		out := make([]*sqlcv1.ListWorkerSlotConfigsRow, 0, len(workerIds))
		for _, wid := range workerIds {
			out = append(out, &sqlcv1.ListWorkerSlotConfigsRow{
				WorkerID: wid,
				SlotType: repo.SlotTypeDefault,
				MaxUnits: 0,
			})
		}
		return out, nil
	}

	return m.listWorkerSlotConfigsFn(ctx, tenantId, workerIds)
}

type mockSchedulerRepo struct {
	assignment repo.AssignmentRepository
}

func (m *mockSchedulerRepo) BatchQueue() repo.BatchQueueFactoryRepository {
	//TODO implement me
	panic("implement me")
}

func (m *mockSchedulerRepo) Concurrency() repo.ConcurrencyRepository {
	panic("unexpected call: Concurrency")
}

func (m *mockSchedulerRepo) Lease() repo.LeaseRepository {
	panic("unexpected call: Lease")
}

func (m *mockSchedulerRepo) QueueFactory() repo.QueueFactoryRepository {
	panic("unexpected call: QueueFactory")
}

func (m *mockSchedulerRepo) RateLimit() repo.RateLimitRepository {
	panic("unexpected call: RateLimit")
}

func (m *mockSchedulerRepo) Assignment() repo.AssignmentRepository {
	if m.assignment == nil {
		panic("mockSchedulerRepo.assignment is nil")
	}
	return m.assignment
}

func (m *mockSchedulerRepo) Optimistic() repo.OptimisticSchedulingRepository {
	panic("unexpected call: Optimistic")
}

// newTestScheduler builds a scheduler with its run loop started; the replenish
// and snapshot loops are not started, so tests drive those directly.
func newTestScheduler(t *testing.T, tenantId uuid.UUID, ar repo.AssignmentRepository) *Scheduler {
	t.Helper()

	l := zerolog.Nop()

	sr := &mockSchedulerRepo{assignment: ar}
	cf := &sharedConfig{
		repo: sr,
		l:    &l,
	}

	// rate limiter not needed for most tests; can be set by the caller if required.
	s := newScheduler(cf, tenantId, nil, &Extensions{})

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go s.run(ctx)

	return s
}

// onLoop runs fn on the scheduler's run loop, which is where all internal state
// must be read and written from.
func onLoop(t *testing.T, s *Scheduler, fn func()) {
	t.Helper()
	require.True(t, s.do(context.Background(), fn))
}

func testWorker(id uuid.UUID) *repo.ListActiveWorkersResult {
	return &repo.ListActiveWorkersResult{
		ID:     id,
		Name:   "w",
		Labels: nil,
	}
}

// seedActionPools installs the given slots into their (worker, slot type) pools
// and indexes the action onto the slots' workers. Slots pre-marked used stay
// off the freelist.
func seedActionPools(t *testing.T, s *Scheduler, actionId string, slots ...*slot) *action {
	t.Helper()

	var a *action

	onLoop(t, s, func() {
		workerSet := make(map[uuid.UUID]struct{})
		workerIds := make([]uuid.UUID, 0)
		byKey := make(map[poolKey][]*slot)

		for _, sl := range slots {
			workerId := sl.getWorkerId()
			if _, ok := workerSet[workerId]; !ok {
				workerSet[workerId] = struct{}{}
				workerIds = append(workerIds, workerId)
			}

			byKey[poolKey{workerId: workerId, slotType: sl.slotType}] = append(byKey[poolKey{workerId: workerId, slotType: sl.slotType}], sl)
		}

		for key, keySlots := range byKey {
			pool := s.pools[key]
			if pool == nil {
				pool = &slotPool{worker: keySlots[0].worker, slotType: key.slotType}
				s.pools[key] = pool
				if s.poolsByWorker[key.workerId] == nil {
					s.poolsByWorker[key.workerId] = make(map[string]*slotPool)
				}
				s.poolsByWorker[key.workerId][key.slotType] = pool
			}

			merged := pool.slots
			for _, sl := range keySlots {
				seen := false
				for _, existing := range pool.slots {
					if existing == sl {
						seen = true
						break
					}
				}
				if !seen {
					merged = append(merged, sl)
				}
			}

			pool.reset(merged, time.Now().Add(defaultSlotExpiry))
		}

		a = &action{workerIds: workerIds}
		s.actions[actionId] = a
	})

	return a
}

func slotsForAction(t *testing.T, s *Scheduler, a *action) []*slot {
	t.Helper()

	var slots []*slot
	onLoop(t, s, func() {
		for _, workerId := range a.workerIds {
			for _, pool := range s.poolsByWorker[workerId] {
				slots = append(slots, pool.slots...)
			}
		}
	})
	return slots
}

// takeSlot reserves a specific slot from its pool the way assignment does,
// releasing any other slots popped while looking for it.
func takeSlot(t *testing.T, s *Scheduler, sl *slot) {
	t.Helper()

	onLoop(t, s, func() {
		var skipped []*slot
		for {
			taken := sl.pool.take()
			require.NotNil(t, taken, "slot not on the freelist")
			if taken == sl {
				break
			}
			skipped = append(skipped, taken)
		}
		for _, other := range skipped {
			sl.pool.release(other)
		}
	})
}

func testQI(tenantId uuid.UUID, actionId string, taskId int64) *sqlcv1.V1QueueItem {
	return &sqlcv1.V1QueueItem{
		ID:         taskId,
		TenantID:   tenantId,
		ActionID:   actionId,
		TaskID:     taskId,
		Queue:      "q",
		StepID:     uuid.New(),
		ExternalID: uuid.New(),
	}
}

func ts(tm time.Time) pgtype.Timestamp {
	return sqlchelpers.TimestampFromTime(tm)
}

func defaultRequest() map[string]int32 {
	return map[string]int32{repo.SlotTypeDefault: 1}
}

// assignOne runs a single assignment on the run loop, the way handleAssignBatch
// does per item.
func assignOne(t *testing.T, s *Scheduler, a *action, qi *sqlcv1.V1QueueItem, labels []*sqlcv1.GetDesiredLabelsRow, requests map[string]int32, rlAck, rlNack func()) *assignSingleResult {
	t.Helper()

	if rlAck == nil {
		rlAck = func() {}
	}
	if rlNack == nil {
		rlNack = func() {}
	}

	r := &assignSingleResult{qi: qi}
	onLoop(t, s, func() {
		s.assignSingleton(a, qi, r, labels, requests, rlAck, rlNack, time.Now())
	})
	return r
}

func TestScheduler_AckNack(t *testing.T) {
	tenantId := uuid.New()
	workerId := uuid.New()

	s := newTestScheduler(t, tenantId, &mockAssignmentRepo{})
	s.setWorkers([]*repo.ListActiveWorkersResult{testWorker(workerId)})

	w := &worker{ListActiveWorkersResult: testWorker(workerId)}
	sl := newSlot(w, repo.SlotTypeDefault)
	sl2 := newSlot(w, repo.SlotTypeDefault)
	seedActionPools(t, s, "A", sl, sl2)

	takeSlot(t, s, sl)
	onLoop(t, s, func() {
		s.unackedSlots[123] = &assignedSlots{slots: []*slot{sl}}
	})

	s.ack([]int{123, 999})

	onLoop(t, s, func() {
		require.Empty(t, s.unackedSlots)
		// acked slots stay used until the next replenish observes the flush
		require.True(t, sl.used)
	})

	// nack should return the slot to the freelist and remove from unacked
	takeSlot(t, s, sl2)
	onLoop(t, s, func() {
		s.unackedSlots[777] = &assignedSlots{slots: []*slot{sl2}}
	})

	s.nack([]int{777})

	onLoop(t, s, func() {
		require.Empty(t, s.unackedSlots)
		require.False(t, sl2.used)
		require.Equal(t, 1, len(sl2.pool.free))
	})
}

func TestScheduler_SetWorkers(t *testing.T) {
	tenantId := uuid.New()

	s := newTestScheduler(t, tenantId, &mockAssignmentRepo{})

	w1 := testWorker(uuid.New())
	w2 := testWorker(uuid.New())

	s.setWorkers([]*repo.ListActiveWorkersResult{w1, w2})

	onLoop(t, s, func() {
		require.Len(t, s.workers, 2)
		require.Equal(t, w1.ID, s.workers[w1.ID].ID)
		require.Equal(t, w2.ID, s.workers[w2.ID].ID)
	})

	s.addWorker(testWorker(uuid.New()))

	onLoop(t, s, func() {
		require.Len(t, s.workers, 3)
	})
}

func TestScheduleRateLimitResult_ShouldRemoveFromQueue(t *testing.T) {
	// nil underlying result -> false
	r := &scheduleRateLimitResult{}
	require.False(t, r.shouldRemoveFromQueue())

	// nextRefillAt far enough in future -> true
	future := time.Now().UTC().Add(rateLimitedRequeueAfterThreshold + 250*time.Millisecond)
	r.rateLimitResult = &rateLimitResult{nextRefillAt: &future}
	require.True(t, r.shouldRemoveFromQueue())

	// nextRefillAt close -> false
	near := time.Now().UTC().Add(rateLimitedRequeueAfterThreshold - 250*time.Millisecond)
	r.rateLimitResult = &rateLimitResult{nextRefillAt: &near}
	require.False(t, r.shouldRemoveFromQueue())
}

func TestSelectSlotsFromPools_SkipsUsedAndStale(t *testing.T) {
	workerId := uuid.New()
	w := &worker{ListActiveWorkersResult: testWorker(workerId)}

	used := newSlot(w, repo.SlotTypeDefault)
	used.used = true
	free := newSlot(w, repo.SlotTypeDefault)

	pool := &slotPool{worker: w, slotType: repo.SlotTypeDefault}
	pool.reset([]*slot{used, free}, time.Now().Add(defaultSlotExpiry))

	now := time.Now()
	require.Equal(t, 1, pool.freeCountAt(now), "used slots must not count as free")

	selected, ok := selectSlotsFromPools(
		map[string]*slotPool{repo.SlotTypeDefault: pool},
		defaultRequest(),
		now,
	)
	require.True(t, ok)
	require.Len(t, selected, 1)
	require.Same(t, free, selected[0])

	// the pool is exhausted now
	_, ok = selectSlotsFromPools(map[string]*slotPool{repo.SlotTypeDefault: pool}, defaultRequest(), now)
	require.False(t, ok)

	// a stale pool has no assignable capacity even with free slots
	stalePool := &slotPool{worker: w, slotType: repo.SlotTypeDefault}
	stalePool.reset([]*slot{newSlot(w, repo.SlotTypeDefault)}, time.Now().Add(-time.Second))

	require.Equal(t, 0, stalePool.freeCountAt(now))
	_, ok = selectSlotsFromPools(map[string]*slotPool{repo.SlotTypeDefault: stalePool}, defaultRequest(), now)
	require.False(t, ok)
}

func TestScheduler_AssignSingleton_RingWraparound(t *testing.T) {
	tenantId := uuid.New()
	workerId1 := uuid.New()
	workerId2 := uuid.New()

	s := newTestScheduler(t, tenantId, &mockAssignmentRepo{})

	w1 := &worker{ListActiveWorkersResult: testWorker(workerId1)}
	w2 := &worker{ListActiveWorkersResult: testWorker(workerId2)}

	// s1 is used/inactive, s2 is active
	s1 := newSlot(w1, repo.SlotTypeDefault)
	s1.used = true
	s2 := newSlot(w2, repo.SlotTypeDefault)

	a := seedActionPools(t, s, "A", s1, s2)
	onLoop(t, s, func() { a.ringOffset = 1 })

	qi := testQI(tenantId, "A", 1)
	res := assignOne(t, s, a, qi, nil, defaultRequest(), nil, nil)
	require.True(t, res.succeeded)
	require.False(t, res.noSlots)
	require.Equal(t, workerId2, res.workerId)
	require.NotZero(t, res.ackId)

	onLoop(t, s, func() {
		_, ok := s.unackedSlots[res.ackId]
		require.True(t, ok)
		require.Equal(t, 2, a.ringOffset, "assignment advances the ring")
	})
}

func TestScheduler_AssignSingleton_NoSlots(t *testing.T) {
	tenantId := uuid.New()
	workerId := uuid.New()

	s := newTestScheduler(t, tenantId, &mockAssignmentRepo{})
	w := &worker{ListActiveWorkersResult: testWorker(workerId)}

	s1 := newSlot(w, repo.SlotTypeDefault)
	s1.used = true

	a := seedActionPools(t, s, "A", s1)

	qi := testQI(tenantId, "A", 1)
	res := assignOne(t, s, a, qi, nil, defaultRequest(), nil, nil)
	require.False(t, res.succeeded)
	require.True(t, res.noSlots)
}

func TestScheduler_AssignSingleton_StickyHardForcesRanking(t *testing.T) {
	tenantId := uuid.New()
	desiredWorkerId := uuid.New()
	otherWorkerId := uuid.New()

	s := newTestScheduler(t, tenantId, &mockAssignmentRepo{})

	wDesired := &worker{ListActiveWorkersResult: testWorker(desiredWorkerId)}
	wOther := &worker{ListActiveWorkersResult: testWorker(otherWorkerId)}

	// Put desired slot second and advance the ring; with HARD sticky it should
	// still be selected.
	otherSlot := newSlot(wOther, repo.SlotTypeDefault)
	desiredSlot := newSlot(wDesired, repo.SlotTypeDefault)

	a := seedActionPools(t, s, "A", otherSlot, desiredSlot)
	onLoop(t, s, func() { a.ringOffset = 1 })

	qi := testQI(tenantId, "A", 1)
	qi.Sticky = sqlcv1.V1StickyStrategyHARD
	qi.DesiredWorkerID = &desiredWorkerId

	res := assignOne(t, s, a, qi, nil, defaultRequest(), nil, nil)
	require.True(t, res.succeeded)
	require.Equal(t, desiredWorkerId, res.workerId)
}

func TestScheduler_AssignSingleton_EqualLabelsRoundRobin(t *testing.T) {
	tenantId := uuid.New()
	workerIds := []uuid.UUID{uuid.New(), uuid.New()}

	regionLabel := &sqlcv1.ListManyWorkerLabelsRow{
		Key:      "region",
		StrValue: sqlchelpers.TextFromStr("us-east-1"),
	}
	desired := []*sqlcv1.GetDesiredLabelsRow{{
		Key:        "region",
		StrValue:   sqlchelpers.TextFromStr("us-east-1"),
		Comparator: sqlcv1.WorkerLabelComparatorEQUAL,
		Weight:     10,
	}}

	workers := make([]*repo.ListActiveWorkersResult, len(workerIds))
	slots := make([]*slot, 0, len(workerIds)*2)

	for i, id := range workerIds {
		w := &repo.ListActiveWorkersResult{
			ID:     id,
			Name:   "w",
			Labels: []*sqlcv1.ListManyWorkerLabelsRow{regionLabel},
		}
		workers[i] = w
		ww := &worker{ListActiveWorkersResult: w}
		// Extra capacity so packing onto the first worker is not forced by slots.
		slots = append(slots, newSlot(ww, repo.SlotTypeDefault), newSlot(ww, repo.SlotTypeDefault))
	}

	s := newTestScheduler(t, tenantId, &mockAssignmentRepo{})
	s.setWorkers(workers)
	a := seedActionPools(t, s, "A", slots...)

	first := assignOne(t, s, a, testQI(tenantId, "A", 1), desired, defaultRequest(), nil, nil)
	require.True(t, first.succeeded)

	second := assignOne(t, s, a, testQI(tenantId, "A", 2), desired, defaultRequest(), nil, nil)
	require.True(t, second.succeeded)
	require.NotEqual(t, first.workerId, second.workerId, "equal-affinity workers with free slots must not pack onto the same worker")
}

func TestScheduler_RankWorkerIds_StickyDoesNotRequirePoolsByWorker(t *testing.T) {
	tenantId := uuid.New()
	desiredWorkerId := uuid.New()

	s := newTestScheduler(t, tenantId, &mockAssignmentRepo{})

	// Candidate is listed for the action, but has no poolsByWorker entry yet.
	// Sticky ranking only needs the worker id, so it must not drop the candidate.
	qi := testQI(tenantId, "A", 1)
	qi.Sticky = sqlcv1.V1StickyStrategyHARD
	qi.DesiredWorkerID = &desiredWorkerId

	var ranked []uuid.UUID
	onLoop(t, s, func() {
		ranked, _ = s.rankWorkerIds(qi, nil, []uuid.UUID{desiredWorkerId})
	})
	require.Equal(t, []uuid.UUID{desiredWorkerId}, ranked)
}

func TestScheduler_RankWorkerIds_LabelsUseWorkersMap(t *testing.T) {
	tenantId := uuid.New()
	workerId := uuid.New()

	s := newTestScheduler(t, tenantId, &mockAssignmentRepo{})
	s.setWorkers([]*repo.ListActiveWorkersResult{{
		ID:   workerId,
		Name: "w",
		Labels: []*sqlcv1.ListManyWorkerLabelsRow{
			{
				Key:      "region",
				StrValue: sqlchelpers.TextFromStr("us-east-1"),
			},
		},
	}})

	qi := testQI(tenantId, "A", 1)
	labels := []*sqlcv1.GetDesiredLabelsRow{{
		Key:        "region",
		StrValue:   sqlchelpers.TextFromStr("us-east-1"),
		Comparator: sqlcv1.WorkerLabelComparatorEQUAL,
		Weight:     10,
	}}

	// No poolsByWorker entry — label ranking must still resolve the worker via s.workers.
	var ranked []uuid.UUID
	onLoop(t, s, func() {
		ranked, _ = s.rankWorkerIds(qi, labels, []uuid.UUID{workerId})
	})
	require.Equal(t, []uuid.UUID{workerId}, ranked)
}

func TestScheduler_AssignSingleton_RateLimitAckIsWiredIntoAck(t *testing.T) {
	tenantId := uuid.New()
	workerId := uuid.New()

	s := newTestScheduler(t, tenantId, &mockAssignmentRepo{})
	w := &worker{ListActiveWorkersResult: testWorker(workerId)}

	sl := newSlot(w, repo.SlotTypeDefault)
	a := seedActionPools(t, s, "A", sl)
	qi := testQI(tenantId, "A", 1)

	ackCount := 0
	rlAck := func() { ackCount++ }

	res := assignOne(t, s, a, qi, nil, defaultRequest(), rlAck, nil)
	require.True(t, res.succeeded)

	s.ack([]int{res.ackId})
	require.Equal(t, 1, ackCount)
}

func TestScheduler_TryAssignBatch_NoActionSlots(t *testing.T) {
	tenantId := uuid.New()

	s := newTestScheduler(t, tenantId, &mockAssignmentRepo{})

	qis := []*sqlcv1.V1QueueItem{
		testQI(tenantId, "missing", 1),
		testQI(tenantId, "missing", 2),
	}

	res, err := s.tryAssignBatch(context.Background(), "missing", qis, nil, nil, nil, nil, nil)
	require.NoError(t, err)
	require.Len(t, res, 2)
	for _, r := range res {
		require.True(t, r.noSlots)
		require.False(t, r.succeeded)
	}
}

func TestScheduler_Replenish_SkipsIfReplenishInProgress(t *testing.T) {
	tenantId := uuid.New()
	s := newTestScheduler(t, tenantId, &mockAssignmentRepo{
		listActionsForWorkersFn: func(ctx context.Context, tenantId uuid.UUID, workerIds []uuid.UUID) ([]*sqlcv1.ListActionsForWorkersRow, error) {
			t.Error("should not hit repo while another replenish is in flight")
			return nil, nil
		},
	})

	onLoop(t, s, func() { s.replenishing = true })

	require.NoError(t, s.replenish(context.Background(), false))
	require.NoError(t, s.replenish(context.Background(), true))

	// the in-flight cycle still owns the flag
	onLoop(t, s, func() { require.True(t, s.replenishing) })
}

func TestScheduler_TryAssign_NotBlockedByReplenishDBReads(t *testing.T) {
	tenantId := uuid.New()
	workerId := uuid.New()

	// the replenish DB read hangs until its 2s timeout on every cycle
	ar := &mockAssignmentRepo{
		listActionsForWorkersFn: func(ctx context.Context, tenantId uuid.UUID, workerIds []uuid.UUID) ([]*sqlcv1.ListActionsForWorkersRow, error) {
			<-ctx.Done()
			return nil, ctx.Err()
		},
	}

	s := newTestScheduler(t, tenantId, ar)
	s.setWorkers([]*repo.ListActiveWorkersResult{testWorker(workerId)})

	const (
		assignBudget = 500 * time.Millisecond
		numProbers   = 16
		probeFor     = 3 * time.Second
	)
	qis := []*sqlcv1.V1QueueItem{testQI(tenantId, "missing", 1)}

	ctx, cancel := context.WithCancel(context.Background())
	loopDone := make(chan struct{})
	go func() {
		defer close(loopDone)
		s.loopReplenish(ctx)
	}()
	time.Sleep(1100 * time.Millisecond)

	probeCtx, stopProbing := context.WithTimeout(context.Background(), probeFor)
	defer stopProbing()

	var (
		mu         sync.Mutex
		maxLatency time.Duration
		wg         sync.WaitGroup
	)

	for i := 0; i < numProbers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for probeCtx.Err() == nil {
				start := time.Now()
				_, _ = s.tryAssignBatch(context.Background(), "missing", qis, nil, nil, nil, nil, nil)
				d := time.Since(start)

				mu.Lock()
				if d > maxLatency {
					maxLatency = d
				}
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	cancel()
	<-loopDone

	t.Logf("max latency observed: %s", maxLatency)
	require.LessOrEqual(t, maxLatency, assignBudget,
		"tryAssignBatch was blocked by replenish DB reads: max latency %s", maxLatency)
}

func TestScheduler_TryAssignBatch_AssignsUntilExhausted(t *testing.T) {
	tenantId := uuid.New()
	workerId := uuid.New()

	s := newTestScheduler(t, tenantId, &mockAssignmentRepo{})
	w := &worker{ListActiveWorkersResult: testWorker(workerId)}

	// two total slots
	sl1 := newSlot(w, repo.SlotTypeDefault)
	sl2 := newSlot(w, repo.SlotTypeDefault)

	seedActionPools(t, s, "A", sl1, sl2)

	qis := []*sqlcv1.V1QueueItem{
		testQI(tenantId, "A", 1),
		testQI(tenantId, "A", 2),
		testQI(tenantId, "A", 3),
	}

	res, err := s.tryAssignBatch(context.Background(), "A", qis, map[uuid.UUID][]*sqlcv1.GetDesiredLabelsRow{}, map[uuid.UUID]map[string]int32{}, nil, nil, nil)
	require.NoError(t, err)

	var assigned, noSlots int
	for _, r := range res {
		if r.succeeded {
			assigned++
		}
		if r.noSlots {
			noSlots++
		}
	}

	require.Equal(t, 2, assigned)
	require.Equal(t, 1, noSlots)
}

func TestScheduler_TryAssignBatch_RateLimitedSkipsAssignment(t *testing.T) {
	tenantId := uuid.New()
	workerId := uuid.New()

	s := newTestScheduler(t, tenantId, &mockAssignmentRepo{})
	l := zerolog.Nop()
	s.rl = &rateLimiter{
		tenantId:     tenantId,
		l:            &l,
		unacked:      make(map[int64]rateLimitSet),
		unflushed:    make(rateLimitSet),
		dbRateLimits: rateLimitSet{"k": {key: "k", val: 0, nextRefillAt: ptrTime(time.Now().UTC().Add(10 * time.Second))}},
	}

	w := &worker{ListActiveWorkersResult: testWorker(workerId)}
	sl := newSlot(w, repo.SlotTypeDefault)
	seedActionPools(t, s, "A", sl)

	qi := testQI(tenantId, "A", 100)
	qis := []*sqlcv1.V1QueueItem{qi}

	rls := map[int64]map[string]int32{
		qi.TaskID: {"k": 1},
	}

	res, err := s.tryAssignBatch(context.Background(), "A", qis, nil, map[uuid.UUID]map[string]int32{}, rls, nil, nil)
	require.NoError(t, err)
	require.Len(t, res, 1)
	require.False(t, res[0].succeeded)
	require.NotNil(t, res[0].rateLimitResult)
	require.False(t, res[0].noSlots)
}

func TestScheduler_TryAssign_GroupsAndFiltersTimedOut(t *testing.T) {
	tenantId := uuid.New()

	s := newTestScheduler(t, tenantId, &mockAssignmentRepo{})
	workerA := &worker{ListActiveWorkersResult: testWorker(uuid.New())}
	workerB := &worker{ListActiveWorkersResult: testWorker(uuid.New())}

	// A and B each have one independently owned worker slot.
	seedActionPools(t, s, "A", newSlot(workerA, repo.SlotTypeDefault))
	seedActionPools(t, s, "B", newSlot(workerB, repo.SlotTypeDefault))

	timeoutQI := testQI(tenantId, "A", 1)
	timeoutQI.ScheduleTimeoutAt = ts(time.Now().UTC().Add(-1 * time.Second))

	a1 := testQI(tenantId, "A", 2)
	a2 := testQI(tenantId, "A", 3) // will be unassigned (only one slot)
	b1 := testQI(tenantId, "B", 4)

	ch := s.tryAssign(
		context.Background(),
		[]*sqlcv1.V1QueueItem{timeoutQI, a1, a2, b1},
		map[uuid.UUID][]*sqlcv1.GetDesiredLabelsRow{},
		map[uuid.UUID]map[string]int32{},
		nil,
		nil,
		nil,
	)

	var (
		assignedIDs  = map[int64]bool{}
		unassignedID = map[int64]bool{}
		timedOutID   = map[int64]bool{}
	)

	for r := range ch {
		for _, to := range r.schedulingTimedOut {
			timedOutID[to.TaskID] = true
		}
		for _, u := range r.unassigned {
			unassignedID[u.TaskID] = true
		}
		for _, a := range r.assigned {
			assignedIDs[a.QueueItem.TaskID] = true
		}
	}

	require.True(t, timedOutID[timeoutQI.TaskID])
	require.True(t, assignedIDs[a1.TaskID] || assignedIDs[a2.TaskID])   // one of them assigned
	require.True(t, unassignedID[a1.TaskID] || unassignedID[a2.TaskID]) // the other unassigned
	require.True(t, assignedIDs[b1.TaskID])
}

func TestScheduler_GetExtensionInput(t *testing.T) {
	tenantId := uuid.New()
	s := newTestScheduler(t, tenantId, &mockAssignmentRepo{})

	qi1 := testQI(tenantId, "A", 1)
	qi2 := testQI(tenantId, "A", 2)

	in := s.getExtensionInput([]*assignResults{
		{unassigned: []*sqlcv1.V1QueueItem{qi1}},
		{unassigned: []*sqlcv1.V1QueueItem{}},
		{unassigned: []*sqlcv1.V1QueueItem{qi2}},
	})

	require.True(t, in.HasUnassignedStepRuns)

	in2 := s.getExtensionInput([]*assignResults{{unassigned: nil}})
	require.False(t, in2.HasUnassignedStepRuns)
}

func TestScheduler_GetSnapshotInput_CanceledContext(t *testing.T) {
	tenantId := uuid.New()
	s := newTestScheduler(t, tenantId, &mockAssignmentRepo{})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	in, ok := s.getSnapshotInput(ctx)
	require.False(t, ok)
	require.Nil(t, in)
}

func TestScheduler_GetSnapshotInput_DerivesUsedSlotsFromCapacity(t *testing.T) {
	tenantId := uuid.New()
	workerId := uuid.New()

	s := newTestScheduler(t, tenantId, &mockAssignmentRepo{
		listActionsForWorkersFn: func(ctx context.Context, tenantId uuid.UUID, workerIds []uuid.UUID) ([]*sqlcv1.ListActionsForWorkersRow, error) {
			return []*sqlcv1.ListActionsForWorkersRow{
				{WorkerId: workerId, ActionId: sqlchelpers.TextFromStr("A")},
			}, nil
		},
		listWorkerSlotConfigsFn: func(ctx context.Context, tenantId uuid.UUID, workerIds []uuid.UUID) ([]*sqlcv1.ListWorkerSlotConfigsRow, error) {
			return []*sqlcv1.ListWorkerSlotConfigsRow{
				{WorkerID: workerId, SlotType: repo.SlotTypeDefault, MaxUnits: 5},
				{WorkerID: workerId, SlotType: repo.SlotTypeDurable, MaxUnits: 10},
			}, nil
		},
		listAvailableSlotsForWorkersAndTypesFn: func(ctx context.Context, tenantId uuid.UUID, params sqlcv1.ListAvailableSlotsForWorkersAndTypesParams) ([]*sqlcv1.ListAvailableSlotsForWorkersAndTypesRow, error) {
			// 3 of the 5 default slots are consumed by running tasks, so only 2 free
			// default slots make it into the in-memory pool; durable slots are idle
			return []*sqlcv1.ListAvailableSlotsForWorkersAndTypesRow{
				{ID: workerId, SlotType: repo.SlotTypeDefault, AvailableSlots: 2},
				{ID: workerId, SlotType: repo.SlotTypeDurable, AvailableSlots: 10},
			}, nil
		},
	})

	s.setWorkers([]*repo.ListActiveWorkersResult{{
		ID:               workerId,
		Name:             "w1",
		TotalSlotsByType: map[string]int{repo.SlotTypeDefault: 5, repo.SlotTypeDurable: 10},
	}})

	require.NoError(t, s.replenish(context.Background(), true))

	in, ok := s.getSnapshotInput(context.Background())
	require.True(t, ok)
	require.NotNil(t, in)
	require.Len(t, in.Workers, 1)
	require.Equal(t, workerId, in.Workers[workerId].WorkerId)
	require.Equal(t, 15, in.Workers[workerId].MaxRuns)

	util := in.WorkerSlotUtilization[workerId]
	require.NotNil(t, util)
	require.Equal(t, 3, util.UtilizedSlots)
	require.Equal(t, 12, util.NonUtilizedSlots)

	byType := in.WorkerSlotUtilizationByType[workerId]
	require.NotNil(t, byType)
	require.Equal(t, &SlotUtilization{UtilizedSlots: 3, NonUtilizedSlots: 2}, byType[repo.SlotTypeDefault])
	require.Equal(t, &SlotUtilization{UtilizedSlots: 0, NonUtilizedSlots: 10}, byType[repo.SlotTypeDurable])
}

func TestScheduler_GetSnapshotInput_DedupSlotsAcrossActions(t *testing.T) {
	tenantId := uuid.New()
	workerId := uuid.New()
	s := newTestScheduler(t, tenantId, &mockAssignmentRepo{})

	s.setWorkers([]*repo.ListActiveWorkersResult{{
		ID:               workerId,
		Name:             "w1",
		TotalSlotsByType: map[string]int{repo.SlotTypeDefault: 3},
	}})

	w := &worker{ListActiveWorkersResult: testWorker(workerId)}
	usedSlot := newSlot(w, repo.SlotTypeDefault)
	usedSlot.used = true
	unusedSlot := newSlot(w, repo.SlotTypeDefault)

	// both actions index the same worker pool
	seedActionPools(t, s, "A", usedSlot, unusedSlot)
	seedActionPools(t, s, "B", usedSlot, unusedSlot)

	in, ok := s.getSnapshotInput(context.Background())
	require.True(t, ok)
	require.NotNil(t, in)
	require.Len(t, in.Workers, 1)
	require.Equal(t, workerId, in.Workers[workerId].WorkerId)

	// the free slot must be counted once across both actions, so 3 total - 1 free = 2 used
	util := in.WorkerSlotUtilization[workerId]
	require.NotNil(t, util)
	require.Equal(t, 2, util.UtilizedSlots)
	require.Equal(t, 1, util.NonUtilizedSlots)
}

func TestScheduler_GetSnapshotInput_WarmupAndSaturation(t *testing.T) {
	tenantId := uuid.New()
	workerId := uuid.New()
	s := newTestScheduler(t, tenantId, &mockAssignmentRepo{})

	s.setWorkers([]*repo.ListActiveWorkersResult{{
		ID:               workerId,
		Name:             "w1",
		TotalSlotsByType: map[string]int{repo.SlotTypeDefault: 3},
	}})

	// before any slots have entered the pool (i.e. before the first replenish), the worker
	// must report zero slots rather than full utilization
	in, ok := s.getSnapshotInput(context.Background())
	require.True(t, ok)
	require.Equal(t, &SlotUtilization{UtilizedSlots: 0, NonUtilizedSlots: 0}, in.WorkerSlotUtilization[workerId])

	// slots appear in the pool: the slot type warms up and utilization is derived from capacity
	w := &worker{ListActiveWorkersResult: testWorker(workerId)}
	freeSlot := newSlot(w, repo.SlotTypeDefault)
	seedActionPools(t, s, "A", freeSlot)

	in, ok = s.getSnapshotInput(context.Background())
	require.True(t, ok)
	require.Equal(t, &SlotUtilization{UtilizedSlots: 2, NonUtilizedSlots: 1}, in.WorkerSlotUtilization[workerId])

	// the pool empties out (all slots assigned and flushed): the warmed type now reports
	// full utilization instead of a transient zero
	onLoop(t, s, func() {
		delete(s.actions, "A")
		s.pools[poolKey{workerId: workerId, slotType: repo.SlotTypeDefault}].reset(nil, time.Now().Add(defaultSlotExpiry))
	})

	in, ok = s.getSnapshotInput(context.Background())
	require.True(t, ok)
	require.Equal(t, &SlotUtilization{UtilizedSlots: 3, NonUtilizedSlots: 0}, in.WorkerSlotUtilization[workerId])

	// removing the worker prunes its warm state
	s.setWorkers([]*repo.ListActiveWorkersResult{})

	in, ok = s.getSnapshotInput(context.Background())
	require.True(t, ok)
	require.NotContains(t, in.WorkerSlotUtilization, workerId)
	onLoop(t, s, func() { require.Empty(t, s.warmedSlotTypes) })
}

func TestScheduler_GetSnapshotInput_FallsBackToWalkedCountsWithoutCapacity(t *testing.T) {
	tenantId := uuid.New()
	workerId := uuid.New()
	s := newTestScheduler(t, tenantId, &mockAssignmentRepo{})

	// no TotalSlots on the worker record
	s.setWorkers([]*repo.ListActiveWorkersResult{{ID: workerId, Name: "w1", Labels: nil}})

	w := &worker{ListActiveWorkersResult: testWorker(workerId)}
	usedSlot := newSlot(w, repo.SlotTypeDefault)
	usedSlot.used = true
	unusedSlot := newSlot(w, repo.SlotTypeDefault)

	seedActionPools(t, s, "A", usedSlot, unusedSlot)

	in, ok := s.getSnapshotInput(context.Background())
	require.True(t, ok)

	util := in.WorkerSlotUtilization[workerId]
	require.NotNil(t, util)
	require.Equal(t, 1, util.UtilizedSlots)
	require.Equal(t, 1, util.NonUtilizedSlots)
}

func TestScheduler_IsTimedOut(t *testing.T) {
	tenantId := uuid.New()
	qi := testQI(tenantId, "A", 1)
	require.False(t, isTimedOut(qi))

	qi.ScheduleTimeoutAt = ts(time.Now().UTC().Add(-1 * time.Millisecond))
	require.True(t, isTimedOut(qi))

	qi.ScheduleTimeoutAt = ts(time.Now().UTC().Add(5 * time.Second))
	require.False(t, isTimedOut(qi))
}

func TestScheduler_LoopsExitOnCancel(t *testing.T) {
	tenantId := uuid.New()
	s := newTestScheduler(t, tenantId, &mockAssignmentRepo{})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	doneRepl := make(chan struct{})
	go func() {
		s.loopReplenish(ctx)
		close(doneRepl)
	}()

	doneSnap := make(chan struct{})
	go func() {
		s.loopSnapshot(ctx)
		close(doneSnap)
	}()

	select {
	case <-doneRepl:
	case <-time.After(250 * time.Millisecond):
		t.Fatalf("loopReplenish did not exit on cancel")
	}

	select {
	case <-doneSnap:
	case <-time.After(250 * time.Millisecond):
		t.Fatalf("loopSnapshot did not exit on cancel")
	}
}

func ptrTime(t time.Time) *time.Time { return &t }

func TestScheduler_Start_Smoke(t *testing.T) {
	l := zerolog.Nop()
	cf := &sharedConfig{
		repo: &mockSchedulerRepo{assignment: &mockAssignmentRepo{}},
		l:    &l,
	}
	s := newScheduler(cf, uuid.New(), nil, &Extensions{})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// should not block or panic even if canceled
	s.start(ctx)

	// ops against a dead run loop fail fast instead of hanging
	require.Eventually(t, func() bool {
		return !s.do(context.Background(), func() {})
	}, time.Second, 5*time.Millisecond)
}

func TestSelectSlotsFromPools_MissingTypeOrInsufficientUnitsFails(t *testing.T) {
	workerId := uuid.New()
	w := &worker{ListActiveWorkersResult: testWorker(workerId)}

	one := newSlot(w, repo.SlotTypeDefault)

	pool := &slotPool{worker: w, slotType: repo.SlotTypeDefault}
	pool.reset([]*slot{one}, time.Now().Add(defaultSlotExpiry))
	poolsByType := map[string]*slotPool{repo.SlotTypeDefault: pool}

	now := time.Now()

	_, ok := selectSlotsFromPools(poolsByType, map[string]int32{repo.SlotTypeDurable: 1}, now)
	require.False(t, ok)

	_, ok = selectSlotsFromPools(poolsByType, map[string]int32{repo.SlotTypeDefault: 2}, now)
	require.False(t, ok)

	// a failed multi-type selection must not leak reservations from the types
	// which did have capacity
	_, ok = selectSlotsFromPools(poolsByType, map[string]int32{repo.SlotTypeDefault: 1, repo.SlotTypeDurable: 1}, now)
	require.False(t, ok)
	require.Equal(t, 1, pool.freeCountAt(now))
}

func TestScheduler_AssignSingleton_MultiUnitSameType(t *testing.T) {
	s := newTestScheduler(t, uuid.New(), &mockAssignmentRepo{})
	workerId := uuid.New()
	w := &worker{ListActiveWorkersResult: testWorker(workerId)}

	s1 := newSlot(w, repo.SlotTypeDefault)
	s2 := newSlot(w, repo.SlotTypeDefault)
	s3 := newSlot(w, repo.SlotTypeDefault)
	s3.used = true // ensure not selected

	a := seedActionPools(t, s, "A", s1, s2, s3)

	qi := testQI(uuid.New(), "A", 1)
	res := assignOne(t, s, a, qi, nil, map[string]int32{repo.SlotTypeDefault: 2}, nil, nil)
	require.True(t, res.succeeded)
	require.Equal(t, workerId, res.workerId)

	onLoop(t, s, func() {
		assigned := s.unackedSlots[res.ackId]
		require.NotNil(t, assigned)
		require.Len(t, assigned.slots, 2)
		for _, sl := range assigned.slots {
			require.True(t, sl.used)
		}
	})
}

func TestScheduler_AssignSingleton_MultiType(t *testing.T) {
	s := newTestScheduler(t, uuid.New(), &mockAssignmentRepo{})
	workerId := uuid.New()
	w := &worker{ListActiveWorkersResult: testWorker(workerId)}

	def := newSlot(w, repo.SlotTypeDefault)
	dur := newSlot(w, repo.SlotTypeDurable)

	a := seedActionPools(t, s, "A", def, dur)

	qi := testQI(uuid.New(), "A", 1)
	res := assignOne(t, s, a, qi, nil, map[string]int32{repo.SlotTypeDefault: 1, repo.SlotTypeDurable: 1}, nil, nil)
	require.True(t, res.succeeded)

	onLoop(t, s, func() {
		assigned := s.unackedSlots[res.ackId]
		require.NotNil(t, assigned)
		require.Len(t, assigned.slots, 2)

		gotTypes := map[string]bool{}
		for _, sl := range assigned.slots {
			gotTypes[sl.slotType] = true
		}
		require.True(t, gotTypes[repo.SlotTypeDefault])
		require.True(t, gotTypes[repo.SlotTypeDurable])
	})
}

func TestScheduler_Nack_CallsRateLimitNackOnce(t *testing.T) {
	tenantId := uuid.New()
	workerId := uuid.New()
	s := newTestScheduler(t, tenantId, &mockAssignmentRepo{})

	w := &worker{ListActiveWorkersResult: testWorker(workerId)}
	sl := newSlot(w, repo.SlotTypeDefault)
	seedActionPools(t, s, "A", sl)
	takeSlot(t, s, sl)

	nackCount := 0
	onLoop(t, s, func() {
		s.unackedSlots[1] = &assignedSlots{
			slots:         []*slot{sl},
			rateLimitNack: func() { nackCount++ },
		}
	})

	s.nack([]int{1})
	s.nack([]int{1})

	require.Equal(t, 1, nackCount)
	onLoop(t, s, func() { require.False(t, sl.used) })
}

func TestScheduler_Replenish_MultipleSlotTypes_CallsRepoPerTypeAndPopulatesSlotsByWorker(t *testing.T) {
	tenantId := uuid.New()
	workerId := uuid.New()

	called := map[string]int{}
	var calledMu sync.Mutex

	ar := &mockAssignmentRepo{
		listActionsForWorkersFn: func(ctx context.Context, tenantId uuid.UUID, workerIds []uuid.UUID) ([]*sqlcv1.ListActionsForWorkersRow, error) {
			return []*sqlcv1.ListActionsForWorkersRow{
				{WorkerId: workerId, ActionId: sqlchelpers.TextFromStr("A")},
			}, nil
		},
		listWorkerSlotConfigsFn: func(ctx context.Context, tenantId uuid.UUID, workerIds []uuid.UUID) ([]*sqlcv1.ListWorkerSlotConfigsRow, error) {
			return []*sqlcv1.ListWorkerSlotConfigsRow{
				{WorkerID: workerId, SlotType: repo.SlotTypeDefault, MaxUnits: 2},
				{WorkerID: workerId, SlotType: repo.SlotTypeDurable, MaxUnits: 2},
			}, nil
		},
		listAvailableSlotsForWorkersFn: func(ctx context.Context, tenantId uuid.UUID, params sqlcv1.ListAvailableSlotsForWorkersParams) ([]*sqlcv1.ListAvailableSlotsForWorkersRow, error) {
			calledMu.Lock()
			called[params.Slottype]++
			calledMu.Unlock()
			switch params.Slottype {
			case repo.SlotTypeDefault:
				return []*sqlcv1.ListAvailableSlotsForWorkersRow{{ID: workerId, AvailableSlots: 2}}, nil
			case repo.SlotTypeDurable:
				return []*sqlcv1.ListAvailableSlotsForWorkersRow{{ID: workerId, AvailableSlots: 2}}, nil
			default:
				return nil, fmt.Errorf("unexpected slot type %q", params.Slottype)
			}
		},
	}

	s := newTestScheduler(t, tenantId, ar)
	s.setWorkers([]*repo.ListActiveWorkersResult{testWorker(workerId)})

	err := s.replenish(context.Background(), true)
	require.NoError(t, err)

	require.Equal(t, 1, called[repo.SlotTypeDefault])
	require.Equal(t, 1, called[repo.SlotTypeDurable])

	var a *action
	onLoop(t, s, func() { a = s.actions["A"] })
	require.NotNil(t, a)
	actionSlots := slotsForAction(t, s, a)
	require.Len(t, actionSlots, 4)

	countByType := map[string]int{}
	for _, sl := range actionSlots {
		if sl.getWorkerId() != workerId {
			continue
		}
		countByType[sl.slotType]++
	}
	require.Equal(t, 2, countByType[repo.SlotTypeDefault])
	require.Equal(t, 2, countByType[repo.SlotTypeDurable])
}

func TestScheduler_Replenish_UnackedCountsPerSlotType(t *testing.T) {
	tenantId := uuid.New()
	workerId := uuid.New()

	ar := &mockAssignmentRepo{
		listActionsForWorkersFn: func(ctx context.Context, tenantId uuid.UUID, workerIds []uuid.UUID) ([]*sqlcv1.ListActionsForWorkersRow, error) {
			return []*sqlcv1.ListActionsForWorkersRow{
				{WorkerId: workerId, ActionId: sqlchelpers.TextFromStr("A")},
			}, nil
		},
		listWorkerSlotConfigsFn: func(ctx context.Context, tenantId uuid.UUID, workerIds []uuid.UUID) ([]*sqlcv1.ListWorkerSlotConfigsRow, error) {
			return []*sqlcv1.ListWorkerSlotConfigsRow{
				{WorkerID: workerId, SlotType: repo.SlotTypeDefault, MaxUnits: 2},
				{WorkerID: workerId, SlotType: repo.SlotTypeDurable, MaxUnits: 2},
			}, nil
		},
		listAvailableSlotsForWorkersFn: func(ctx context.Context, tenantId uuid.UUID, params sqlcv1.ListAvailableSlotsForWorkersParams) ([]*sqlcv1.ListAvailableSlotsForWorkersRow, error) {
			return []*sqlcv1.ListAvailableSlotsForWorkersRow{{ID: workerId, AvailableSlots: 2}}, nil
		},
	}

	s := newTestScheduler(t, tenantId, ar)
	s.setWorkers([]*repo.ListActiveWorkersResult{testWorker(workerId)})

	// Seed one unacked durable slot; should only reduce *durable* new-slot count.
	w := &worker{ListActiveWorkersResult: testWorker(workerId)}
	unackedDurable := &slot{worker: w, slotType: repo.SlotTypeDurable, used: true}
	onLoop(t, s, func() {
		s.unackedSlots[1] = &assignedSlots{slots: []*slot{unackedDurable}}
	})

	err := s.replenish(context.Background(), true)
	require.NoError(t, err)

	var a *action
	onLoop(t, s, func() { a = s.actions["A"] })
	require.NotNil(t, a)

	countDefault := 0
	countDurable := 0
	foundUnacked := false
	for _, sl := range slotsForAction(t, s, a) {
		if sl.getWorkerId() != workerId {
			continue
		}

		switch sl.slotType {
		case repo.SlotTypeDefault:
			countDefault++
		case repo.SlotTypeDurable:
			countDurable++
			if sl == unackedDurable {
				foundUnacked = true
			}
		}
	}

	// default should be unaffected: 2 fresh default slots
	require.Equal(t, 2, countDefault)
	// durable should still total to 2, but include the unacked durable slot
	require.Equal(t, 2, countDurable)
	require.True(t, foundUnacked, "expected unacked durable slot to be carried forward into replenished slots")

	// the carried slot points at the rebuilt pool so a later nack can release it
	onLoop(t, s, func() {
		require.NotNil(t, unackedDurable.pool)
		require.Equal(t, repo.SlotTypeDurable, unackedDurable.pool.slotType)
		require.True(t, unackedDurable.used)
		require.Equal(t, 1, len(unackedDurable.pool.free))
	})
}

func TestScheduler_Replenish_SubtractsAcksDuringReplenish(t *testing.T) {
	tenantId := uuid.New()
	workerId := uuid.New()

	// The availability read reports 3 free slots, but an assignment acks while
	// the read is in flight: its slot must not be resurrected into the pool.
	availableRead := make(chan struct{})
	ackDone := make(chan struct{})

	ar := &mockAssignmentRepo{
		listActionsForWorkersFn: func(ctx context.Context, tenantId uuid.UUID, workerIds []uuid.UUID) ([]*sqlcv1.ListActionsForWorkersRow, error) {
			return []*sqlcv1.ListActionsForWorkersRow{
				{WorkerId: workerId, ActionId: sqlchelpers.TextFromStr("A")},
			}, nil
		},
		listAvailableSlotsForWorkersFn: func(ctx context.Context, tenantId uuid.UUID, params sqlcv1.ListAvailableSlotsForWorkersParams) ([]*sqlcv1.ListAvailableSlotsForWorkersRow, error) {
			close(availableRead)
			<-ackDone
			return []*sqlcv1.ListAvailableSlotsForWorkersRow{{ID: workerId, AvailableSlots: 3}}, nil
		},
	}

	s := newTestScheduler(t, tenantId, ar)
	s.setWorkers([]*repo.ListActiveWorkersResult{testWorker(workerId)})

	// one outstanding assignment, not yet flushed
	w := &worker{ListActiveWorkersResult: testWorker(workerId)}
	sl := newSlot(w, repo.SlotTypeDefault)
	seedActionPools(t, s, "A", sl)
	takeSlot(t, s, sl)
	onLoop(t, s, func() {
		s.unackedSlots[42] = &assignedSlots{slots: []*slot{sl}}
	})

	replenishDone := make(chan error, 1)
	go func() {
		replenishDone <- s.replenish(context.Background(), true)
	}()

	// ack the assignment while the availability read is in flight
	<-availableRead
	s.ack([]int{42})
	close(ackDone)

	require.NoError(t, <-replenishDone)

	onLoop(t, s, func() {
		pool := s.pools[poolKey{workerId: workerId, slotType: repo.SlotTypeDefault}]
		require.NotNil(t, pool)
		// 3 reported available - 1 acked during the read = 2 slots
		require.Len(t, pool.slots, 2)
		require.Len(t, pool.free, 2)
		require.False(t, s.replenishing)
		require.Nil(t, s.ackedDuringReplenish)
	})
}

func TestScheduler_Replenish_PropagatesRepoErrors(t *testing.T) {
	tenantId := uuid.New()
	workerId := uuid.New()
	sentinel := fmt.Errorf("boom")

	requireFlagCleared := func(t *testing.T, s *Scheduler) {
		t.Helper()
		onLoop(t, s, func() {
			require.False(t, s.replenishing, "replenishing flag must be cleared on error")
		})
	}

	t.Run("ListActionsForWorkers", func(t *testing.T) {
		s := newTestScheduler(t, tenantId, &mockAssignmentRepo{
			listActionsForWorkersFn: func(ctx context.Context, tenantId uuid.UUID, workerIds []uuid.UUID) ([]*sqlcv1.ListActionsForWorkersRow, error) {
				return nil, sentinel
			},
		})
		s.setWorkers([]*repo.ListActiveWorkersResult{testWorker(workerId)})
		err := s.replenish(context.Background(), true)
		require.ErrorIs(t, err, sentinel)
		requireFlagCleared(t, s)
	})

	t.Run("ListWorkerSlotConfigs", func(t *testing.T) {
		s := newTestScheduler(t, tenantId, &mockAssignmentRepo{
			listActionsForWorkersFn: func(ctx context.Context, tenantId uuid.UUID, workerIds []uuid.UUID) ([]*sqlcv1.ListActionsForWorkersRow, error) {
				return []*sqlcv1.ListActionsForWorkersRow{
					{WorkerId: workerId, ActionId: sqlchelpers.TextFromStr("A")},
				}, nil
			},
			listWorkerSlotConfigsFn: func(ctx context.Context, tenantId uuid.UUID, workerIds []uuid.UUID) ([]*sqlcv1.ListWorkerSlotConfigsRow, error) {
				return nil, sentinel
			},
		})
		s.setWorkers([]*repo.ListActiveWorkersResult{testWorker(workerId)})
		err := s.replenish(context.Background(), true)
		require.ErrorIs(t, err, sentinel)
		requireFlagCleared(t, s)
	})

	t.Run("ListAvailableSlotsForWorkers", func(t *testing.T) {
		s := newTestScheduler(t, tenantId, &mockAssignmentRepo{
			listActionsForWorkersFn: func(ctx context.Context, tenantId uuid.UUID, workerIds []uuid.UUID) ([]*sqlcv1.ListActionsForWorkersRow, error) {
				return []*sqlcv1.ListActionsForWorkersRow{
					{WorkerId: workerId, ActionId: sqlchelpers.TextFromStr("A")},
				}, nil
			},
			listWorkerSlotConfigsFn: func(ctx context.Context, tenantId uuid.UUID, workerIds []uuid.UUID) ([]*sqlcv1.ListWorkerSlotConfigsRow, error) {
				return []*sqlcv1.ListWorkerSlotConfigsRow{
					{WorkerID: workerId, SlotType: repo.SlotTypeDefault, MaxUnits: 2},
				}, nil
			},
			listAvailableSlotsForWorkersFn: func(ctx context.Context, tenantId uuid.UUID, params sqlcv1.ListAvailableSlotsForWorkersParams) ([]*sqlcv1.ListAvailableSlotsForWorkersRow, error) {
				return nil, sentinel
			},
		})
		s.setWorkers([]*repo.ListActiveWorkersResult{testWorker(workerId)})
		err := s.replenish(context.Background(), true)
		require.ErrorIs(t, err, sentinel)
		requireFlagCleared(t, s)
	})
}

func TestScheduler_Replenish_CreatesActionAndSlots(t *testing.T) {
	tenantId := uuid.New()
	workerId := uuid.New()

	ar := &mockAssignmentRepo{
		listActionsForWorkersFn: func(ctx context.Context, gotTenantId uuid.UUID, workerIds []uuid.UUID) ([]*sqlcv1.ListActionsForWorkersRow, error) {
			require.Equal(t, tenantId, gotTenantId)
			require.Len(t, workerIds, 1)
			require.Equal(t, workerId, workerIds[0])

			return []*sqlcv1.ListActionsForWorkersRow{
				{WorkerId: workerId, ActionId: sqlchelpers.TextFromStr("A")},
			}, nil
		},
		listAvailableSlotsForWorkersFn: func(ctx context.Context, tenantId uuid.UUID, params sqlcv1.ListAvailableSlotsForWorkersParams) ([]*sqlcv1.ListAvailableSlotsForWorkersRow, error) {
			require.Equal(t, repo.SlotTypeDefault, params.Slottype)
			return []*sqlcv1.ListAvailableSlotsForWorkersRow{
				{ID: workerId, AvailableSlots: 3},
			}, nil
		},
	}

	s := newTestScheduler(t, tenantId, ar)
	s.setWorkers([]*repo.ListActiveWorkersResult{testWorker(workerId)})

	err := s.replenish(context.Background(), true)
	require.NoError(t, err)

	var a *action
	onLoop(t, s, func() { a = s.actions["A"] })
	require.NotNil(t, a)

	actionSlots := slotsForAction(t, s, a)
	require.Len(t, actionSlots, 3)
	require.Equal(t, 3, a.lastReplenishedSlotCount)
	require.Equal(t, 1, a.lastReplenishedWorkerCount)

	for _, sl := range actionSlots {
		require.Equal(t, workerId, sl.getWorkerId())
		require.Equal(t, repo.SlotTypeDefault, sl.slotType)
	}
}

func TestScheduler_Replenish_RemovesActionWhenWorkerPoolHasNoCapacity(t *testing.T) {
	tenantId := uuid.New()
	workerId := uuid.New()

	ar := &mockAssignmentRepo{
		listActionsForWorkersFn: func(ctx context.Context, tenantId uuid.UUID, workerIds []uuid.UUID) ([]*sqlcv1.ListActionsForWorkersRow, error) {
			return []*sqlcv1.ListActionsForWorkersRow{
				{WorkerId: workerId, ActionId: sqlchelpers.TextFromStr("A")},
			}, nil
		},
		// simulate no rows returned => no new slots written
		listAvailableSlotsForWorkersFn: func(ctx context.Context, tenantId uuid.UUID, params sqlcv1.ListAvailableSlotsForWorkersParams) ([]*sqlcv1.ListAvailableSlotsForWorkersRow, error) {
			require.Equal(t, repo.SlotTypeDefault, params.Slottype)
			return []*sqlcv1.ListAvailableSlotsForWorkersRow{}, nil
		},
	}

	s := newTestScheduler(t, tenantId, ar)
	s.setWorkers([]*repo.ListActiveWorkersResult{testWorker(workerId)})

	// all seeded capacity is consumed, so the action triggers a replenish
	w := &worker{ListActiveWorkersResult: testWorker(workerId)}
	used := newSlot(w, repo.SlotTypeDefault)
	used.used = true

	a := seedActionPools(t, s, "A", used)
	onLoop(t, s, func() { a.lastReplenishedSlotCount = 2 })

	err := s.replenish(context.Background(), false)
	require.NoError(t, err)

	onLoop(t, s, func() {
		_, ok := s.actions["A"]
		require.False(t, ok)
		require.Empty(t, s.pools[poolKey{workerId: workerId, slotType: repo.SlotTypeDefault}].slots)
	})
}

func TestScheduler_Replenish_RefreshesAllWorkerPoolsWhenTriggered(t *testing.T) {
	tenantId := uuid.New()

	// Chain topology: w1 registers {A, B}, w2 registers {B, C}, w3 registers {C, D}.
	// A stale pool on one worker triggers a refresh of the canonical inventory for
	// every active worker, including pools that serve existing actions.
	w1Id := uuid.New()
	w2Id := uuid.New()
	w3Id := uuid.New()

	ar := &mockAssignmentRepo{
		listActionsForWorkersFn: func(ctx context.Context, tenantId uuid.UUID, workerIds []uuid.UUID) ([]*sqlcv1.ListActionsForWorkersRow, error) {
			return []*sqlcv1.ListActionsForWorkersRow{
				{WorkerId: w1Id, ActionId: sqlchelpers.TextFromStr("A")},
				{WorkerId: w1Id, ActionId: sqlchelpers.TextFromStr("B")},
				{WorkerId: w2Id, ActionId: sqlchelpers.TextFromStr("B")},
				{WorkerId: w2Id, ActionId: sqlchelpers.TextFromStr("C")},
				{WorkerId: w3Id, ActionId: sqlchelpers.TextFromStr("C")},
				{WorkerId: w3Id, ActionId: sqlchelpers.TextFromStr("D")},
			}, nil
		},
		// no new slots available, so replenish clears out all pools
		listAvailableSlotsForWorkersFn: func(ctx context.Context, tenantId uuid.UUID, params sqlcv1.ListAvailableSlotsForWorkersParams) ([]*sqlcv1.ListAvailableSlotsForWorkersRow, error) {
			return []*sqlcv1.ListAvailableSlotsForWorkersRow{}, nil
		},
	}

	s := newTestScheduler(t, tenantId, ar)
	s.setWorkers([]*repo.ListActiveWorkersResult{testWorker(w1Id), testWorker(w2Id), testWorker(w3Id)})

	w1 := &worker{ListActiveWorkersResult: testWorker(w1Id)}
	w2 := &worker{ListActiveWorkersResult: testWorker(w2Id)}
	w3 := &worker{ListActiveWorkersResult: testWorker(w3Id)}

	// seedAction creates an action with enough active slots that the heuristic does not
	// independently mark it for replenish (activeCount > lastReplenishedSlotCount/2
	// and worker count unchanged).
	seedAction := func(actionId string, workerCount int, w *worker) *action {
		a := seedActionPools(t, s, actionId,
			newSlot(w, repo.SlotTypeDefault),
			newSlot(w, repo.SlotTypeDefault),
		)

		onLoop(t, s, func() {
			a.lastReplenishedSlotCount = 2
			a.lastReplenishedWorkerCount = workerCount
		})

		return a
	}

	seedAction("B", 2, w1)
	seedAction("C", 2, w2)
	actD := seedAction("D", 1, w3)

	// force w3's pool stale so D reports zero active slots
	onLoop(t, s, func() {
		s.pools[poolKey{workerId: w3Id, slotType: repo.SlotTypeDefault}].expiresAt = time.Now().Add(-time.Second)
	})

	require.NoError(t, s.replenish(context.Background(), false))

	// The refreshed worker inventory is authoritative; stale action-owned slots
	// are not carried into the new pools.
	require.NotNil(t, actD)
	onLoop(t, s, func() {
		for _, pool := range s.pools {
			require.Empty(t, pool.slots)
		}
		_, ok := s.actions["D"]
		require.False(t, ok)
	})
}

func TestScheduler_Replenish_DenseSharedActions(t *testing.T) {
	tenantId := uuid.New()

	const (
		numWorkers = 50
		numActions = 200
	)

	workerIds := make([]uuid.UUID, numWorkers)
	activeWorkers := make([]*repo.ListActiveWorkersResult, numWorkers)
	for i := range workerIds {
		workerIds[i] = uuid.New()
		activeWorkers[i] = testWorker(workerIds[i])
	}

	actionIds := make([]string, numActions)
	for i := range actionIds {
		actionIds[i] = fmt.Sprintf("action-%03d", i)
	}

	ar := &mockAssignmentRepo{
		// every worker registers every action
		listActionsForWorkersFn: func(ctx context.Context, tenantId uuid.UUID, ids []uuid.UUID) ([]*sqlcv1.ListActionsForWorkersRow, error) {
			rows := make([]*sqlcv1.ListActionsForWorkersRow, 0, numWorkers*numActions)
			for _, wid := range workerIds {
				for _, aid := range actionIds {
					rows = append(rows, &sqlcv1.ListActionsForWorkersRow{
						WorkerId: wid,
						ActionId: sqlchelpers.TextFromStr(aid),
					})
				}
			}
			return rows, nil
		},
		listAvailableSlotsForWorkersFn: func(ctx context.Context, tenantId uuid.UUID, params sqlcv1.ListAvailableSlotsForWorkersParams) ([]*sqlcv1.ListAvailableSlotsForWorkersRow, error) {
			rows := make([]*sqlcv1.ListAvailableSlotsForWorkersRow, 0, len(params.Workerids))
			for _, wid := range params.Workerids {
				rows = append(rows, &sqlcv1.ListAvailableSlotsForWorkersRow{ID: wid, AvailableSlots: 1})
			}
			return rows, nil
		},
	}

	s := newTestScheduler(t, tenantId, ar)
	s.setWorkers(activeWorkers)

	require.NoError(t, s.replenish(context.Background(), true))

	onLoop(t, s, func() {
		require.Len(t, s.actions, numActions)

		for _, aid := range actionIds {
			a := s.actions[aid]
			require.NotNil(t, a, "action %s missing after replenish", aid)
			require.Len(t, a.workerIds, numWorkers, "action %s should index every worker", aid)
			require.Equal(t, numWorkers, a.lastReplenishedSlotCount)
			require.Equal(t, numWorkers, a.lastReplenishedWorkerCount)
		}
		require.Len(t, s.pools, numWorkers)
	})
}

func BenchmarkScheduler_Replenish_DenseSharedActions(b *testing.B) {
	tenantId := uuid.New()

	const (
		numWorkers = 100
		numActions = 500
	)

	workerIds := make([]uuid.UUID, numWorkers)
	activeWorkers := make([]*repo.ListActiveWorkersResult, numWorkers)
	for i := range workerIds {
		workerIds[i] = uuid.New()
		activeWorkers[i] = testWorker(workerIds[i])
	}

	actionIds := make([]string, numActions)
	for i := range actionIds {
		actionIds[i] = fmt.Sprintf("action-%04d", i)
	}

	actionRows := make([]*sqlcv1.ListActionsForWorkersRow, 0, numWorkers*numActions)
	for _, wid := range workerIds {
		for _, aid := range actionIds {
			actionRows = append(actionRows, &sqlcv1.ListActionsForWorkersRow{
				WorkerId: wid,
				ActionId: sqlchelpers.TextFromStr(aid),
			})
		}
	}

	ar := &mockAssignmentRepo{
		listActionsForWorkersFn: func(ctx context.Context, tenantId uuid.UUID, ids []uuid.UUID) ([]*sqlcv1.ListActionsForWorkersRow, error) {
			return actionRows, nil
		},
		listAvailableSlotsForWorkersFn: func(ctx context.Context, tenantId uuid.UUID, params sqlcv1.ListAvailableSlotsForWorkersParams) ([]*sqlcv1.ListAvailableSlotsForWorkersRow, error) {
			rows := make([]*sqlcv1.ListAvailableSlotsForWorkersRow, 0, len(params.Workerids))
			for _, wid := range params.Workerids {
				rows = append(rows, &sqlcv1.ListAvailableSlotsForWorkersRow{ID: wid, AvailableSlots: 1})
			}
			return rows, nil
		},
	}

	l := zerolog.Nop()
	sr := &mockSchedulerRepo{assignment: ar}
	cf := &sharedConfig{repo: sr, l: &l}
	s := newScheduler(cf, tenantId, nil, &Extensions{})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go s.run(ctx)

	s.setWorkers(activeWorkers)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := s.replenish(context.Background(), true); err != nil {
			b.Fatal(err)
		}
	}
}

func TestScheduler_Replenish_UpdatesAllActionIndexes(t *testing.T) {
	tenantId := uuid.New()
	workerId := uuid.New()

	ar := &mockAssignmentRepo{
		listActionsForWorkersFn: func(ctx context.Context, tenantId uuid.UUID, workerIds []uuid.UUID) ([]*sqlcv1.ListActionsForWorkersRow, error) {
			return []*sqlcv1.ListActionsForWorkersRow{
				{WorkerId: workerId, ActionId: sqlchelpers.TextFromStr("A")},
				{WorkerId: workerId, ActionId: sqlchelpers.TextFromStr("B")},
			}, nil
		},
		listAvailableSlotsForWorkersFn: func(ctx context.Context, tenantId uuid.UUID, params sqlcv1.ListAvailableSlotsForWorkersParams) ([]*sqlcv1.ListAvailableSlotsForWorkersRow, error) {
			require.Equal(t, repo.SlotTypeDefault, params.Slottype)
			return []*sqlcv1.ListAvailableSlotsForWorkersRow{
				{ID: workerId, AvailableSlots: 2},
			}, nil
		},
	}

	s := newTestScheduler(t, tenantId, ar)
	s.setWorkers([]*repo.ListActiveWorkersResult{testWorker(workerId)})

	// Seed actions so the replenish-decision logic runs: A's capacity is fully
	// consumed (triggers), B thinks it replenished many more slots than are
	// active (triggers).
	w := &worker{ListActiveWorkersResult: testWorker(workerId)}
	usedSlot := newSlot(w, repo.SlotTypeDefault)
	usedSlot.used = true

	aA := seedActionPools(t, s, "A", usedSlot)
	aB := seedActionPools(t, s, "B", newSlot(w, repo.SlotTypeDefault))
	onLoop(t, s, func() {
		aA.lastReplenishedSlotCount = 2
		aA.lastReplenishedWorkerCount = 1
		aB.lastReplenishedSlotCount = 100
		aB.lastReplenishedWorkerCount = 1
	})

	err := s.replenish(context.Background(), false)
	require.NoError(t, err)

	onLoop(t, s, func() {
		a := s.actions["A"]
		b := s.actions["B"]
		require.NotNil(t, a)
		require.NotNil(t, b)
		require.Equal(t, []uuid.UUID{workerId}, a.workerIds)
		require.Equal(t, []uuid.UUID{workerId}, b.workerIds)
		require.Len(t, s.pools[poolKey{workerId: workerId, slotType: repo.SlotTypeDefault}].slots, 2)
	})
}

func TestScheduler_TryAssignBatch_ParkedRetryAfterReplenish(t *testing.T) {
	tenantId := uuid.New()
	workerId := uuid.New()

	s := newTestScheduler(t, tenantId, &mockAssignmentRepo{})
	w := &worker{ListActiveWorkersResult: testWorker(workerId)}

	// action exists but its only slot is taken, and a replenish cycle is in flight
	sl := newSlot(w, repo.SlotTypeDefault)
	sl.used = true
	seedActionPools(t, s, "A", sl)
	onLoop(t, s, func() { s.replenishing = true })

	type batchResult struct {
		res []*assignSingleResult
		err error
	}
	resultCh := make(chan batchResult, 1)

	go func() {
		res, err := s.tryAssignBatch(context.Background(), "A", []*sqlcv1.V1QueueItem{testQI(tenantId, "A", 1)}, nil, nil, nil, nil, nil)
		resultCh <- batchResult{res, err}
	}()

	// the batch misses and parks behind the in-flight cycle
	require.Eventually(t, func() bool {
		parked := false
		onLoop(t, s, func() { parked = len(s.afterReplenish) == 1 })
		return parked
	}, time.Second, time.Millisecond)

	select {
	case <-resultCh:
		t.Fatal("batch completed while parked")
	default:
	}

	// the cycle ends and capacity arrives: the parked retry must assign
	onLoop(t, s, func() {
		sl.pool.release(sl)
		s.endReplenishCycle()
	})

	select {
	case r := <-resultCh:
		require.NoError(t, r.err)
		require.Len(t, r.res, 1)
		require.True(t, r.res[0].succeeded, "parked retry should assign once capacity lands")
		require.False(t, r.res[0].noSlots)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for parked batch to complete")
	}
}

func TestScheduler_TryAssignBatch_ParkedRetryTimesOut(t *testing.T) {
	tenantId := uuid.New()
	workerId := uuid.New()

	s := newTestScheduler(t, tenantId, &mockAssignmentRepo{})
	w := &worker{ListActiveWorkersResult: testWorker(workerId)}

	sl := newSlot(w, repo.SlotTypeDefault)
	sl.used = true
	seedActionPools(t, s, "A", sl)
	onLoop(t, s, func() { s.replenishing = true })

	start := time.Now()
	res, err := s.tryAssignBatch(context.Background(), "A", []*sqlcv1.V1QueueItem{testQI(tenantId, "A", 1)}, nil, nil, nil, nil, nil)
	waited := time.Since(start)

	// the cycle never ends: the park must give up after its bounded wait and
	// report the miss rather than blocking behind a stalled replenish
	require.NoError(t, err)
	require.Len(t, res, 1)
	require.False(t, res[0].succeeded)
	require.True(t, res[0].noSlots)
	require.GreaterOrEqual(t, waited, parkedAssignRetryTimeout)
	require.Less(t, waited, 10*parkedAssignRetryTimeout)

	// the cycle eventually ends with capacity available; the timed-out retry
	// must be a no-op — the result already belongs to the caller
	onLoop(t, s, func() {
		sl.pool.release(sl)
		s.endReplenishCycle()
	})

	onLoop(t, s, func() {
		require.Empty(t, s.unackedSlots, "timed-out parked retry must not assign")
		require.Equal(t, 1, len(sl.pool.free))
	})
	require.True(t, res[0].noSlots)
}
