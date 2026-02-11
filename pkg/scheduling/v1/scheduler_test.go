//go:build !e2e && !load && !rampup && !integration

package v1

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	repo "github.com/hatchet-dev/hatchet/pkg/repository"
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

func newTestScheduler(t *testing.T, tenantId uuid.UUID, ar repo.AssignmentRepository) *Scheduler {
	t.Helper()

	l := zerolog.Nop()

	sr := &mockSchedulerRepo{assignment: ar}
	cf := &sharedConfig{
		repo: sr,
		l:    &l,
	}

	// rate limiter not needed for most tests; can be set by the caller if required.
	return newScheduler(cf, tenantId, nil, &Extensions{})
}

func testWorker(id uuid.UUID) *repo.ListActiveWorkersResult {
	return &repo.ListActiveWorkersResult{
		ID:     id,
		Name:   "w",
		Labels: nil,
	}
}

func actionWithSlots(actionId string, slots ...*slot) (*action, error) {
	a := &action{
		actionId: actionId,
		slots:    slots,
		// populate index for tests; production code builds it in replenish.
		slotsByTypeAndWorkerId: make(map[string]map[uuid.UUID][]*slot),
	}

	for _, sl := range slots {
		slotType, err := sl.getSlotType()
		if err != nil {
			return nil, fmt.Errorf("getSlotType failed: %w", err)
		}

		workerId := sl.getWorkerId()

		if _, ok := a.slotsByTypeAndWorkerId[slotType]; !ok {
			a.slotsByTypeAndWorkerId[slotType] = make(map[uuid.UUID][]*slot)
		}
		a.slotsByTypeAndWorkerId[slotType][workerId] = append(a.slotsByTypeAndWorkerId[slotType][workerId], sl)
	}

	return a, nil
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
	return pgtype.Timestamp{Time: tm, Valid: true}
}

func requireEventually(t *testing.T, dur time.Duration, f func() bool) {
	t.Helper()
	deadline := time.Now().Add(dur)
	for time.Now().Before(deadline) {
		if f() {
			return
		}
		time.Sleep(1 * time.Millisecond)
	}
	require.True(t, f())
}

func TestScheduler_AckNack(t *testing.T) {
	tenantId := uuid.New()
	workerId := uuid.New()

	s := newTestScheduler(t, tenantId, &mockAssignmentRepo{})
	s.setWorkers([]*repo.ListActiveWorkersResult{testWorker(workerId)})

	w := &worker{ListActiveWorkersResult: testWorker(workerId)}
	sl := newSlot(w, newSlotMeta([]string{"A"}, repo.SlotTypeDefault))
	require.True(t, sl.use(nil, nil))

	s.unackedSlots[123] = &assignedSlots{slots: []*slot{sl}}

	s.ack([]int{123, 999})

	require.True(t, sl.ackd)
	require.NotNil(t, sl.expiresAt)
	require.Empty(t, s.unackedSlots)

	// nack should reset used=false and remove from unacked
	sl2 := newSlot(w, newSlotMeta([]string{"A"}, repo.SlotTypeDefault))
	require.True(t, sl2.use(nil, nil))
	s.unackedSlots[777] = &assignedSlots{slots: []*slot{sl2}}

	s.nack([]int{777})

	require.True(t, sl2.ackd)
	require.False(t, sl2.used)
	require.Empty(t, s.unackedSlots)
}

