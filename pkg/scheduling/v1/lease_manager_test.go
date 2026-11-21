//go:build !e2e && !load && !rampup && !integration

package v1

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/google/uuid"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	v1repo "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

type mockLeaseRepo struct {
	mock.Mock
}

func (m *mockLeaseRepo) ListQueues(ctx context.Context, tenantId uuid.UUID) ([]*sqlcv1.V1Queue, error) {
	args := m.Called(ctx, tenantId)
	return args.Get(0).([]*sqlcv1.V1Queue), args.Error(1)
}

func (m *mockLeaseRepo) ListActiveWorkers(ctx context.Context, tenantId uuid.UUID) ([]*v1.ListActiveWorkersResult, error) {
	args := m.Called(ctx, tenantId)
	return args.Get(0).([]*v1.ListActiveWorkersResult), args.Error(1)
}

func (m *mockLeaseRepo) ListConcurrencyStrategies(ctx context.Context, tenantId uuid.UUID) ([]*sqlcv1.V1StepConcurrency, error) {
	args := m.Called(ctx, tenantId)
	return args.Get(0).([]*sqlcv1.V1StepConcurrency), args.Error(1)
}

func (m *mockLeaseRepo) AcquireOrExtendLeases(ctx context.Context, tenantId uuid.UUID, kind sqlcv1.LeaseKind, resourceIds []string, existingLeases []*sqlcv1.Lease) ([]*sqlcv1.Lease, error) {
	args := m.Called(ctx, kind, resourceIds, existingLeases)
	return args.Get(0).([]*sqlcv1.Lease), args.Error(1)
}

func (m *mockLeaseRepo) RenewLeases(ctx context.Context, tenantId uuid.UUID, leases []*sqlcv1.Lease) ([]*sqlcv1.Lease, error) {
	args := m.Called(ctx, leases)
	return args.Get(0).([]*sqlcv1.Lease), args.Error(1)
}

func (m *mockLeaseRepo) ReleaseLeases(ctx context.Context, tenantId uuid.UUID, leases []*sqlcv1.Lease) error {
	args := m.Called(ctx, leases)
	return args.Error(0)
}

type fakeBatchQueueFactory struct {
	resources []*sqlcv1.ListDistinctBatchResourcesRow
}

func (f *fakeBatchQueueFactory) NewBatchQueue(uuid.UUID) v1repo.BatchQueueRepository {
	return &fakeBatchQueueRepo{resources: f.resources}
}

type fakeBatchQueueRepo struct {
	resources []*sqlcv1.ListDistinctBatchResourcesRow
}

func (f *fakeBatchQueueRepo) ListBatchResources(_ context.Context, stepIds []uuid.UUID) ([]*sqlcv1.ListDistinctBatchResourcesRow, error) {
	if stepIds == nil {
		return f.resources, nil
	}

	wanted := make(map[uuid.UUID]struct{}, len(stepIds))
	for _, id := range stepIds {
		wanted[id] = struct{}{}
	}

	rows := make([]*sqlcv1.ListDistinctBatchResourcesRow, 0, len(f.resources))
	for _, row := range f.resources {
		if _, ok := wanted[row.StepID]; ok {
			rows = append(rows, row)
		}
	}
	return rows, nil
}

func (f *fakeBatchQueueRepo) ListBatchedQueueItems(context.Context, uuid.UUID, []int64, int32) ([]*sqlcv1.V1BatchedQueueItem, error) {
	return nil, nil
}

func (f *fakeBatchQueueRepo) GetBatchedQueueItemsByIds(context.Context, []int64) ([]*sqlcv1.V1BatchedQueueItem, error) {
	return nil, nil
}

func (f *fakeBatchQueueRepo) DeleteBatchedQueueItems(context.Context, []int64) error {
	return nil
}

func (f *fakeBatchQueueRepo) MoveBatchedQueueItems(context.Context, []int64) ([]*sqlcv1.MoveBatchedQueueItemsRow, error) {
	return nil, nil
}

func (f *fakeBatchQueueRepo) ListExistingBatchedQueueItemIds(context.Context, []int64) (map[int64]struct{}, error) {
	return map[int64]struct{}{}, nil
}

