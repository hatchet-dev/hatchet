package repository

import (
	"context"

	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

type SchedulerRepository interface {
	Concurrency() ConcurrencyRepository
	Lease() LeaseRepository
	QueueFactory() QueueFactoryRepository
	RateLimit() RateLimitRepository
	Assignment() AssignmentRepository
}

type LeaseRepository interface {
	ListQueues(ctx context.Context, tenantId uuid.UUID) ([]*sqlcv1.V1Queue, error)
	ListActiveWorkers(ctx context.Context, tenantId uuid.UUID) ([]*ListActiveWorkersResult, error)
	ListConcurrencyStrategies(ctx context.Context, tenantId uuid.UUID) ([]*sqlcv1.V1StepConcurrency, error)

	AcquireOrExtendLeases(ctx context.Context, tenantId uuid.UUID, kind sqlcv1.LeaseKind, resourceIds []string, existingLeases []*sqlcv1.Lease) ([]*sqlcv1.Lease, error)
	ReleaseLeases(ctx context.Context, tenantId uuid.UUID, leases []*sqlcv1.Lease) error
}

type QueueFactoryRepository interface {
	NewQueue(tenantId uuid.UUID, queueName string) QueueRepository
}

type QueueRepository interface {
	ListQueueItems(ctx context.Context, limit int) ([]*sqlcv1.V1QueueItem, error)
	MarkQueueItemsProcessed(ctx context.Context, r *AssignResults) (succeeded []*AssignedItem, failed []*AssignedItem, err error)

	GetTaskRateLimits(ctx context.Context, queueItems []*sqlcv1.V1QueueItem) (map[int64]map[string]int32, error)
	RequeueRateLimitedItems(ctx context.Context, tenantId uuid.UUID, queueName string) ([]*sqlcv1.RequeueRateLimitedQueueItemsRow, error)
	GetDesiredLabels(ctx context.Context, stepIds []uuid.UUID) (map[string][]*sqlcv1.GetDesiredLabelsRow, error)
	Cleanup()
}

type AssignmentRepository interface {
	ListActionsForWorkers(ctx context.Context, tenantId uuid.UUID, workerIds []uuid.UUID) ([]*sqlcv1.ListActionsForWorkersRow, error)
	ListAvailableSlotsForWorkers(ctx context.Context, tenantId uuid.UUID, params sqlcv1.ListAvailableSlotsForWorkersParams) ([]*sqlcv1.ListAvailableSlotsForWorkersRow, error)
}

type schedulerRepository struct {
	concurrency  ConcurrencyRepository
	lease        LeaseRepository
	queueFactory QueueFactoryRepository
	rateLimit    RateLimitRepository
	assignment   AssignmentRepository
}

func newSchedulerRepository(shared *sharedRepository) *schedulerRepository {
	return &schedulerRepository{
		concurrency:  newConcurrencyRepository(shared),
		lease:        newLeaseRepository(shared),
		queueFactory: newQueueFactoryRepository(shared),
		rateLimit:    newRateLimitRepository(shared),
		assignment:   newAssignmentRepository(shared),
	}
}

func (d *schedulerRepository) Concurrency() ConcurrencyRepository {
	return d.concurrency
}

func (d *schedulerRepository) Lease() LeaseRepository {
	return d.lease
}

func (d *schedulerRepository) QueueFactory() QueueFactoryRepository {
	return d.queueFactory
}

func (d *schedulerRepository) RateLimit() RateLimitRepository {
	return d.rateLimit
}

func (d *schedulerRepository) Assignment() AssignmentRepository {
	return d.assignment
}
