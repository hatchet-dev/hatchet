package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
)

type SchedulerRepository interface {
	Lease() LeaseRepository
	QueueFactory() QueueFactoryRepository
	RateLimit() RateLimitRepository
	Assignment() AssignmentRepository
}

type ListActiveWorkersResult struct {
	ID      string
	MaxRuns int
	Labels  []*dbsqlc.ListManyWorkerLabelsRow
}

type LeaseRepository interface {
	ListQueues(ctx context.Context, tenantId pgtype.UUID) ([]*dbsqlc.Queue, error)
	ListActiveWorkers(ctx context.Context, tenantId pgtype.UUID) ([]*ListActiveWorkersResult, error)

	AcquireOrExtendLeases(ctx context.Context, tenantId pgtype.UUID, kind dbsqlc.LeaseKind, resourceIds []string, existingLeases []*dbsqlc.Lease) ([]*dbsqlc.Lease, error)
	ReleaseLeases(ctx context.Context, tenantId pgtype.UUID, leases []*dbsqlc.Lease) error
}

type RateLimitResult struct {
	ExceededKey   string
	ExceededUnits int32
	ExceededVal   int32
	StepRunId     pgtype.UUID
}

type AssignedItem struct {
	WorkerId pgtype.UUID

	QueueItem *dbsqlc.QueueItem
}

type AssignResults struct {
	Assigned           []*AssignedItem
	Unassigned         []*dbsqlc.QueueItem
	SchedulingTimedOut []*dbsqlc.QueueItem
	RateLimited        []*RateLimitResult
}

type QueueFactoryRepository interface {
	NewQueue(tenantId pgtype.UUID, queueName string) QueueRepository
}

type QueueRepository interface {
	ListQueueItems(ctx context.Context, limit int) ([]*dbsqlc.QueueItem, error)
	MarkQueueItemsProcessed(ctx context.Context, r *AssignResults) (succeeded []*AssignedItem, failed []*AssignedItem, err error)
	GetStepRunRateLimits(ctx context.Context, queueItems []*dbsqlc.QueueItem) (map[string]map[string]int32, error)
	GetDesiredLabels(ctx context.Context, stepIds []pgtype.UUID) (map[string][]*dbsqlc.GetDesiredLabelsRow, error)
	Cleanup()
}

type RateLimitRepository interface {
	ListCandidateRateLimits(ctx context.Context, tenantId pgtype.UUID) ([]string, error)
	UpdateRateLimits(ctx context.Context, tenantId pgtype.UUID, updates map[string]int) (map[string]int, error)
}

type AssignmentRepository interface {
	ListActionsForWorkers(ctx context.Context, tenantId pgtype.UUID, workerIds []pgtype.UUID) ([]*dbsqlc.ListActionsForWorkersRow, error)
	ListAvailableSlotsForWorkers(ctx context.Context, tenantId pgtype.UUID, params dbsqlc.ListAvailableSlotsForWorkersParams) ([]*dbsqlc.ListAvailableSlotsForWorkersRow, error)
}