func (f *fakeBatchQueueRepo) CommitAssignments(context.Context, []*v1repo.BatchAssignment) ([]*v1repo.BatchAssignment, error) {
	return nil, nil
}

func (f *fakeBatchQueueRepo) ReserveAndCommitBatchRun(
	context.Context,
	uuid.UUID, uuid.UUID,
	string, string, string,
	int,
	[]*v1repo.BatchAssignment,
) (bool, []*v1repo.BatchAssignment, error) {
	return true, nil, nil
}

type fakeLeaseRepo struct {
	resourceSets [][]string

	// ownedElsewhere lists resource ids another scheduler holds; they come back unacquired
	ownedElsewhere map[string]struct{}
}

func (f *fakeLeaseRepo) ListQueues(context.Context, uuid.UUID) ([]*sqlcv1.V1Queue, error) {
	return nil, nil
}

func (f *fakeLeaseRepo) ListActiveWorkers(context.Context, uuid.UUID) ([]*v1.ListActiveWorkersResult, error) {
	return nil, nil
}

func (f *fakeLeaseRepo) GetActiveWorker(context.Context, uuid.UUID, uuid.UUID) (*v1.ListActiveWorkersResult, error) {
	return nil, nil
}

func (f *fakeLeaseRepo) ListConcurrencyStrategies(context.Context, uuid.UUID) ([]*sqlcv1.V1StepConcurrency, error) {
	return nil, nil
}

func (f *fakeLeaseRepo) GetConcurrencyStrategy(context.Context, uuid.UUID, int64) (*sqlcv1.V1StepConcurrency, error) {
	return nil, nil
}

func (f *fakeLeaseRepo) AcquireOrExtendLeases(_ context.Context, _ uuid.UUID, _ sqlcv1.LeaseKind, resourceIds []string, _ []*sqlcv1.Lease) ([]*sqlcv1.Lease, error) {
	f.resourceSets = append(f.resourceSets, resourceIds)
	leases := make([]*sqlcv1.Lease, len(resourceIds))
	for i, id := range resourceIds {
		if _, ok := f.ownedElsewhere[id]; ok {
			leases[i] = &sqlcv1.Lease{}
			continue
		}
		leases[i] = &sqlcv1.Lease{ResourceId: id}
	}
	return leases, nil
}

func (f *fakeLeaseRepo) ReleaseLeases(context.Context, uuid.UUID, []*sqlcv1.Lease) error {
	return nil
}

type fakeSchedulerRepo struct {
	leaseRepo *fakeLeaseRepo
	batchRepo *fakeBatchQueueFactory
}

func (f *fakeSchedulerRepo) Optimistic() v1repo.OptimisticSchedulingRepository {
	return nil
}

func (f *fakeSchedulerRepo) Concurrency() v1repo.ConcurrencyRepository {
	return nil
}

func (f *fakeSchedulerRepo) Lease() v1repo.LeaseRepository {
	return f.leaseRepo
}

func (f *fakeSchedulerRepo) QueueFactory() v1repo.QueueFactoryRepository {
	return nil
}

func (f *fakeSchedulerRepo) BatchQueue() v1repo.BatchQueueFactoryRepository {
	return f.batchRepo
}

func (f *fakeSchedulerRepo) RateLimit() v1repo.RateLimitRepository {
	return nil
}

func (f *fakeSchedulerRepo) Assignment() v1repo.AssignmentRepository {
	return nil
}

func (m *mockLeaseRepo) GetActiveWorker(ctx context.Context, tenantId, workerId uuid.UUID) (*v1.ListActiveWorkersResult, error) {
	args := m.Called(ctx, tenantId, workerId)
	return args.Get(0).(*v1.ListActiveWorkersResult), args.Error(1)
}

func (m *mockLeaseRepo) GetConcurrencyStrategy(ctx context.Context, tenantId uuid.UUID, id int64) (*sqlcv1.V1StepConcurrency, error) {
	args := m.Called(ctx, tenantId, id)
	return args.Get(0).(*sqlcv1.V1StepConcurrency), args.Error(1)
}

