package v2

import (
	"context"

	"github.com/hatchet-dev/hatchet/pkg/repository/v2/sqlcv2"
	"github.com/jackc/pgx/v5/pgtype"
)

type SchedulerRepository interface {
	Lease() LeaseRepository
	QueueFactory() QueueFactoryRepository
	RateLimit() RateLimitRepository
	Assignment() AssignmentRepository
}

type LeaseRepository interface {
	ListQueues(ctx context.Context, tenantId pgtype.UUID) ([]*sqlcv2.V2Queue, error)
	ListActiveWorkers(ctx context.Context, tenantId pgtype.UUID) ([]*ListActiveWorkersResult, error)

	AcquireOrExtendLeases(ctx context.Context, tenantId pgtype.UUID, kind sqlcv2.LeaseKind, resourceIds []string, existingLeases []*sqlcv2.Lease) ([]*sqlcv2.Lease, error)
	ReleaseLeases(ctx context.Context, tenantId pgtype.UUID, leases []*sqlcv2.Lease) error
}

type QueueFactoryRepository interface {
	NewQueue(tenantId pgtype.UUID, queueName string) QueueRepository
}

type QueueRepository interface {
	ListQueueItems(ctx context.Context, limit int) ([]*sqlcv2.V2QueueItem, error)
	MarkQueueItemsProcessed(ctx context.Context, r *AssignResults) (succeeded []*AssignedItem, failed []*AssignedItem, err error)

	// TODO: ADD THIS
	// GetStepRunRateLimits(ctx context.Context, queueItems []*sqlcv2.V2QueueItem) (map[string]map[string]int32, error)
	GetDesiredLabels(ctx context.Context, stepIds []pgtype.UUID) (map[string][]*sqlcv2.GetDesiredLabelsRow, error)
	Cleanup()
}

type RateLimitRepository interface {
	ListCandidateRateLimits(ctx context.Context, tenantId pgtype.UUID) ([]string, error)
	UpdateRateLimits(ctx context.Context, tenantId pgtype.UUID, updates map[string]int) (map[string]int, error)
}

type AssignmentRepository interface {
	ListActionsForWorkers(ctx context.Context, tenantId pgtype.UUID, workerIds []pgtype.UUID) ([]*sqlcv2.ListActionsForWorkersRow, error)
	ListAvailableSlotsForWorkers(ctx context.Context, tenantId pgtype.UUID, params sqlcv2.ListAvailableSlotsForWorkersParams) ([]*sqlcv2.ListAvailableSlotsForWorkersRow, error)
}

type schedulerRepository struct {
	lease        LeaseRepository
	queueFactory QueueFactoryRepository
	rateLimit    RateLimitRepository
	assignment   AssignmentRepository
}

func newSchedulerRepository(shared *sharedRepository) *schedulerRepository {
	return &schedulerRepository{
		lease:        newLeaseRepository(shared),
		queueFactory: newQueueFactoryRepository(shared),
		rateLimit:    newRateLimitRepository(shared),
		assignment:   newAssignmentRepository(shared),
	}
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
