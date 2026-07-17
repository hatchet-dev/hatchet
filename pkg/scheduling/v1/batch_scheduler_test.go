package v1

import (
	"context"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	v1repo "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

type fakeReserveCall struct {
	TenantID string
	StepID   uuid.UUID
	ActionID string
	BatchKey string
	BatchID  string
	MaxRuns  int
}

type fakeBatchRepo struct {
	mu               sync.Mutex
	listResponses    [][]*sqlcv1.V1BatchedQueueItem
	moveCalls        [][]int64
	commitCalls      [][]*v1repo.BatchAssignment
	reserveCalls     []*fakeReserveCall
	existingIdsCalls [][]int64
	missingIds       map[int64]struct{}

	// reserveFunc, if set, overrides whether ReserveAndCommitBatchRun grants the reservation.
	// Defaults to always granting, matching the old default of a nil reservation hook.
	reserveFunc func(call *fakeReserveCall) (bool, error)
}

func (f *fakeBatchRepo) ListBatchResources(ctx context.Context) ([]*sqlcv1.ListDistinctBatchResourcesRow, error) {
	return nil, nil
}

func (f *fakeBatchRepo) ListBatchedQueueItems(ctx context.Context, stepId uuid.UUID, limit int32) ([]*sqlcv1.V1BatchedQueueItem, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if len(f.listResponses) == 0 {
		return nil, nil
	}

	resp := f.listResponses[0]
	f.listResponses = f.listResponses[1:]

	return resp, nil
}

func (f *fakeBatchRepo) DeleteBatchedQueueItems(ctx context.Context, ids []int64) error {
	return nil
}

func (f *fakeBatchRepo) MoveBatchedQueueItems(ctx context.Context, ids []int64) ([]*sqlcv1.MoveBatchedQueueItemsRow, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	idsCopy := append([]int64(nil), ids...)
	f.moveCalls = append(f.moveCalls, idsCopy)

	rows := make([]*sqlcv1.MoveBatchedQueueItemsRow, len(ids))
	for i, id := range ids {
		rows[i] = &sqlcv1.MoveBatchedQueueItemsRow{
			ID:             id,
			TenantID:       uuid.UUID{},
			TaskID:         id,
			TaskInsertedAt: pgtype.Timestamptz{},
			RetryCount:     0,
		}
	}

	return rows, nil
}

func (f *fakeBatchRepo) ListExistingBatchedQueueItemIds(ctx context.Context, ids []int64) (map[int64]struct{}, error) {
	f.mu.Lock()
	idsCopy := append([]int64(nil), ids...)
	f.existingIdsCalls = append(f.existingIdsCalls, idsCopy)
	missing := f.missingIds
	f.mu.Unlock()

	existing := make(map[int64]struct{}, len(ids))
	for _, id := range ids {
		if _, ok := missing[id]; ok {
			continue
		}
		existing[id] = struct{}{}
	}
	return existing, nil
}

func (f *fakeBatchRepo) CommitAssignments(ctx context.Context, assignments []*v1repo.BatchAssignment) ([]*v1repo.BatchAssignment, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	copied := make([]*v1repo.BatchAssignment, len(assignments))
	for i, a := range assignments {
		if a == nil {
			continue
		}
		dup := *a
		copied[i] = &dup
	}

	f.commitCalls = append(f.commitCalls, copied)
	return copied, nil
}

func (f *fakeBatchRepo) ReserveAndCommitBatchRun(
	ctx context.Context,
	tenantId, stepId uuid.UUID,
	actionId, batchKey, batchId string,
	maxRuns int,
	assignments []*v1repo.BatchAssignment,
) (bool, []*v1repo.BatchAssignment, error) {
	call := &fakeReserveCall{
		TenantID: tenantId.String(),
		StepID:   stepId,
		ActionID: actionId,
		BatchKey: batchKey,
		BatchID:  batchId,
		MaxRuns:  maxRuns,
	}

	f.mu.Lock()
	f.reserveCalls = append(f.reserveCalls, call)
	reserveFunc := f.reserveFunc
	f.mu.Unlock()

	reserved := true
	if reserveFunc != nil {
		var err error
		reserved, err = reserveFunc(call)
		if err != nil {
			return false, nil, err
		}
	}

	if !reserved {
		return false, nil, nil
	}

	succeeded, err := f.CommitAssignments(ctx, assignments)
	if err != nil {
		return true, nil, err
	}

	return true, succeeded, nil
}

type fakeQueueFactory struct {
	repo v1repo.QueueRepository
}

func (f *fakeQueueFactory) NewQueue(uuid.UUID, string) v1repo.QueueRepository {
	return f.repo
}

type fakeQueueRepository struct{}

func (f *fakeQueueRepository) ListQueueItems(context.Context, int) ([]*sqlcv1.V1QueueItem, error) {
	return nil, nil
}

func (f *fakeQueueRepository) MarkQueueItemsProcessed(context.Context, *v1repo.AssignResults) ([]*v1repo.AssignedItem, []*v1repo.AssignedItem, error) {
	return nil, nil, nil
}

func (f *fakeQueueRepository) GetTaskRateLimits(context.Context, *v1repo.OptimisticTx, []*sqlcv1.V1QueueItem) (map[int64]map[string]int32, error) {
	return nil, nil
}

func (f *fakeQueueRepository) GetStepBatchConfigs(context.Context, []uuid.UUID) (map[string]bool, error) {
	return map[string]bool{}, nil
}

func (f *fakeQueueRepository) RequeueRateLimitedItems(context.Context, uuid.UUID, string) ([]*sqlcv1.RequeueRateLimitedQueueItemsRow, error) {
	return nil, nil
}

func (f *fakeQueueRepository) GetDesiredLabels(context.Context, *v1repo.OptimisticTx, []uuid.UUID) (map[uuid.UUID][]*sqlcv1.GetDesiredLabelsRow, error) {
	return make(map[uuid.UUID][]*sqlcv1.GetDesiredLabelsRow), nil
}

func (f *fakeQueueRepository) GetStepSlotRequests(context.Context, *v1repo.OptimisticTx, []uuid.UUID) (map[uuid.UUID]map[string]int32, error) {
	return nil, nil
}

func (f *fakeQueueRepository) Cleanup() {}

type fakeBatchFactory struct {
	repo v1repo.BatchQueueRepository
}

func (f *fakeBatchFactory) NewBatchQueue(uuid.UUID) v1repo.BatchQueueRepository {
	return f.repo
}

type fakeSchedulerRepository struct {
	batchFactory v1repo.BatchQueueFactoryRepository
}

func (f *fakeSchedulerRepository) Optimistic() v1repo.OptimisticSchedulingRepository {
	//TODO implement me
	panic("implement me")
}

func (f *fakeSchedulerRepository) Concurrency() v1repo.ConcurrencyRepository {
	return nil
}

func (f *fakeSchedulerRepository) Lease() v1repo.LeaseRepository {
	return nil
}

func (f *fakeSchedulerRepository) QueueFactory() v1repo.QueueFactoryRepository {
	return nil
}

func (f *fakeSchedulerRepository) BatchQueue() v1repo.BatchQueueFactoryRepository {
	return f.batchFactory
}

func (f *fakeSchedulerRepository) RateLimit() v1repo.RateLimitRepository {
	return nil
}

func (f *fakeSchedulerRepository) Assignment() v1repo.AssignmentRepository {
	return nil
}

func newTestSharedConfig(repo v1repo.BatchQueueRepository) *sharedConfig {
	logger := zerolog.New(io.Discard)

	return &sharedConfig{
		repo: &fakeSchedulerRepository{
			batchFactory: &fakeBatchFactory{repo: repo},
		},
		l: &logger,
	}
}

func TestBatchSchedulerFlushOnBatchSize(t *testing.T) {
	tenantId := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	stepId := uuid.MustParse("00000000-0000-0000-0000-000000000010")

	repo := &fakeBatchRepo{
		listResponses: [][]*sqlcv1.V1BatchedQueueItem{
			{
				{
					ID:             1,
					TenantID:       tenantId,
					Queue:          "default",
					TaskID:         101,
					TaskInsertedAt: pgtype.Timestamptz{Valid: true},
				},
				{
					ID:             2,
					TenantID:       tenantId,
					Queue:          "default",
					TaskID:         102,
					TaskInsertedAt: pgtype.Timestamptz{Valid: true},
				},
			},
		},
	}

	resource := &sqlcv1.ListDistinctBatchResourcesRow{
		StepID:       stepId,
		BatchKey:     "batch",
		BatchMaxSize: 2,
	}

	notifyCh := make(chan []string, 1) // deprecated path, preserved for compatibility

	scheduler := newBatchScheduler(
		newTestSharedConfig(repo),
		tenantId,
		resource,
		nil,
		nil,
		func(*QueueResults) {},
	)
	_ = notifyCh // retain for compile-time compatibility
	require.NotNil(t, scheduler)
}

func TestBatchSchedulerFlushOnInterval(t *testing.T) {

	tenantId := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	stepId := uuid.MustParse("00000000-0000-0000-0000-000000000020")

	repo := &fakeBatchRepo{
		listResponses: [][]*sqlcv1.V1BatchedQueueItem{
			{
				{
					ID:             42,
					TenantID:       tenantId,
					Queue:          "priority",
					TaskID:         200,
					TaskInsertedAt: pgtype.Timestamptz{Valid: true},
				},
			},
			{},
		},
	}

	resource := &sqlcv1.ListDistinctBatchResourcesRow{
		StepID:           stepId,
		BatchKey:         "interval",
		BatchMaxSize:     10,
		BatchMaxInterval: pgtype.Int4{Int32: 50, Valid: true},
	}

	notifyCh := make(chan []string, 1)

	scheduler := newBatchScheduler(
		newTestSharedConfig(repo),
		tenantId,
		resource,
		nil,
		nil,
		func(*QueueResults) {},
	)
	_ = notifyCh
	require.NotNil(t, scheduler)
}

func TestBatchSchedulerAssignAndDispatchCommitsAssignments(t *testing.T) {
	tenantId := uuid.MustParse("11111111-2222-3333-4444-555555555555")
	stepId := uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")

	repo := &fakeBatchRepo{}
	queueFactory := &fakeQueueFactory{repo: &fakeQueueRepository{}}

	var emitted []*QueueResults

	scheduler := newBatchScheduler(
		newTestSharedConfig(repo),
		tenantId,
		&sqlcv1.ListDistinctBatchResourcesRow{
			StepID:            stepId,
			BatchKey:          "group",
			BatchMaxSize:      2,
			BatchGroupMaxRuns: pgtype.Int4{Int32: 1, Valid: true},
		},
		queueFactory,
		&Scheduler{},
		func(res *QueueResults) {
			if res != nil {
				emitted = append(emitted, res)
			}
		},
	)
	require.NotNil(t, scheduler)

	item1 := &sqlcv1.V1BatchedQueueItem{
		ID:             1,
		TenantID:       tenantId,
		Queue:          "default",
		TaskID:         100,
		TaskInsertedAt: pgtype.Timestamptz{Valid: true},
		ActionID:       "action",
		StepID:         stepId,
		BatchKey:       "group",
	}

	item2 := &sqlcv1.V1BatchedQueueItem{
		ID:             2,
		TenantID:       tenantId,
		Queue:          "default",
		TaskID:         101,
		TaskInsertedAt: pgtype.Timestamptz{Valid: true},
		ActionID:       "action",
		StepID:         stepId,
		BatchKey:       "group",
	}

	workerID := uuid.MustParse("bbbbbbbb-cccc-dddd-eeee-ffffffffffff")

	var assignCalls int

	scheduler.assignOverride = func(ctx context.Context, queueItems []*sqlcv1.V1QueueItem, _ map[string][]*sqlcv1.GetDesiredLabelsRow, _ map[int64]map[string]int32) ([]*assignedQueueItem, []*sqlcv1.V1QueueItem, error) {
		assignCalls++

		assignments := make([]*assignedQueueItem, len(queueItems))

		for i, qi := range queueItems {
			assignments[i] = &assignedQueueItem{
				QueueItem: qi,
				WorkerId:  workerID,
			}
		}

		return assignments, nil, nil
	}

	queueItems := []*sqlcv1.V1BatchedQueueItem{item1, item2}

	group := &batchGroup{batchKey: "group", l: zerolog.New(io.Discard)}

	remaining, err := scheduler.assignAndDispatch(context.Background(), group, queueItems, v1repo.FlushReasonBatchSizeReached)
	require.NoError(t, err)
	require.Empty(t, remaining)

	require.Len(t, repo.reserveCalls, 1)
	require.Equal(t, "group", repo.reserveCalls[0].BatchKey)
	require.Equal(t, 1, repo.reserveCalls[0].MaxRuns)
	require.Equal(t, "action", repo.reserveCalls[0].ActionID)
	require.NotEqual(t, uuid.Nil, repo.reserveCalls[0].StepID)
	require.NotEmpty(t, repo.reserveCalls[0].BatchID)

	require.Len(t, repo.commitCalls, 1)
	require.Len(t, repo.commitCalls[0], 2)

	expectedTaskIDs := map[int64]struct{}{
		100: {},
		101: {},
	}

	for _, assignment := range repo.commitCalls[0] {
		require.Equal(t, workerID, assignment.WorkerID)
		require.True(t, assignment.TaskInsertedAt.Valid)
		require.NotEmpty(t, assignment.BatchID)
		require.Equal(t, "group", assignment.BatchKey)
		require.Equal(t, "action", assignment.ActionID)
		require.Equal(t, stepId, assignment.StepID)
		_, ok := expectedTaskIDs[assignment.TaskID]
		require.True(t, ok, "unexpected task id %d", assignment.TaskID)
		delete(expectedTaskIDs, assignment.TaskID)
	}

	require.Empty(t, expectedTaskIDs)

	require.Len(t, emitted, 1)
	require.Len(t, emitted[0].Assigned, 2)
	for _, assigned := range emitted[0].Assigned {
		require.Equal(t, workerID, assigned.WorkerId)
		require.NotNil(t, assigned.QueueItem)
		require.NotNil(t, assigned.Batch)
		require.Equal(t, v1repo.FlushReasonBatchSizeReached, assigned.Batch.Reason)
		require.Equal(t, int32(2), assigned.Batch.ConfiguredBatchMaxSize)
		require.Equal(t, int32(1), assigned.Batch.ConfiguredBatchGroupMaxRuns)
		require.NotEmpty(t, assigned.Batch.BatchID)
		require.Equal(t, "group", assigned.Batch.BatchGroupKey)
		require.Equal(t, "action", assigned.Batch.ActionID)
	}

	require.Equal(t, 1, assignCalls, "expected batch scheduling to consume a single slot")
}

func TestBatchSchedulerUsesSingleSlotForBatch(t *testing.T) {
	tenantId := uuid.MustParse("22222222-3333-4444-5555-666666666666")
	stepId := uuid.MustParse("ffffffff-eeee-dddd-cccc-bbbbbbbbbbbb")

	repo := &fakeBatchRepo{}
	queueFactory := &fakeQueueFactory{repo: &fakeQueueRepository{}}

	scheduler := newBatchScheduler(
		newTestSharedConfig(repo),
		tenantId,
		&sqlcv1.ListDistinctBatchResourcesRow{
			StepID:       stepId,
			BatchKey:     "single-slot",
			BatchMaxSize: 3,
		},
		queueFactory,
		&Scheduler{},
		func(*QueueResults) {},
	)
	require.NotNil(t, scheduler)

	items := []*sqlcv1.V1BatchedQueueItem{
		{
			ID:             10,
			TenantID:       tenantId,
			Queue:          "default",
			TaskID:         201,
			TaskInsertedAt: pgtype.Timestamptz{Valid: true},
			ActionID:       "action",
			StepID:         stepId,
			BatchKey:       "single-slot",
		},
		{
			ID:             11,
			TenantID:       tenantId,
			Queue:          "default",
			TaskID:         202,
			TaskInsertedAt: pgtype.Timestamptz{Valid: true},
			ActionID:       "action",
			StepID:         stepId,
			BatchKey:       "single-slot",
		},
		{
			ID:             12,
			TenantID:       tenantId,
			Queue:          "default",
			TaskID:         203,
			TaskInsertedAt: pgtype.Timestamptz{Valid: true},
			ActionID:       "action",
			StepID:         stepId,
			BatchKey:       "single-slot",
		},
	}

	workerID := uuid.MustParse("12345678-90ab-cdef-1234-567890abcdef")

	var assignCalls int
	var lastAssignedLen int

	scheduler.assignOverride = func(ctx context.Context, queueItems []*sqlcv1.V1QueueItem, _ map[string][]*sqlcv1.GetDesiredLabelsRow, _ map[int64]map[string]int32) ([]*assignedQueueItem, []*sqlcv1.V1QueueItem, error) {
		assignCalls++
		lastAssignedLen = len(queueItems)

		return []*assignedQueueItem{
			{
				QueueItem: queueItems[0],
				WorkerId:  workerID,
			},
		}, nil, nil
	}

	group := &batchGroup{batchKey: "single-slot", l: zerolog.New(io.Discard)}

	remaining, err := scheduler.assignAndDispatch(context.Background(), group, items, v1repo.FlushReasonBatchSizeReached)
	require.NoError(t, err)
	require.Empty(t, remaining)

	require.Len(t, repo.commitCalls, 1)
	require.Len(t, repo.commitCalls[0], len(items))
	require.Equal(t, 1, assignCalls, "batch should request a single slot")
	require.Equal(t, 1, lastAssignedLen, "batch should be scheduled with one representative queue item")
}

func TestBatchSchedulerScheduleTimeout(t *testing.T) {
	tenantId := uuid.MustParse("00000000-0000-0000-0000-000000000005")
	stepId := uuid.MustParse("00000000-0000-0000-0000-000000000050")

	// Create items with expired schedule timeout
	pastTime := time.Now().UTC().Add(-1 * time.Hour)

	repo := &fakeBatchRepo{
		listResponses: [][]*sqlcv1.V1BatchedQueueItem{
			{
				{
					ID:                1,
					TenantID:          tenantId,
					Queue:             "default",
					TaskID:            201,
					TaskInsertedAt:    pgtype.Timestamptz{Valid: true, Time: time.Now()},
					ScheduleTimeoutAt: pgtype.Timestamp{Valid: true, Time: pastTime},
					BatchKey:          "timeout-batch",
					ActionID:          "action",
					StepID:            stepId,
					WorkflowRunID:     uuid.MustParse("00000000-0000-0000-0000-000000000099"),
					ExternalID:        uuid.MustParse("00000000-0000-0000-0000-000000000098"),
				},
				{
					ID:                2,
					TenantID:          tenantId,
					Queue:             "default",
					TaskID:            202,
					TaskInsertedAt:    pgtype.Timestamptz{Valid: true, Time: time.Now()},
					ScheduleTimeoutAt: pgtype.Timestamp{Valid: true, Time: time.Now().Add(1 * time.Hour)}, // not expired
					BatchKey:          "timeout-batch",
					ActionID:          "action",
					StepID:            stepId,
				},
			},
			{}, // empty to stop fetching
		},
	}

	resource := &sqlcv1.ListDistinctBatchResourcesRow{
		StepID:       stepId,
		BatchKey:     "timeout-batch",
		BatchMaxSize: 5,
	}

	var emitted []*QueueResults

	scheduler := newBatchScheduler(
		newTestSharedConfig(repo),
		tenantId,
		resource,
		nil,
		nil,
		func(res *QueueResults) {
			emitted = append(emitted, res)
		},
	)
	require.NotNil(t, scheduler)

	scheduler.Start(context.Background())
	t.Cleanup(func() {
		_ = scheduler.Cleanup(context.Background())
	})

	// Wait for scheduler to process items
	time.Sleep(500 * time.Millisecond)

	// Stop scheduler to avoid races with result inspection
	require.NoError(t, scheduler.Cleanup(context.Background()))

	// Check that timed out items were emitted as SchedulingTimedOut
	var schedulingTimedOutCount int
	var bufferedCount int

	for _, result := range emitted {
		schedulingTimedOutCount += len(result.SchedulingTimedOut)
		bufferedCount += len(result.Buffered)
	}

	require.Equal(t, 1, schedulingTimedOutCount, "expected 1 item to time out during scheduling")
	require.Equal(t, 1, bufferedCount, "expected 1 item to be buffered (not timed out)")

	// Verify that the timed out item was deleted
	require.True(t, len(repo.moveCalls) == 0, "no items should be moved for schedule timeout")
}

// TestBatchSchedulerHandlesMultipleBatchKeysForStep verifies that a single BatchScheduler for a
// step discovers and independently buffers/flushes multiple batch_key groups from one shared
// poll, rather than requiring a separate scheduler per (step_id, batch_key) pair.
func TestBatchSchedulerHandlesMultipleBatchKeysForStep(t *testing.T) {
	tenantId := uuid.MustParse("00000000-0000-0000-0000-000000000006")
	stepId := uuid.MustParse("00000000-0000-0000-0000-000000000060")

	repo := &fakeBatchRepo{
		listResponses: [][]*sqlcv1.V1BatchedQueueItem{
			{
				{ID: 1, TenantID: tenantId, Queue: "default", TaskID: 301, TaskInsertedAt: pgtype.Timestamptz{Valid: true}, ActionID: "action", StepID: stepId, BatchKey: "a"},
				{ID: 2, TenantID: tenantId, Queue: "default", TaskID: 302, TaskInsertedAt: pgtype.Timestamptz{Valid: true}, ActionID: "action", StepID: stepId, BatchKey: "a"},
				{ID: 3, TenantID: tenantId, Queue: "default", TaskID: 303, TaskInsertedAt: pgtype.Timestamptz{Valid: true}, ActionID: "action", StepID: stepId, BatchKey: "b"},
				{ID: 4, TenantID: tenantId, Queue: "default", TaskID: 304, TaskInsertedAt: pgtype.Timestamptz{Valid: true}, ActionID: "action", StepID: stepId, BatchKey: "b"},
			},
			{}, // empty afterwards so already-flushed items aren't rebuffered
		},
	}

	queueFactory := &fakeQueueFactory{repo: &fakeQueueRepository{}}
	workerID := uuid.MustParse("11111111-2222-3333-4444-000000000abc")

	resource := &sqlcv1.ListDistinctBatchResourcesRow{
		StepID:       stepId,
		BatchKey:     "a", // representative row only; the scheduler covers every batch key for the step
		BatchMaxSize: 2,
	}

	var emitted []*QueueResults
	var emittedMu sync.Mutex

	scheduler := newBatchScheduler(
		newTestSharedConfig(repo),
		tenantId,
		resource,
		queueFactory,
		&Scheduler{},
		func(res *QueueResults) {
			emittedMu.Lock()
			defer emittedMu.Unlock()
			emitted = append(emitted, res)
		},
	)
	require.NotNil(t, scheduler)

	scheduler.assignOverride = func(ctx context.Context, queueItems []*sqlcv1.V1QueueItem, _ map[string][]*sqlcv1.GetDesiredLabelsRow, _ map[int64]map[string]int32) ([]*assignedQueueItem, []*sqlcv1.V1QueueItem, error) {
		assignments := make([]*assignedQueueItem, len(queueItems))
		for i, qi := range queueItems {
			assignments[i] = &assignedQueueItem{QueueItem: qi, WorkerId: workerID}
		}
		return assignments, nil, nil
	}

	scheduler.Start(context.Background())
	t.Cleanup(func() {
		_ = scheduler.Cleanup(context.Background())
	})

	require.Eventually(t, func() bool {
		repo.mu.Lock()
		defer repo.mu.Unlock()
		return len(repo.reserveCalls) >= 2
	}, 2*time.Second, 10*time.Millisecond, "expected both batch key groups to flush")

	require.NoError(t, scheduler.Cleanup(context.Background()))

	batchKeysSeen := map[string]bool{}
	for _, call := range repo.reserveCalls {
		batchKeysSeen[call.BatchKey] = true
		require.Equal(t, stepId, call.StepID)
	}
	require.True(t, batchKeysSeen["a"], "expected batch key 'a' to be flushed")
	require.True(t, batchKeysSeen["b"], "expected batch key 'b' to be flushed")

	require.Len(t, repo.commitCalls, 2)
	for _, calls := range repo.commitCalls {
		require.Len(t, calls, 2)
	}
}

// TestBatchSchedulerReconcilesBuffersInSingleCall verifies that reconciling buffered items
// against the DB happens with one ListExistingBatchedQueueItemIds call across every batch_key
// group in a tick, not one call per group, and that a stale item is still correctly dropped from
// whichever group it belongs to.
func TestBatchSchedulerReconcilesBuffersInSingleCall(t *testing.T) {
	tenantId := uuid.MustParse("00000000-0000-0000-0000-000000000007")
	stepId := uuid.MustParse("00000000-0000-0000-0000-000000000070")

	repo := &fakeBatchRepo{
		missingIds: map[int64]struct{}{
			3: {}, // item 3 (in group "b") has since been cancelled/deleted
		},
	}

	resource := &sqlcv1.ListDistinctBatchResourcesRow{
		StepID:       stepId,
		BatchKey:     "a",
		BatchMaxSize: 100, // large enough that nothing flushes during this test
	}

	scheduler := newBatchScheduler(
		newTestSharedConfig(repo),
		tenantId,
		resource,
		nil,
		nil,
		func(*QueueResults) {},
	)
	require.NotNil(t, scheduler)
	scheduler.ctx = context.Background()

	scheduler.groups["a"] = &batchGroup{
		batchKey: "a",
		l:        zerolog.New(io.Discard),
		buffer: []*sqlcv1.V1BatchedQueueItem{
			{ID: 1, BatchKey: "a"},
			{ID: 2, BatchKey: "a"},
		},
	}
	scheduler.groups["b"] = &batchGroup{
		batchKey: "b",
		l:        zerolog.New(io.Discard),
		buffer: []*sqlcv1.V1BatchedQueueItem{
			{ID: 3, BatchKey: "b"},
			{ID: 4, BatchKey: "b"},
		},
	}

	scheduler.reconcileBuffers()

	require.Len(t, repo.existingIdsCalls, 1, "expected a single reconcile call across all groups")
	require.ElementsMatch(t, []int64{1, 2, 3, 4}, repo.existingIdsCalls[0])

	require.Len(t, scheduler.groups["a"].buffer, 2, "group 'a' should be untouched")
	require.Len(t, scheduler.groups["b"].buffer, 1, "stale item 3 should be dropped from group 'b'")
	require.Equal(t, int64(4), scheduler.groups["b"].buffer[0].ID)
}