func TestLeaseManager_AcquireWorkerLeases(t *testing.T) {
	l := zerolog.Nop()
	tenantId := uuid.UUID{}
	mockLeaseRepo := &mockLeaseRepo{}
	leaseManager := &LeaseManager{
		lr:       mockLeaseRepo,
		conf:     &sharedConfig{l: &l},
		tenantId: tenantId,
	}

	mockWorkers := []*v1.ListActiveWorkersResult{
		{ID: uuid.New(), Labels: nil},
		{ID: uuid.New(), Labels: nil},
	}
	mockLeases := []*sqlcv1.Lease{
		{ID: 1, ResourceId: "worker-1"},
		{ID: 2, ResourceId: "worker-2"},
	}

	mockLeaseRepo.On("ListActiveWorkers", mock.Anything, tenantId).Return(mockWorkers, nil)
	mockLeaseRepo.On("AcquireOrExtendLeases", mock.Anything, sqlcv1.LeaseKindWORKER, mock.Anything, mock.Anything).Return(mockLeases, nil)

	err := leaseManager.acquireWorkerLeases(context.Background())
	assert.NoError(t, err)
	assert.Len(t, leaseManager.workerLeases, 2)
}

func TestLeaseManager_AcquireQueueLeases(t *testing.T) {
	l := zerolog.Nop()
	tenantId := uuid.UUID{}
	mockLeaseRepo := &mockLeaseRepo{}
	leaseManager := &LeaseManager{
		lr:       mockLeaseRepo,
		conf:     &sharedConfig{l: &l},
		tenantId: tenantId,
	}

	mockQueues := []*sqlcv1.V1Queue{
		{Name: "queue-1"},
		{Name: "queue-2"},
	}
	mockLeases := []*sqlcv1.Lease{
		{ID: 1, ResourceId: "queue-1"},
		{ID: 2, ResourceId: "queue-2"},
	}

	mockLeaseRepo.On("ListQueues", mock.Anything, tenantId).Return(mockQueues, nil)
	mockLeaseRepo.On("AcquireOrExtendLeases", mock.Anything, sqlcv1.LeaseKindQUEUE, mock.Anything, mock.Anything).Return(mockLeases, nil)

	err := leaseManager.acquireQueueLeases(context.Background())
	assert.NoError(t, err)
	assert.Len(t, leaseManager.queueLeases, 2)
}

func TestLeaseManager_SendWorkerIds(t *testing.T) {
	tenantId := uuid.UUID{}
	// buffered so the non-blocking send in sendWorkerIds cannot race the
	// receiver and drop the message
	workersCh := make(notifierCh[*v1.ListActiveWorkersResult], 1)
	leaseManager := &LeaseManager{
		tenantId:  tenantId,
		workersCh: workersCh,
	}

	mockWorkers := []*v1.ListActiveWorkersResult{
		{ID: uuid.New(), Labels: nil},
	}

	leaseManager.sendWorkerIds(mockWorkers, false)

	result := <-workersCh
	assert.Equal(t, mockWorkers, result.items)
}

func TestLeaseManager_SendQueues(t *testing.T) {
	tenantId := uuid.UUID{}
	// buffered so the non-blocking send in sendQueues cannot race the
	// receiver and drop the message
	queuesCh := make(notifierCh[string], 1)
	leaseManager := &LeaseManager{
		tenantId: tenantId,
		queuesCh: queuesCh,
	}

	mockQueues := []string{"queue-1", "queue-2"}

	leaseManager.sendQueues(mockQueues, false)

	result := <-queuesCh
	assert.Equal(t, mockQueues, result.items)
}

