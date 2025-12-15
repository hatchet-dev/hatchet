package v1

import (
	"context"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	v1repo "github.com/hatchet-dev/hatchet/pkg/repository/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"
)

type fakeBatchRepo struct {
	mu            sync.Mutex
	listResponses [][]*sqlcv1.V1BatchedQueueItem
	moveCalls     [][]int64
	commitCalls   [][]*v1repo.BatchAssignment
}

func (f *fakeBatchRepo) ListBatchResources(ctx context.Context) ([]*sqlcv1.ListDistinctBatchResourcesRow, error) {
	return nil, nil
}

func (f *fakeBatchRepo) ListBatchedQueueItems(ctx context.Context, stepId pgtype.UUID, batchKey string, afterId pgtype.Int8, limit int32) ([]*sqlcv1.V1BatchedQueueItem, error) {
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
			TenantID:       pgtype.UUID{Valid: true},
			TaskID:         id,
			TaskInsertedAt: pgtype.Timestamptz{},
			RetryCount:     0,
		}
	}

	return rows, nil
}

func (f *fakeBatchRepo) ListExistingBatchedQueueItemIds(ctx context.Context, ids []int64) (map[int64]struct{}, error) {
	existing := make(map[int64]struct{}, len(ids))
	for _, id := range ids {
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

type fakeQueueFactory struct {
	repo v1repo.QueueRepository
}

func (f *fakeQueueFactory) NewQueue(pgtype.UUID, string) v1repo.QueueRepository {
	return f.repo
}

type fakeQueueRepository struct{}

func (f *fakeQueueRepository) ListQueueItems(context.Context, int) ([]*sqlcv1.V1QueueItem, error) {
	return nil, nil
}

func (f *fakeQueueRepository) MarkQueueItemsProcessed(context.Context, *v1repo.AssignResults) ([]*v1repo.AssignedItem, []*v1repo.AssignedItem, error) {
	return nil, nil, nil
}

func (f *fakeQueueRepository) GetTaskRateLimits(context.Context, []*sqlcv1.V1QueueItem) (map[int64]map[string]int32, error) {
	return nil, nil
}

func (f *fakeQueueRepository) GetStepBatchConfigs(context.Context, []pgtype.UUID) (map[string]bool, error) {
	return map[string]bool{}, nil
}

func (f *fakeQueueRepository) RequeueRateLimitedItems(context.Context, pgtype.UUID, string) ([]*sqlcv1.RequeueRateLimitedQueueItemsRow, error) {
	return nil, nil
}

func (f *fakeQueueRepository) GetDesiredLabels(context.Context, []pgtype.UUID) (map[string][]*sqlcv1.GetDesiredLabelsRow, error) {
	return make(map[string][]*sqlcv1.GetDesiredLabelsRow), nil
}

func (f *fakeQueueRepository) Cleanup() {}

type fakeBatchFactory struct {
	repo v1repo.BatchQueueRepository
}

func (f *fakeBatchFactory) NewBatchQueue(pgtype.UUID) v1repo.BatchQueueRepository {
	return f.repo
}

type fakeSchedulerRepository struct {
	batchFactory v1repo.BatchQueueFactoryRepository
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
	t.Skip("TODO: update for new batch scheduler flush semantics")

	tenantId := sqlchelpers.UUIDFromStr("00000000-0000-0000-0000-000000000001")
	stepId := sqlchelpers.UUIDFromStr("00000000-0000-0000-0000-000000000010")

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
		StepID:               stepId,
		BatchKey:             "batch",
		BatchSize:            2,
		BatchFlushIntervalMs: 0,
	}

	notifyCh := make(chan []string, 1) // deprecated path, preserved for compatibility

	scheduler := newBatchScheduler(
		newTestSharedConfig(repo),
		tenantId,
		resource,
		nil,
		nil,
		func(*QueueResults) {},
		nil,
	)
	_ = notifyCh // retain for compile-time compatibility
	require.NotNil(t, scheduler)
}

func TestBatchSchedulerFlushOnInterval(t *testing.T) {
	t.Skip("TODO: update for new batch scheduler flush semantics")

	tenantId := sqlchelpers.UUIDFromStr("00000000-0000-0000-0000-000000000002")
	stepId := sqlchelpers.UUIDFromStr("00000000-0000-0000-0000-000000000020")

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
		StepID:               stepId,
		BatchKey:             "interval",
		BatchSize:            10,
		BatchFlushIntervalMs: 50,
	}

	notifyCh := make(chan []string, 1)

	scheduler := newBatchScheduler(
		newTestSharedConfig(repo),
		tenantId,
		resource,
		nil,
		nil,
		func(*QueueResults) {},
		nil,
	)
	_ = notifyCh
	require.NotNil(t, scheduler)
}