func TestScheduler_SetWorkers_GetWorkers(t *testing.T) {
	tenantId := uuid.New()

	s := newTestScheduler(t, tenantId, &mockAssignmentRepo{})

	w1 := testWorker(uuid.New())
	w2 := testWorker(uuid.New())

	s.setWorkers([]*repo.ListActiveWorkersResult{w1, w2})

	got := s.getWorkers()
	require.Len(t, got, 2)
	require.Equal(t, w1.ID, got[w1.ID].ID)
	require.Equal(t, w2.ID, got[w2.ID].ID)
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

func TestSelectSlotsForWorker_SkipsInactive(t *testing.T) {
	workerId := uuid.New()
	w := &worker{ListActiveWorkersResult: testWorker(workerId)}

	s1 := newSlot(w, newSlotMeta([]string{"A"}, repo.SlotTypeDefault))
	require.True(t, s1.use(nil, nil)) // used => inactive

	s2 := newSlot(w, newSlotMeta([]string{"A"}, repo.SlotTypeDefault))
	// expire s2
	past := time.Now().Add(-1 * time.Second)
	s2.mu.Lock()
	s2.expiresAt = &past
	s2.mu.Unlock()

	s3 := newSlot(w, newSlotMeta([]string{"A"}, repo.SlotTypeDefault))

	slotsByTypeAndWorkerId := map[string]map[uuid.UUID][]*slot{
		repo.SlotTypeDefault: {workerId: {s1, s2, s3}},
	}

	selected, ok := selectSlotsForWorker(
		slotsByTypeAndWorkerId,
		workerId,
		map[string]int32{repo.SlotTypeDefault: 1},
	)
	require.True(t, ok)
	require.Len(t, selected, 1)
	require.Same(t, s3, selected[0])
}

func TestScheduler_TryAssignSingleton_RingWraparound(t *testing.T) {
	tenantId := uuid.New()
	workerId1 := uuid.New()
	workerId2 := uuid.New()

	s := newTestScheduler(t, tenantId, &mockAssignmentRepo{})

	w1 := &worker{ListActiveWorkersResult: testWorker(workerId1)}
	w2 := &worker{ListActiveWorkersResult: testWorker(workerId2)}

	// s1 is used/inactive, s2 is active
	s1 := newSlot(w1, newSlotMeta([]string{"A"}, repo.SlotTypeDefault))
	require.True(t, s1.use(nil, nil))
	s2 := newSlot(w2, newSlotMeta([]string{"A"}, repo.SlotTypeDefault))

	a, err := actionWithSlots("A", s1, s2)
	require.NoError(t, err)
	req := map[string]int32{repo.SlotTypeDefault: 1}

	qi := testQI(tenantId, "A", 1)
	res, err := s.tryAssignSingleton(context.Background(), qi, a, []*slot{s1, s2}, 1, nil, req, func() {}, func() {})
	require.NoError(t, err)
	require.True(t, res.succeeded)
	require.False(t, res.noSlots)
	require.Equal(t, workerId2, res.workerId)
	require.NotZero(t, res.ackId)

	s.unackedMu.Lock()
	_, ok := s.unackedSlots[res.ackId]
	s.unackedMu.Unlock()
	require.True(t, ok)
}

func TestScheduler_TryAssignSingleton_NoSlots(t *testing.T) {
	tenantId := uuid.New()
	workerId := uuid.New()

	s := newTestScheduler(t, tenantId, &mockAssignmentRepo{})
	w := &worker{ListActiveWorkersResult: testWorker(workerId)}

	s1 := newSlot(w, newSlotMeta([]string{"A"}, repo.SlotTypeDefault))
	require.True(t, s1.use(nil, nil))

	a, err := actionWithSlots("A", s1)
	require.NoError(t, err)
	req := map[string]int32{repo.SlotTypeDefault: 1}

	qi := testQI(tenantId, "A", 1)
	res, err := s.tryAssignSingleton(context.Background(), qi, a, []*slot{s1}, 0, nil, req, func() {}, func() {})
	require.NoError(t, err)
	require.False(t, res.succeeded)
	require.True(t, res.noSlots)
}

func TestScheduler_TryAssignSingleton_StickyHardForcesRanking(t *testing.T) {
	tenantId := uuid.New()
	desiredWorkerId := uuid.New()
	otherWorkerId := uuid.New()

	s := newTestScheduler(t, tenantId, &mockAssignmentRepo{})

	wDesired := &worker{ListActiveWorkersResult: testWorker(desiredWorkerId)}
	wOther := &worker{ListActiveWorkersResult: testWorker(otherWorkerId)}

	// Put desired slot second; with HARD sticky it should still be selected.
	otherSlot := newSlot(wOther, newSlotMeta([]string{"A"}, repo.SlotTypeDefault))
	desiredSlot := newSlot(wDesired, newSlotMeta([]string{"A"}, repo.SlotTypeDefault))

	a, err := actionWithSlots("A", otherSlot, desiredSlot)
	require.NoError(t, err)
	req := map[string]int32{repo.SlotTypeDefault: 1}

	qi := testQI(tenantId, "A", 1)
	qi.Sticky = sqlcv1.V1StickyStrategyHARD
	qi.DesiredWorkerID = &desiredWorkerId

	res, err := s.tryAssignSingleton(context.Background(), qi, a, []*slot{otherSlot, desiredSlot}, 1, nil, req, func() {}, func() {})
	require.NoError(t, err)
	require.True(t, res.succeeded)
	require.Equal(t, desiredWorkerId, res.workerId)
}

func TestScheduler_TryAssignSingleton_RateLimitAckIsWiredIntoSlotAck(t *testing.T) {
	tenantId := uuid.New()
	workerId := uuid.New()

	s := newTestScheduler(t, tenantId, &mockAssignmentRepo{})
	w := &worker{ListActiveWorkersResult: testWorker(workerId)}

	sl := newSlot(w, newSlotMeta([]string{"A"}, repo.SlotTypeDefault))
	a, err := actionWithSlots("A", sl)
	require.NoError(t, err)
	req := map[string]int32{repo.SlotTypeDefault: 1}
	qi := testQI(tenantId, "A", 1)

	ackCount := 0
	rlAck := func() { ackCount++ }

	res, err := s.tryAssignSingleton(context.Background(), qi, a, []*slot{sl}, 0, nil, req, rlAck, func() {})
	require.NoError(t, err)
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

	res, _, err := s.tryAssignBatch(context.Background(), "missing", qis, 0, nil, nil, nil)
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
			return nil, nil
		},
		listAvailableSlotsForWorkersFn: func(ctx context.Context, tenantId uuid.UUID, params sqlcv1.ListAvailableSlotsForWorkersParams) ([]*sqlcv1.ListAvailableSlotsForWorkersRow, error) {
			return nil, nil
		},
	})

	// hold replenish lock to force TryLock() failure
	s.replenishMu.Lock()
	defer s.replenishMu.Unlock()

	require.NoError(t, s.replenish(context.Background(), false))
}