func TestLeaseManager_AcquireWorkersBeforeListenerReady(t *testing.T) {
	tenantId := uuid.UUID{}
	workersCh := make(notifierCh[*v1.ListActiveWorkersResult])
	leaseManager := &LeaseManager{
		tenantId:  tenantId,
		workersCh: workersCh,
	}

	mockWorkers1 := []*v1.ListActiveWorkersResult{
		{ID: uuid.New(), Labels: nil},
	}
	mockWorkers2 := []*v1.ListActiveWorkersResult{
		{ID: uuid.New(), Labels: nil},
		{ID: uuid.New(), Labels: nil},
	}

	// Send workers before listener is ready
	go leaseManager.sendWorkerIds(mockWorkers1, false)
	time.Sleep(100 * time.Millisecond)
	resultCh := make(chan []*v1.ListActiveWorkersResult)
	go func() {
		msg := <-workersCh
		resultCh <- msg.items
	}()
	time.Sleep(100 * time.Millisecond)
	go leaseManager.sendWorkerIds(mockWorkers2, false)
	time.Sleep(100 * time.Millisecond)

	// Ensure only the latest workers are sent over the channel
	result := <-resultCh
	assert.Equal(t, mockWorkers2, result)
	assert.Len(t, workersCh, 0) // Ensure no additional workers are left in the channel
}

func TestAcquireBatchLeasesUsesStepLevelResourceIds(t *testing.T) {
	tenantID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	stepID := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	otherStepID := uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc")

	resources := []*sqlcv1.ListDistinctBatchResourcesRow{
		{StepID: stepID},
		{StepID: otherStepID},
	}

	leaseRepo := &fakeLeaseRepo{}
	repo := &fakeSchedulerRepo{
		leaseRepo: leaseRepo,
		batchRepo: &fakeBatchQueueFactory{resources: resources},
	}

	logger := zerolog.New(io.Discard)
	lm, _, _, _, batchesCh := newLeaseManager(&sharedConfig{repo: repo, l: &logger}, tenantID)

	require.NoError(t, lm.acquireBatchLeases(context.Background()))

	select {
	case msg := <-batchesCh:
		require.False(t, msg.isIncremental)
		require.Len(t, msg.items, 2, "every leased step should propagate")
	case <-time.After(1 * time.Second):
		t.Fatal("expected batch resources to be sent")
	}

	require.Len(t, leaseRepo.resourceSets, 1)
	acquired := leaseRepo.resourceSets[0]
	require.ElementsMatch(t, []string{
		stepID.String(),
		otherStepID.String(),
	}, acquired, "leases should be requested per step_id only")
}

func TestNotifyNewBatchStepsSendsIncrementalRowsForLeasedSteps(t *testing.T) {
	tenantID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	stepID := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	ownedStepID := uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc")
	unpendingStepID := uuid.MustParse("dddddddd-dddd-dddd-dddd-dddddddddddd")

	leaseRepo := &fakeLeaseRepo{ownedElsewhere: map[string]struct{}{ownedStepID.String(): {}}}
	repo := &fakeSchedulerRepo{
		leaseRepo: leaseRepo,
		batchRepo: &fakeBatchQueueFactory{resources: []*sqlcv1.ListDistinctBatchResourcesRow{
			{StepID: stepID, BatchMaxSize: 5, BatchMaxInterval: pgtype.Int4{Int32: 1000, Valid: true}},
			{StepID: ownedStepID, BatchMaxSize: 5},
		}},
	}

	logger := zerolog.New(io.Discard)
	lm, _, _, _, batchesCh := newLeaseManager(&sharedConfig{repo: repo, l: &logger}, tenantID)

	require.NoError(t, lm.notifyNewBatchSteps(context.Background(), []uuid.UUID{stepID, ownedStepID, unpendingStepID}))

	select {
	case msg := <-batchesCh:
		require.True(t, msg.isIncremental)
		require.Len(t, msg.items, 1, "only the step this scheduler leased should start")
		require.Equal(t, stepID, msg.items[0].StepID)
		require.Equal(t, int32(5), msg.items[0].BatchMaxSize)
		require.Equal(t, int32(1000), msg.items[0].BatchMaxInterval.Int32)
	case <-time.After(1 * time.Second):
		t.Fatal("expected an incremental batch message")
	}

	require.Len(t, lm.batchLeases, 1)
	require.Equal(t, stepID.String(), lm.batchLeases[0].ResourceId)

	// a step whose lease is already held is sent again without another acquire
	require.NoError(t, lm.notifyNewBatchSteps(context.Background(), []uuid.UUID{stepID}))
	require.Len(t, leaseRepo.resourceSets, 1)

	select {
	case msg := <-batchesCh:
		require.True(t, msg.isIncremental)
		require.Len(t, msg.items, 1)
	case <-time.After(1 * time.Second):
		t.Fatal("expected an incremental batch message for the held lease")
	}
}

