//go:build !e2e && !load && !rampup && !integration

package v1

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/google/uuid"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
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

func (f *fakeBatchQueueRepo) ListBatchResources(context.Context) ([]*sqlcv1.ListDistinctBatchResourcesRow, error) {
	return f.resources, nil
}

func (f *fakeBatchQueueRepo) ListBatchedQueueItems(context.Context, uuid.UUID, string, int32) ([]*sqlcv1.V1BatchedQueueItem, error) {
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
	workersCh := make(notifierCh[*v1.ListActiveWorkersResult])
	leaseManager := &LeaseManager{
		tenantId:  tenantId,
		workersCh: workersCh,
	}

	mockWorkers := []*v1.ListActiveWorkersResult{
		{ID: uuid.New(), Labels: nil},
	}

	go leaseManager.sendWorkerIds(mockWorkers, false)

	result := <-workersCh
	assert.Equal(t, mockWorkers, result.items)
}

func TestLeaseManager_SendQueues(t *testing.T) {
	tenantId := uuid.UUID{}
	queuesCh := make(notifierCh[string])
	leaseManager := &LeaseManager{
		tenantId: tenantId,
		queuesCh: queuesCh,
	}

	mockQueues := []string{"queue-1", "queue-2"}

	go leaseManager.sendQueues(mockQueues, false)

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
		{StepID: stepID, BatchKey: "key-1"},
		{StepID: stepID, BatchKey: "key-2"},
		{StepID: otherStepID, BatchKey: "key-3"},
	}

	leaseRepo := &fakeLeaseRepo{}
	repo := &fakeSchedulerRepo{
		leaseRepo: leaseRepo,
		batchRepo: &fakeBatchQueueFactory{resources: resources},
	}

	logger := zerolog.New(io.Discard)
	lm, _, _, _, batchesCh := newLeaseManager(&sharedConfig{repo: repo, l: &logger}, tenantID)

	received := make(chan []*sqlcv1.ListDistinctBatchResourcesRow, 1)
	ready := make(chan struct{})
	go func() {
		close(ready)
		if rows, ok := <-batchesCh; ok {
			received <- rows
		}
	}()

	<-ready

	require.NoError(t, lm.acquireBatchLeases(context.Background()))

	select {
	case rows := <-received:
		require.Len(t, rows, 3, "all batch keys for leased steps should propagate")
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