func TestScheduler_Replenish_SkipsIfCannotAcquireActionsLock(t *testing.T) {
	tenantId := uuid.New()
	s := newTestScheduler(t, tenantId, &mockAssignmentRepo{
		listActionsForWorkersFn: func(ctx context.Context, tenantId uuid.UUID, workerIds []uuid.UUID) ([]*sqlcv1.ListActionsForWorkersRow, error) {
			t.Fatalf("should not hit repo when actions lock can't be acquired")
			return nil, nil
		},
		listAvailableSlotsForWorkersFn: func(ctx context.Context, tenantId uuid.UUID, params sqlcv1.ListAvailableSlotsForWorkersParams) ([]*sqlcv1.ListAvailableSlotsForWorkersRow, error) {
			t.Fatalf("should not hit repo when actions lock can't be acquired")
			return nil, nil
		},
	})

	// Hold the actions write lock so TryLock fails (mustReplenish=false path).
	s.actionsMu.Lock()
	defer s.actionsMu.Unlock()

	require.NoError(t, s.replenish(context.Background(), false))
}

func TestScheduler_Replenish_DoesNotLockUnackedMuBeforeActionLocks(t *testing.T) {
	tenantId := uuid.New()
	workerId := uuid.New()

	workerSlotConfigsCalled := make(chan struct{})

	ar := &mockAssignmentRepo{
		listActionsForWorkersFn: func(ctx context.Context, gotTenantId uuid.UUID, workerIds []uuid.UUID) ([]*sqlcv1.ListActionsForWorkersRow, error) {
			require.Equal(t, tenantId, gotTenantId)
			require.Len(t, workerIds, 1)
			require.Equal(t, workerId, workerIds[0])

			return []*sqlcv1.ListActionsForWorkersRow{
				{
					WorkerId: workerId,
					ActionId: pgtype.Text{String: "A", Valid: true},
				},
			}, nil
		},
		listWorkerSlotConfigsFn: func(ctx context.Context, gotTenantId uuid.UUID, workerIds []uuid.UUID) ([]*sqlcv1.ListWorkerSlotConfigsRow, error) {
			require.Equal(t, tenantId, gotTenantId)
			require.Len(t, workerIds, 1)
			require.Equal(t, workerId, workerIds[0])

			select {
			case <-workerSlotConfigsCalled:
				// already closed
			default:
				close(workerSlotConfigsCalled)
			}

			return []*sqlcv1.ListWorkerSlotConfigsRow{
				{
					WorkerID: workerId,
					SlotType: repo.SlotTypeDefault,
					MaxUnits: 1,
				},
			}, nil
		},
		listAvailableSlotsForWorkersFn: func(ctx context.Context, gotTenantId uuid.UUID, params sqlcv1.ListAvailableSlotsForWorkersParams) ([]*sqlcv1.ListAvailableSlotsForWorkersRow, error) {
			require.Equal(t, tenantId, gotTenantId)
			require.Equal(t, repo.SlotTypeDefault, params.Slottype)
			require.Len(t, params.Workerids, 1)
			require.Equal(t, workerId, params.Workerids[0])

			return []*sqlcv1.ListAvailableSlotsForWorkersRow{
				{
					ID:             workerId,
					AvailableSlots: 1,
				},
			}, nil
		},
	}

	s := newTestScheduler(t, tenantId, ar)
	s.setWorkers([]*repo.ListActiveWorkersResult{testWorker(workerId)})

	// Pre-create an action so replenish includes it in orderedLock(actionsToLock).
	w := &worker{ListActiveWorkersResult: testWorker(workerId)}
	sl := newSlot(w, newSlotMeta([]string{"A"}, repo.SlotTypeDefault))
	a, err := actionWithSlots("A", sl)
	require.NoError(t, err)
	s.actions["A"] = a

	a.mu.Lock()

	replenishDone := make(chan error, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		replenishDone <- s.replenish(ctx, true)
	}()

	select {
	case <-workerSlotConfigsCalled:
	case <-time.After(2 * time.Second):
		a.mu.Unlock()
		t.Fatalf("timed out waiting for replenish to call ListWorkerSlotConfigs")
	}

	// While replenish is blocked trying to acquire action locks, it must not hold unackedMu.
	// If lock order ever regresses (unackedMu before action.mu), this will fail.
	deadline := time.Now().Add(50 * time.Millisecond)
	for time.Now().Before(deadline) {
		if ok := s.unackedMu.TryLock(); ok {
			s.unackedMu.Unlock()
		} else {
			a.mu.Unlock()
			t.Fatalf("replenish acquired unackedMu while action.mu was held (lock order violation)")
		}
		time.Sleep(1 * time.Millisecond)
	}

	a.mu.Unlock()

	select {
	case err := <-replenishDone:
		require.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for replenish to complete (possible deadlock)")
	}
}