func TestAddBatchSchedulersKeepsExistingSchedulers(t *testing.T) {
	tenantID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	runningStepID := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	newStepID := uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc")

	logger := zerolog.New(io.Discard)
	cf := &sharedConfig{
		repo: &fakeSchedulerRepo{leaseRepo: &fakeLeaseRepo{}, batchRepo: &fakeBatchQueueFactory{}},
		l:    &logger,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tm := &tenantManager{
		cf:              cf,
		l:               logger,
		tenantId:        tenantID,
		batchSchedulers: make(map[uuid.UUID]*BatchScheduler),
		resultsCh:       make(chan *QueueResults, 1),
	}

	tm.addBatchSchedulers(ctx, []*sqlcv1.ListDistinctBatchResourcesRow{{StepID: runningStepID, BatchMaxSize: 5}})
	running := tm.batchSchedulers[runningStepID]
	require.NotNil(t, running)

	tm.addBatchSchedulers(ctx, []*sqlcv1.ListDistinctBatchResourcesRow{
		{StepID: runningStepID, BatchMaxSize: 5},
		{StepID: newStepID, BatchMaxSize: 5},
	})

	require.Len(t, tm.batchSchedulers, 2)
	require.Same(t, running, tm.batchSchedulers[runningStepID], "a running scheduler is not replaced")

	// a scheduler that stopped itself for idleness is replaced
	running.cancel()
	tm.addBatchSchedulers(ctx, []*sqlcv1.ListDistinctBatchResourcesRow{{StepID: runningStepID, BatchMaxSize: 5}})
	require.NotSame(t, running, tm.batchSchedulers[runningStepID])
}

func TestNotifyBatchedQueueItemsSkipsStepsWithRunningSchedulers(t *testing.T) {
	tenantID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	runningStepID := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	idleStepID := uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc")

	leaseRepo := &fakeLeaseRepo{}
	repo := &fakeSchedulerRepo{
		leaseRepo: leaseRepo,
		batchRepo: &fakeBatchQueueFactory{resources: []*sqlcv1.ListDistinctBatchResourcesRow{
			{StepID: runningStepID, BatchMaxSize: 5},
			{StepID: idleStepID, BatchMaxSize: 5},
		}},
	}

	logger := zerolog.New(io.Discard)
	cf := &sharedConfig{repo: repo, l: &logger}
	lm, _, _, _, batchesCh := newLeaseManager(cf, tenantID)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tm := &tenantManager{
		cf:                   cf,
		l:                    logger,
		tenantId:             tenantID,
		leaseManager:         lm,
		batchSchedulers:      make(map[uuid.UUID]*BatchScheduler),
		pendingBatchNotifies: make(map[uuid.UUID]struct{}),
		resultsCh:            make(chan *QueueResults, 1),
		ctx:                  ctx,
	}

	tm.addBatchSchedulers(ctx, []*sqlcv1.ListDistinctBatchResourcesRow{{StepID: runningStepID, BatchMaxSize: 5}})

	tm.notifyBatchedQueueItems([]*sqlcv1.V1QueueItem{
		{StepID: runningStepID},
		{StepID: idleStepID},
		{StepID: idleStepID},
	})

	select {
	case msg := <-batchesCh:
		require.True(t, msg.isIncremental)
		require.Len(t, msg.items, 1)
		require.Equal(t, idleStepID, msg.items[0].StepID)
	case <-time.After(1 * time.Second):
		t.Fatal("expected an incremental batch message for the idle step")
	}

	require.Eventually(t, func() bool {
		tm.batchSchedulersMu.Lock()
		defer tm.batchSchedulersMu.Unlock()
		return len(tm.pendingBatchNotifies) == 0
	}, time.Second, 10*time.Millisecond)

	require.Len(t, leaseRepo.resourceSets, 1)
	require.Equal(t, []string{idleStepID.String()}, leaseRepo.resourceSets[0])
}
