package v1

import (
	"context"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"
	"github.com/jackc/pgx/v5/pgtype"
)

type SchedulerRepository interface {
	Concurrency() ConcurrencyRepository
	Lease() LeaseRepository
	QueueFactory() QueueFactoryRepository
	BatchQueue() BatchQueueFactoryRepository
	RateLimit() RateLimitRepository
	Assignment() AssignmentRepository
}

type LeaseRepository interface {
	ListQueues(ctx context.Context, tenantId pgtype.UUID) ([]*sqlcv1.V1Queue, error)
	ListActiveWorkers(ctx context.Context, tenantId pgtype.UUID) ([]*ListActiveWorkersResult, error)
	ListConcurrencyStrategies(ctx context.Context, tenantId pgtype.UUID) ([]*sqlcv1.V1StepConcurrency, error)

	AcquireOrExtendLeases(ctx context.Context, tenantId pgtype.UUID, kind sqlcv1.LeaseKind, resourceIds []string, existingLeases []*sqlcv1.Lease) ([]*sqlcv1.Lease, error)
	ReleaseLeases(ctx context.Context, tenantId pgtype.UUID, leases []*sqlcv1.Lease) error
}

type QueueFactoryRepository interface {
	NewQueue(tenantId pgtype.UUID, queueName string) QueueRepository
}

type BatchQueueFactoryRepository interface {
	NewBatchQueue(tenantId pgtype.UUID) BatchQueueRepository
}

type QueueRepository interface {
	ListQueueItems(ctx context.Context, limit int) ([]*sqlcv1.V1QueueItem, error)
	MarkQueueItemsProcessed(ctx context.Context, r *AssignResults) (succeeded []*AssignedItem, failed []*AssignedItem, err error)

	GetTaskRateLimits(ctx context.Context, queueItems []*sqlcv1.V1QueueItem) (map[int64]map[string]int32, error)
	GetStepBatchConfigs(ctx context.Context, stepIds []pgtype.UUID) (map[string]bool, error)
	RequeueRateLimitedItems(ctx context.Context, tenantId pgtype.UUID, queueName string) ([]*sqlcv1.RequeueRateLimitedQueueItemsRow, error)
	GetDesiredLabels(ctx context.Context, stepIds []pgtype.UUID) (map[string][]*sqlcv1.GetDesiredLabelsRow, error)
	Cleanup()
}

type BatchQueueRepository interface {
	ListBatchResources(ctx context.Context) ([]*sqlcv1.ListDistinctBatchResourcesRow, error)
	ListBatchedQueueItems(ctx context.Context, stepId pgtype.UUID, batchKey string, afterId pgtype.Int8, limit int32) ([]*sqlcv1.V1BatchedQueueItem, error)
	ListExistingBatchedQueueItemIds(ctx context.Context, ids []int64) (map[int64]struct{}, error)
	DeleteBatchedQueueItems(ctx context.Context, ids []int64) error
	MoveBatchedQueueItems(ctx context.Context, ids []int64) ([]*sqlcv1.MoveBatchedQueueItemsRow, error)
	CommitAssignments(ctx context.Context, assignments []*BatchAssignment) ([]*BatchAssignment, error)
}

type BatchAssignment struct {
	BatchQueueItemID int64
	TaskID           int64
	TaskInsertedAt   pgtype.Timestamptz
	RetryCount       int32
	WorkerID         pgtype.UUID

	BatchID  string
	StepID   pgtype.UUID
	ActionID string
	BatchKey string
}

type RateLimitRepository interface {
	UpdateRateLimits(ctx context.Context, tenantId pgtype.UUID, updates map[string]int) ([]*sqlcv1.ListRateLimitsForTenantWithMutateRow, *time.Time, error)
}

type AssignmentRepository interface {
	ListActionsForWorkers(ctx context.Context, tenantId pgtype.UUID, workerIds []pgtype.UUID) ([]*sqlcv1.ListActionsForWorkersRow, error)
	ListAvailableSlotsForWorkers(ctx context.Context, tenantId pgtype.UUID, params sqlcv1.ListAvailableSlotsForWorkersParams) ([]*sqlcv1.ListAvailableSlotsForWorkersRow, error)
}

type schedulerRepository struct {
	concurrency  ConcurrencyRepository
	lease        LeaseRepository
	queueFactory QueueFactoryRepository
	batchQueue   BatchQueueFactoryRepository
	rateLimit    RateLimitRepository
	assignment   AssignmentRepository
}

func newSchedulerRepository(shared *sharedRepository) *schedulerRepository {
	return &schedulerRepository{
		concurrency:  newConcurrencyRepository(shared),
		lease:        newLeaseRepository(shared),
		queueFactory: newQueueFactoryRepository(shared),
		batchQueue:   newBatchQueueFactoryRepository(shared),
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

func (d *schedulerRepository) BatchQueue() BatchQueueFactoryRepository {
	return d.batchQueue
}

func (d *schedulerRepository) RateLimit() RateLimitRepository {
	return d.rateLimit
}

func (d *schedulerRepository) Assignment() AssignmentRepository {
	return d.assignment
}
