package v2

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
)

type mockLeaseRepo struct {
	mock.Mock
}

func (m *mockLeaseRepo) ListQueues(ctx context.Context, tenantId pgtype.UUID) ([]*dbsqlc.Queue, error) {
	args := m.Called(ctx, tenantId)
	return args.Get(0).([]*dbsqlc.Queue), args.Error(1)
}

func (m *mockLeaseRepo) ListActiveWorkers(ctx context.Context, tenantId pgtype.UUID) ([]*repository.ListActiveWorkersResult, error) {
	args := m.Called(ctx, tenantId)
	return args.Get(0).([]*repository.ListActiveWorkersResult), args.Error(1)
}

func (m *mockLeaseRepo) AcquireOrExtendLeases(ctx context.Context, tenantId pgtype.UUID, kind dbsqlc.LeaseKind, resourceIds []string, existingLeases []*dbsqlc.Lease) ([]*dbsqlc.Lease, error) {
	args := m.Called(ctx, kind, resourceIds, existingLeases)
	return args.Get(0).([]*dbsqlc.Lease), args.Error(1)
}

func (m *mockLeaseRepo) RenewLeases(ctx context.Context, tenantId pgtype.UUID, leases []*dbsqlc.Lease) ([]*dbsqlc.Lease, error) {
	args := m.Called(ctx, leases)
	return args.Get(0).([]*dbsqlc.Lease), args.Error(1)
}

func (m *mockLeaseRepo) ReleaseLeases(ctx context.Context, tenantId pgtype.UUID, leases []*dbsqlc.Lease) error {
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

	mockWorkers := []*repository.ListActiveWorkersResult{
		{ID: "worker-1", Labels: nil},
		{ID: "worker-2", Labels: nil},
	}
	mockLeases := []*dbsqlc.Lease{
		{ID: 1, ResourceId: "worker-1"},
		{ID: 2, ResourceId: "worker-2"},
	}

	mockLeaseRepo.On("ListActiveWorkers", mock.Anything, tenantId).Return(mockWorkers, nil)
	mockLeaseRepo.On("AcquireOrExtendLeases", mock.Anything, dbsqlc.LeaseKindWORKER, mock.Anything, mock.Anything).Return(mockLeases, nil)

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

	mockQueues := []*dbsqlc.Queue{
		{Name: "queue-1"},
		{Name: "queue-2"},
	}
	mockLeases := []*dbsqlc.Lease{
		{ID: 1, ResourceId: "queue-1"},
		{ID: 2, ResourceId: "queue-2"},
	}

	mockLeaseRepo.On("ListQueues", mock.Anything, tenantId).Return(mockQueues, nil)
	mockLeaseRepo.On("AcquireOrExtendLeases", mock.Anything, dbsqlc.LeaseKindQUEUE, mock.Anything, mock.Anything).Return(mockLeases, nil)

	err := leaseManager.acquireQueueLeases(context.Background())
	assert.NoError(t, err)
	assert.Len(t, leaseManager.queueLeases, 2)
}

func TestLeaseManager_SendWorkerIds(t *testing.T) {
	tenantId := pgtype.UUID{}
	workersCh := make(chan []*repository.ListActiveWorkersResult)
	leaseManager := &LeaseManager{
		tenantId:  tenantId,
		workersCh: workersCh,
	}

	mockWorkers := []*repository.ListActiveWorkersResult{
		{ID: "worker-1", Labels: nil},
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
	workersCh := make(chan []*repository.ListActiveWorkersResult)
	leaseManager := &LeaseManager{
		tenantId:  tenantId,
		workersCh: workersCh,
	}

	mockWorkers1 := []*repository.ListActiveWorkersResult{
		{ID: "worker-1", Labels: nil},
	}
	mockWorkers2 := []*repository.ListActiveWorkersResult{
		{ID: "worker-1", Labels: nil},
		{ID: "worker-2", Labels: nil},
	}

	// Send workers before listener is ready
	go leaseManager.sendWorkerIds(mockWorkers1)
	time.Sleep(100 * time.Millisecond)
	resultCh := make(chan []*repository.ListActiveWorkersResult)
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
