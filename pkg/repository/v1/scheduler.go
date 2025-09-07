package v1

import (
	"context"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"
	"github.com/jackc/pgx/v5/pgtype"
)

type SchedulerRepository interface {
	Concurrency() ConcurrencyRepository
	Lease() LeaseRepository
	QueueFactory() QueueFactoryRepository
	RateLimit() RateLimitRepository
	Assignment() AssignmentRepository
	Optimistic() OptimisticSchedulingRepository
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

type QueueRepository interface {
	ListQueueItems(ctx context.Context, limit int) ([]*sqlcv1.V1QueueItem, error)
	MarkQueueItemsProcessed(ctx context.Context, r *AssignResults) (succeeded []*AssignedItem, failed []*AssignedItem, err error)

	GetTaskRateLimits(ctx context.Context, queueItems []*sqlcv1.V1QueueItem) (map[int64]map[string]int32, error)
	RequeueRateLimitedItems(ctx context.Context, tenantId pgtype.UUID, queueName string) ([]*sqlcv1.RequeueRateLimitedQueueItemsRow, error)
	GetDesiredLabels(ctx context.Context, stepIds []pgtype.UUID) (map[string][]*sqlcv1.GetDesiredLabelsRow, error)
	Cleanup()
}

type RateLimitRepository interface {
	UpdateRateLimits(ctx context.Context, tenantId pgtype.UUID, updates map[string]int) ([]*sqlcv1.ListRateLimitsForTenantWithMutateRow, *time.Time, error)
}

type AssignmentRepository interface {
	ListActionsForWorkers(ctx context.Context, tenantId pgtype.UUID, workerIds []pgtype.UUID) ([]*sqlcv1.ListActionsForWorkersRow, error)
	ListAvailableSlotsForWorkers(ctx context.Context, tenantId pgtype.UUID, params sqlcv1.ListAvailableSlotsForWorkersParams) ([]*sqlcv1.ListAvailableSlotsForWorkersRow, error)
}

type OptimisticTx struct {
	tx         sqlcv1.DBTX
	commit     func(ctx context.Context) error
	rollback   func()
	postCommit []func()
}

func (s *sharedRepository) PrepareOptimisticTx(ctx context.Context) (*OptimisticTx, error) {
	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, s.pool, s.l)

	if err != nil {
		return nil, err
	}

	return &OptimisticTx{
		tx:         tx,
		commit:     commit,
		rollback:   rollback,
		postCommit: make([]func(), 0),
	}, nil
}

func (o *OptimisticTx) AddPostCommit(f func()) {
	o.postCommit = append(o.postCommit, f)
}

func (o *OptimisticTx) Commit(ctx context.Context) error {
	err := o.commit(ctx)

	if err != nil {
		return err
	}

	for _, f := range o.postCommit {
		f()
	}

	return err
}

func (o *OptimisticTx) Rollback() {
	o.rollback()
}

type OptimisticSchedulingRepository interface {
	StartTx(ctx context.Context) (*OptimisticTx, error)

	TriggerFromEvents(ctx context.Context, tx *OptimisticTx, tenantId string, opts []EventTriggerOpts) ([]*sqlcv1.V1QueueItem, *TriggerFromEventsResult, error)

	TriggerFromNames(ctx context.Context, tx *OptimisticTx, tenantId pgtype.UUID, opts []*WorkflowNameTriggerOpts) ([]*sqlcv1.V1QueueItem, []*sqlcv1.V1Task, []*DAGWithData, error)

	MarkQueueItemsProcessed(ctx context.Context, tx *OptimisticTx, tenantId pgtype.UUID, r *AssignResults) (succeeded []*AssignedItem, failed []*AssignedItem, err error)
}

type schedulerRepository struct {
	concurrency  ConcurrencyRepository
	lease        LeaseRepository
	queueFactory QueueFactoryRepository
	rateLimit    RateLimitRepository
	assignment   AssignmentRepository
	optimistic   OptimisticSchedulingRepository
}

func newSchedulerRepository(shared *sharedRepository) *schedulerRepository {
	return &schedulerRepository{
		concurrency:  newConcurrencyRepository(shared),
		lease:        newLeaseRepository(shared),
		queueFactory: newQueueFactoryRepository(shared),
		rateLimit:    newRateLimitRepository(shared),
		assignment:   newAssignmentRepository(shared),
		optimistic:   newOptimisticSchedulingRepository(shared),
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

func (d *schedulerRepository) Optimistic() OptimisticSchedulingRepository {
	return d.optimistic
}