func TestBatchSchedulerAssignAndDispatchCommitsAssignments(t *testing.T) {
	tenantId := sqlchelpers.UUIDFromStr("11111111-2222-3333-4444-555555555555")
	stepId := sqlchelpers.UUIDFromStr("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")

	repo := &fakeBatchRepo{}
	queueFactory := &fakeQueueFactory{repo: &fakeQueueRepository{}}

	var emitted []*QueueResults
	var reserveRequests []*BatchReservationRequest

	scheduler := newBatchScheduler(
		newTestSharedConfig(repo),
		tenantId,
		&sqlcv1.ListDistinctBatchResourcesRow{
			StepID:               stepId,
			BatchKey:             "group",
			BatchSize:            2,
			BatchFlushIntervalMs: 0,
			BatchMaxRuns:         1,
		},
		queueFactory,
		&Scheduler{},
		func(res *QueueResults) {
			if res != nil {
				emitted = append(emitted, res)
			}
		},
		func(ctx context.Context, req *BatchReservationRequest) (bool, error) {
			if req != nil {
				c := *req
				reserveRequests = append(reserveRequests, &c)
			}
			return true, nil
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

	workerID := sqlchelpers.UUIDFromStr("bbbbbbbb-cccc-dddd-eeee-ffffffffffff")

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

	remaining, err := scheduler.assignAndDispatch(context.Background(), queueItems, flushReasonBatchSizeReached)
	require.NoError(t, err)
	require.Empty(t, remaining)

	require.Len(t, reserveRequests, 1)
	require.Equal(t, "group", reserveRequests[0].BatchKey)
	require.Equal(t, 1, reserveRequests[0].MaxRuns)
	require.Equal(t, "action", reserveRequests[0].ActionID)
	require.True(t, reserveRequests[0].StepID.Valid)
	require.NotEmpty(t, reserveRequests[0].BatchID)

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
		require.Equal(t, string(flushReasonBatchSizeReached), assigned.Batch.Reason)
		require.Equal(t, int32(2), assigned.Batch.ConfiguredBatchSize)
		require.Equal(t, int32(1), assigned.Batch.MaxRuns)
		require.NotEmpty(t, assigned.Batch.BatchID)
		require.Equal(t, "group", assigned.Batch.BatchKey)
		require.Equal(t, "action", assigned.Batch.ActionID)
	}

	require.Equal(t, 1, assignCalls, "expected batch scheduling to consume a single slot")
}

func TestBatchSchedulerUsesSingleSlotForBatch(t *testing.T) {
	tenantId := sqlchelpers.UUIDFromStr("22222222-3333-4444-5555-666666666666")
	stepId := sqlchelpers.UUIDFromStr("ffffffff-eeee-dddd-cccc-bbbbbbbbbbbb")

	repo := &fakeBatchRepo{}
	queueFactory := &fakeQueueFactory{repo: &fakeQueueRepository{}}

	scheduler := newBatchScheduler(
		newTestSharedConfig(repo),
		tenantId,
		&sqlcv1.ListDistinctBatchResourcesRow{
			StepID:               stepId,
			BatchKey:             "single-slot",
			BatchSize:            3,
			BatchFlushIntervalMs: 0,
		},
		queueFactory,
		&Scheduler{},
		func(*QueueResults) {},
		nil,
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

	workerID := sqlchelpers.UUIDFromStr("12345678-90ab-cdef-1234-567890abcdef")

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

	remaining, err := scheduler.assignAndDispatch(context.Background(), items, flushReasonBatchSizeReached)
	require.NoError(t, err)
	require.Empty(t, remaining)

	require.Len(t, repo.commitCalls, 1)
	require.Len(t, repo.commitCalls[0], len(items))
	require.Equal(t, 1, assignCalls, "batch should request a single slot")
	require.Equal(t, 1, lastAssignedLen, "batch should be scheduled with one representative queue item")
}

func TestBatchSchedulerScheduleTimeout(t *testing.T) {
	tenantId := sqlchelpers.UUIDFromStr("00000000-0000-0000-0000-000000000005")
	stepId := sqlchelpers.UUIDFromStr("00000000-0000-0000-0000-000000000050")

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
					WorkflowRunID:     sqlchelpers.UUIDFromStr("00000000-0000-0000-0000-000000000099"),
					ExternalID:        sqlchelpers.UUIDFromStr("00000000-0000-0000-0000-000000000098"),
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
		StepID:               stepId,
		BatchKey:             "timeout-batch",
		BatchSize:            5,
		BatchFlushIntervalMs: 0,
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
		nil,
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