func TestScheduler_TryAssignBatch_AssignsUntilExhausted(t *testing.T) {
	tenantId := uuid.New()
	workerId := uuid.New()

	s := newTestScheduler(t, tenantId, &mockAssignmentRepo{})
	w := &worker{ListActiveWorkersResult: testWorker(workerId)}

	// two total slots
	sl1 := newSlot(w, newSlotMeta([]string{"A"}, repo.SlotTypeDefault))
	sl2 := newSlot(w, newSlotMeta([]string{"A"}, repo.SlotTypeDefault))

	actA, err := actionWithSlots("A", sl1, sl2)
	require.NoError(t, err)
	s.actions["A"] = actA

	qis := []*sqlcv1.V1QueueItem{
		testQI(tenantId, "A", 1),
		testQI(tenantId, "A", 2),
		testQI(tenantId, "A", 3),
	}

	res, newOffset, err := s.tryAssignBatch(context.Background(), "A", qis, 0, map[uuid.UUID][]*sqlcv1.GetDesiredLabelsRow{}, map[uuid.UUID]map[string]int32{}, nil)
	require.NoError(t, err)
	require.Equal(t, 3, newOffset)

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
	sl := newSlot(w, newSlotMeta([]string{"A"}, repo.SlotTypeDefault))
	actA, err := actionWithSlots("A", sl)
	require.NoError(t, err)
	s.actions["A"] = actA

	qi := testQI(tenantId, "A", 100)
	qis := []*sqlcv1.V1QueueItem{qi}

	rls := map[int64]map[string]int32{
		qi.TaskID: {"k": 1},
	}

	res, _, err := s.tryAssignBatch(context.Background(), "A", qis, 0, nil, map[uuid.UUID]map[string]int32{}, rls)
	require.NoError(t, err)
	require.Len(t, res, 1)
	require.False(t, res[0].succeeded)
	require.NotNil(t, res[0].rateLimitResult)
	require.False(t, res[0].noSlots)
}

