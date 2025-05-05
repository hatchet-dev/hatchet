//go:build !e2e && !load && !rampup && !integration

package v2

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	v1 "github.com/hatchet-dev/hatchet/pkg/repository/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"
)

type mockLeaseRepo struct {
	mock.Mock
}

func (m *mockLeaseRepo) ListQueues(ctx context.Context, tenantId pgtype.UUID) ([]*sqlcv1.V1Queue, error) {
	args := m.Called(ctx, tenantId)
	return args.Get(0).([]*sqlcv1.V1Queue), args.Error(1)
}

func (m *mockLeaseRepo) ListActiveWorkers(ctx context.Context, tenantId pgtype.UUID) ([]*v1.ListActiveWorkersResult, error) {
	args := m.Called(ctx, tenantId)
	return args.Get(0).([]*v1.ListActiveWorkersResult), args.Error(1)
}

func (m *mockLeaseRepo) ListConcurrencyStrategies(ctx context.Context, tenantId pgtype.UUID) ([]*sqlcv1.V1StepConcurrency, error) {
	args := m.Called(ctx, tenantId)
	return args.Get(0).([]*sqlcv1.V1StepConcurrency), args.Error(1)
}

func (m *mockLeaseRepo) AcquireOrExtendLeases(ctx context.Context, tenantId pgtype.UUID, kind sqlcv1.LeaseKind, resourceIds []string, existingLeases []*sqlcv1.Lease) ([]*sqlcv1.Lease, error) {
	args := m.Called(ctx, kind, resourceIds, existingLeases)
	return args.Get(0).([]*sqlcv1.Lease), args.Error(1)
}

func (m *mockLeaseRepo) RenewLeases(ctx context.Context, tenantId pgtype.UUID, leases []*sqlcv1.Lease) ([]*sqlcv1.Lease, error) {
	args := m.Called(ctx, leases)
	return args.Get(0).([]*sqlcv1.Lease), args.Error(1)
}

func (m *mockLeaseRepo) ReleaseLeases(ctx context.Context, tenantId pgtype.UUID, leases []*sqlcv1.Lease) error {
	args := m.Called(ctx, leases)
	return args.Error(0)
}

func TestLeaseManager_AcquireWorkerLeases(t *testing.T) {
	l := zerolog.Nop()
	tenantId := pgtype.UUID{}
	mockLeaseRepo := &mockLeaseRepo{}
	leaseManager := &LeaseManager{
		lr:       mockLeaseRepo,
		conf:     &sharedConfig{l: &l},
		tenantId: tenantId,
	}

	mockWorkers := []*v1.ListActiveWorkersResult{
		{ID: uuid.NewString(), Labels: nil},
		{ID: uuid.NewString(), Labels: nil},
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
	tenantId := pgtype.UUID{}
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
	tenantId := pgtype.UUID{}
	workersCh := make(chan []*v1.ListActiveWorkersResult)
	leaseManager := &LeaseManager{
		tenantId:  tenantId,
		workersCh: workersCh,
	}

	mockWorkers := []*v1.ListActiveWorkersResult{
		{ID: uuid.NewString(), Labels: nil},
	}

	go leaseManager.sendWorkerIds(mockWorkers)

	result := <-workersCh
	assert.Equal(t, mockWorkers, result)
}

func TestLeaseManager_SendQueues(t *testing.T) {
	tenantId := pgtype.UUID{}
	queuesCh := make(chan []string)
	leaseManager := &LeaseManager{
		tenantId: tenantId,
		queuesCh: queuesCh,
	}

	mockQueues := []string{"queue-1", "queue-2"}

	go leaseManager.sendQueues(mockQueues)

	result := <-queuesCh
	assert.Equal(t, mockQueues, result)
}

func TestLeaseManager_AcquireWorkersBeforeListenerReady(t *testing.T) {
	tenantId := pgtype.UUID{}
	workersCh := make(chan []*v1.ListActiveWorkersResult)
	leaseManager := &LeaseManager{
		tenantId:  tenantId,
		workersCh: workersCh,
	}

	mockWorkers1 := []*v1.ListActiveWorkersResult{
		{ID: uuid.NewString(), Labels: nil},
	}
	mockWorkers2 := []*v1.ListActiveWorkersResult{
		{ID: uuid.NewString(), Labels: nil},
		{ID: uuid.NewString(), Labels: nil},
	}

	// Send workers before listener is ready
	go leaseManager.sendWorkerIds(mockWorkers1)
	time.Sleep(100 * time.Millisecond)
	resultCh := make(chan []*v1.ListActiveWorkersResult)
	go func() {
		resultCh <- <-workersCh
	}()
	time.Sleep(100 * time.Millisecond)
	go leaseManager.sendWorkerIds(mockWorkers2)
	time.Sleep(100 * time.Millisecond)

	// Ensure only the latest workers are sent over the channel
	result := <-resultCh
	assert.Equal(t, mockWorkers2, result)
	assert.Len(t, workersCh, 0) // Ensure no additional workers are left in the channel
}