func TestScheduler_TryAssign_GroupsAndFiltersTimedOut(t *testing.T) {
	tenantId := uuid.New()
	workerId := uuid.New()

	s := newTestScheduler(t, tenantId, &mockAssignmentRepo{})
	w := &worker{ListActiveWorkersResult: testWorker(workerId)}

	// A has 1 slot, B has 1 slot
	actA, err := actionWithSlots("A", newSlot(w, newSlotMeta([]string{"A"}, repo.SlotTypeDefault)))
	require.NoError(t, err)
	s.actions["A"] = actA
	actB, err := actionWithSlots("B", newSlot(w, newSlotMeta([]string{"B"}, repo.SlotTypeDefault)))
	require.NoError(t, err)
	s.actions["B"] = actB

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

func TestScheduler_GetSnapshotInput_BestEffortTryLock(t *testing.T) {
	tenantId := uuid.New()
	s := newTestScheduler(t, tenantId, &mockAssignmentRepo{})

	// Hold write lock so TryRLock fails.
	s.actionsMu.Lock()
	defer s.actionsMu.Unlock()

	in, ok := s.getSnapshotInput(false)
	require.False(t, ok)
	require.Nil(t, in)
}

func TestScheduler_GetSnapshotInput_DedupSlotsAcrossActions(t *testing.T) {
	tenantId := uuid.New()
	workerId := uuid.New()
	s := newTestScheduler(t, tenantId, &mockAssignmentRepo{})

	s.setWorkers([]*repo.ListActiveWorkersResult{{ID: workerId, Name: "w1", Labels: nil}})

	w := &worker{ListActiveWorkersResult: testWorker(workerId)}
	sharedSlot := newSlot(w, newSlotMeta([]string{"A", "B"}, repo.SlotTypeDefault))
	require.True(t, sharedSlot.use(nil, nil)) // used
	unusedSlot := newSlot(w, newSlotMeta([]string{"A", "B"}, repo.SlotTypeDefault))

	actA, err := actionWithSlots("A", sharedSlot, unusedSlot)
	require.NoError(t, err)
	s.actions["A"] = actA
	actB, err := actionWithSlots("B", sharedSlot, unusedSlot) // duplicate pointers
	require.NoError(t, err)
	s.actions["B"] = actB

	in, ok := s.getSnapshotInput(true)
	require.True(t, ok)
	require.NotNil(t, in)
	require.Len(t, in.Workers, 1)
	require.Equal(t, workerId, in.Workers[workerId].WorkerId)

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
	tenantId := uuid.New()
	s := newTestScheduler(t, tenantId, &mockAssignmentRepo{})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// should not block or panic even if canceled
	s.start(ctx)
}

func TestSelectSlotsForWorker_MissingTypeOrInsufficientUnitsFails(t *testing.T) {
	workerId := uuid.New()
	w := &worker{ListActiveWorkersResult: testWorker(workerId)}

	one := newSlot(w, newSlotMeta([]string{"A"}, repo.SlotTypeDefault))

	slotsByTypeAndWorkerId := map[string]map[uuid.UUID][]*slot{
		repo.SlotTypeDefault: {workerId: {one}},
	}

	_, ok := selectSlotsForWorker(slotsByTypeAndWorkerId, workerId, map[string]int32{repo.SlotTypeDurable: 1})
	require.False(t, ok)

	_, ok = selectSlotsForWorker(slotsByTypeAndWorkerId, workerId, map[string]int32{repo.SlotTypeDefault: 2})
	require.False(t, ok)
}

func TestFindAssignableSlots_MultiUnitSameType(t *testing.T) {
	workerId := uuid.New()
	w := &worker{ListActiveWorkersResult: testWorker(workerId)}

	s1 := newSlot(w, newSlotMeta([]string{"A"}, repo.SlotTypeDefault))
	s2 := newSlot(w, newSlotMeta([]string{"A"}, repo.SlotTypeDefault))
	s3 := newSlot(w, newSlotMeta([]string{"A"}, repo.SlotTypeDefault))
	require.True(t, s3.use(nil, nil)) // used; ensure not selected

	a, err := actionWithSlots("A", s1, s2, s3)
	require.NoError(t, err)

	assigned := findAssignableSlots(a.slots, a, map[string]int32{repo.SlotTypeDefault: 2}, nil, nil)
	require.NotNil(t, assigned)
	require.Len(t, assigned.slots, 2)
	require.Equal(t, workerId, assigned.workerId())

	// both selected are now used
	for _, sl := range assigned.slots {
		require.True(t, sl.isUsed())
	}
}

func TestFindAssignableSlots_MultiType(t *testing.T) {
	workerId := uuid.New()
	w := &worker{ListActiveWorkersResult: testWorker(workerId)}

	def := newSlot(w, newSlotMeta([]string{"A"}, repo.SlotTypeDefault))
	dur := newSlot(w, newSlotMeta([]string{"A"}, repo.SlotTypeDurable))

	a, err := actionWithSlots("A", def, dur)
	require.NoError(t, err)

	assigned := findAssignableSlots(a.slots, a, map[string]int32{repo.SlotTypeDefault: 1, repo.SlotTypeDurable: 1}, nil, nil)
	require.NotNil(t, assigned)
	require.Len(t, assigned.slots, 2)

	gotTypes := map[string]bool{}
	for _, sl := range assigned.slots {
		slotType, err := sl.getSlotType()
		require.NoError(t, err)
		gotTypes[slotType] = true
	}
	require.True(t, gotTypes[repo.SlotTypeDefault])
	require.True(t, gotTypes[repo.SlotTypeDurable])
}

func TestUseSelectedSlots_PartialAllocationRollback(t *testing.T) {
	workerId := uuid.New()
	w := &worker{ListActiveWorkersResult: testWorker(workerId)}

	s1 := newSlot(w, newSlotMeta([]string{"A"}, repo.SlotTypeDefault))
	s2 := newSlot(w, newSlotMeta([]string{"A"}, repo.SlotTypeDefault))

	// Simulate a concurrent take of the second slot after selection but before useSelectedSlots.
	require.True(t, s2.use(nil, nil))

	used, ok := useSelectedSlots([]*slot{s1, s2})
	require.False(t, ok)
	require.Nil(t, used)

	// rollback should have nacked s1 (used=false)
	require.False(t, s1.isUsed())
	// s2 was taken by the simulated concurrent use
	require.True(t, s2.isUsed())
}

func TestScheduler_Nack_CallsRateLimitNackOnce(t *testing.T) {
	tenantId := uuid.New()
	workerId := uuid.New()
	s := newTestScheduler(t, tenantId, &mockAssignmentRepo{})

	w := &worker{ListActiveWorkersResult: testWorker(workerId)}
	sl := newSlot(w, newSlotMeta([]string{"A"}, repo.SlotTypeDefault))
	require.True(t, sl.use(nil, nil))

	nackCount := 0
	as := &assignedSlots{
		slots:         []*slot{sl},
		rateLimitNack: func() { nackCount++ },
	}

	s.unackedSlots[1] = as
	s.nack([]int{1})

	require.Equal(t, 1, nackCount)
	require.False(t, sl.isUsed())
}

func TestScheduler_Replenish_MultipleSlotTypes_CallsRepoPerTypeAndPopulatesSlotsByWorker(t *testing.T) {
	tenantId := uuid.New()
	workerId := uuid.New()

	called := map[string]int{}

	ar := &mockAssignmentRepo{
		listActionsForWorkersFn: func(ctx context.Context, tenantId uuid.UUID, workerIds []uuid.UUID) ([]*sqlcv1.ListActionsForWorkersRow, error) {
			return []*sqlcv1.ListActionsForWorkersRow{
				{WorkerId: workerId, ActionId: pgtype.Text{String: "A", Valid: true}},
			}, nil
		},
		listWorkerSlotConfigsFn: func(ctx context.Context, tenantId uuid.UUID, workerIds []uuid.UUID) ([]*sqlcv1.ListWorkerSlotConfigsRow, error) {
			return []*sqlcv1.ListWorkerSlotConfigsRow{
				{WorkerID: workerId, SlotType: repo.SlotTypeDefault, MaxUnits: 2},
				{WorkerID: workerId, SlotType: repo.SlotTypeDurable, MaxUnits: 2},
			}, nil
		},
		listAvailableSlotsForWorkersFn: func(ctx context.Context, tenantId uuid.UUID, params sqlcv1.ListAvailableSlotsForWorkersParams) ([]*sqlcv1.ListAvailableSlotsForWorkersRow, error) {
			called[params.Slottype]++
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

	a := s.actions["A"]
	require.NotNil(t, a)
	require.Len(t, a.slots, 4)

	countByType := map[string]int{}
	for _, sl := range a.slots {
		if sl.getWorkerId() != workerId {
			continue
		}
		slotType, err := sl.getSlotType()
		require.NoError(t, err)
		countByType[slotType]++
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
				{WorkerId: workerId, ActionId: pgtype.Text{String: "A", Valid: true}},
			}, nil
		},
		listWorkerSlotConfigsFn: func(ctx context.Context, tenantId uuid.UUID, workerIds []uuid.UUID) ([]*sqlcv1.ListWorkerSlotConfigsRow, error) {
			return []*sqlcv1.ListWorkerSlotConfigsRow{
				{WorkerID: workerId, SlotType: repo.SlotTypeDefault, MaxUnits: 2},
				{WorkerID: workerId, SlotType: repo.SlotTypeDurable, MaxUnits: 2},
			}, nil
		},
		listAvailableSlotsForWorkersFn: func(ctx context.Context, tenantId uuid.UUID, params sqlcv1.ListAvailableSlotsForWorkersParams) ([]*sqlcv1.ListAvailableSlotsForWorkersRow, error) {
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

	// Seed one unacked durable slot; should only reduce *durable* new-slot count.
	w := &worker{ListActiveWorkersResult: testWorker(workerId)}
	unackedDurable := newSlot(w, newSlotMeta([]string{"A"}, repo.SlotTypeDurable))
	require.True(t, unackedDurable.use(nil, nil))
	s.unackedSlots[1] = &assignedSlots{slots: []*slot{unackedDurable}}

	err := s.replenish(context.Background(), true)
	require.NoError(t, err)

	a := s.actions["A"]
	require.NotNil(t, a)
	countDefault := 0
	countDurable := 0
	foundUnacked := false
	for _, sl := range a.slots {
		if sl.getWorkerId() != workerId {
			continue
		}
		slotType, err := sl.getSlotType()
		require.NoError(t, err)

		switch slotType {
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
}

func TestScheduler_Replenish_PropagatesRepoErrors(t *testing.T) {
	tenantId := uuid.New()
	workerId := uuid.New()
	sentinel := fmt.Errorf("boom")

	t.Run("ListActionsForWorkers", func(t *testing.T) {
		s := newTestScheduler(t, tenantId, &mockAssignmentRepo{
			listActionsForWorkersFn: func(ctx context.Context, tenantId uuid.UUID, workerIds []uuid.UUID) ([]*sqlcv1.ListActionsForWorkersRow, error) {
				return nil, sentinel
			},
		})
		s.setWorkers([]*repo.ListActiveWorkersResult{testWorker(workerId)})
		err := s.replenish(context.Background(), true)
		require.ErrorIs(t, err, sentinel)
	})

	t.Run("ListWorkerSlotConfigs", func(t *testing.T) {
		s := newTestScheduler(t, tenantId, &mockAssignmentRepo{
			listActionsForWorkersFn: func(ctx context.Context, tenantId uuid.UUID, workerIds []uuid.UUID) ([]*sqlcv1.ListActionsForWorkersRow, error) {
				return []*sqlcv1.ListActionsForWorkersRow{
					{WorkerId: workerId, ActionId: pgtype.Text{String: "A", Valid: true}},
				}, nil
			},
			listWorkerSlotConfigsFn: func(ctx context.Context, tenantId uuid.UUID, workerIds []uuid.UUID) ([]*sqlcv1.ListWorkerSlotConfigsRow, error) {
				return nil, sentinel
			},
		})
		s.setWorkers([]*repo.ListActiveWorkersResult{testWorker(workerId)})
		err := s.replenish(context.Background(), true)
		require.ErrorIs(t, err, sentinel)
	})

	t.Run("ListAvailableSlotsForWorkers", func(t *testing.T) {
		s := newTestScheduler(t, tenantId, &mockAssignmentRepo{
			listActionsForWorkersFn: func(ctx context.Context, tenantId uuid.UUID, workerIds []uuid.UUID) ([]*sqlcv1.ListActionsForWorkersRow, error) {
				return []*sqlcv1.ListActionsForWorkersRow{
					{WorkerId: workerId, ActionId: pgtype.Text{String: "A", Valid: true}},
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
	})
}

func TestScheduler_Replenish_CreatesActionAndSlots(t *testing.T) {
	tenantId := uuid.New()
	workerId := uuid.New()

	ar := &mockAssignmentRepo{
		listActionsForWorkersFn: func(ctx context.Context, tenantId uuid.UUID, workerIds []uuid.UUID) ([]*sqlcv1.ListActionsForWorkersRow, error) {
			return []*sqlcv1.ListActionsForWorkersRow{
				{WorkerId: workerId, ActionId: pgtype.Text{String: "A", Valid: true}},
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

	a, ok := s.actions["A"]
	require.True(t, ok)
	require.NotNil(t, a)
	require.Len(t, a.slots, 3)
	require.Equal(t, 3, a.lastReplenishedSlotCount)
	require.Equal(t, 1, a.lastReplenishedWorkerCount)

	for _, sl := range a.slots {
		require.Equal(t, workerId, sl.getWorkerId())
		slotType, err := sl.getSlotType()
		require.NoError(t, err)
		require.Equal(t, repo.SlotTypeDefault, slotType)
	}
}

func TestScheduler_Replenish_CleansExpiredSlotsWhenNoNewSlotsLoaded(t *testing.T) {
	tenantId := uuid.New()
	workerId := uuid.New()

	ar := &mockAssignmentRepo{
		listActionsForWorkersFn: func(ctx context.Context, tenantId uuid.UUID, workerIds []uuid.UUID) ([]*sqlcv1.ListActionsForWorkersRow, error) {
			return []*sqlcv1.ListActionsForWorkersRow{
				{WorkerId: workerId, ActionId: pgtype.Text{String: "A", Valid: true}},
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

	w := &worker{ListActiveWorkersResult: testWorker(workerId)}
	expired := newSlot(w, newSlotMeta([]string{"A"}, repo.SlotTypeDefault))
	past := time.Now().Add(-1 * time.Second)
	expired.mu.Lock()
	expired.expiresAt = &past
	expired.mu.Unlock()

	used := newSlot(w, newSlotMeta([]string{"A"}, repo.SlotTypeDefault))
	require.True(t, used.use(nil, nil))

	actA, err := actionWithSlots("A", expired, used)
	require.NoError(t, err)
	s.actions["A"] = actA
	s.actions["A"].lastReplenishedSlotCount = 2

	err = s.replenish(context.Background(), false)
	require.NoError(t, err)

	a := s.actions["A"]
	require.NotNil(t, a)
	require.Len(t, a.slots, 1)
	require.Same(t, used, a.slots[0])
}

func TestScheduler_Replenish_UpdatesAllWorkerActionsForLockSafety(t *testing.T) {
	tenantId := uuid.New()
	workerId := uuid.New()

	ar := &mockAssignmentRepo{
		listActionsForWorkersFn: func(ctx context.Context, tenantId uuid.UUID, workerIds []uuid.UUID) ([]*sqlcv1.ListActionsForWorkersRow, error) {
			return []*sqlcv1.ListActionsForWorkersRow{
				{WorkerId: workerId, ActionId: pgtype.Text{String: "A", Valid: true}},
				{WorkerId: workerId, ActionId: pgtype.Text{String: "B", Valid: true}},
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

	// Seed actions so FUNCTION 1 decision logic runs.
	w := &worker{ListActiveWorkersResult: testWorker(workerId)}
	usedSlot := newSlot(w, newSlotMeta([]string{"A", "B"}, repo.SlotTypeDefault))
	require.True(t, usedSlot.use(nil, nil))

	actA, err := actionWithSlots("A", usedSlot)
	require.NoError(t, err)
	s.actions["A"] = actA
	s.actions["A"].lastReplenishedSlotCount = 2
	s.actions["A"].lastReplenishedWorkerCount = 1

	actB, err := actionWithSlots("B", newSlot(w, newSlotMeta([]string{"A", "B"}, repo.SlotTypeDefault)))
	require.NoError(t, err)
	s.actions["B"] = actB
	s.actions["B"].lastReplenishedSlotCount = 100
	s.actions["B"].lastReplenishedWorkerCount = 1

	err = s.replenish(context.Background(), false)
	require.NoError(t, err)

	a := s.actions["A"]
	b := s.actions["B"]
	require.NotNil(t, a)
	require.NotNil(t, b)
	require.Len(t, a.slots, 2)
	require.Len(t, b.slots, 2)

	// Compare as sets (order is randomized per action).
	setA := map[*slot]bool{}
	for _, sl := range a.slots {
		setA[sl] = true
	}
	for _, sl := range b.slots {
		require.True(t, setA[sl], "expected slot pointers shared across actions for same worker capacity")
	}
}
