package scheduling

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// QueueResults is the result set a queuer emits after flushing a batch of
// scheduling decisions to the database.
type QueueResults struct {
	TenantId uuid.UUID
	Assigned []*v1.AssignedItem
	Buffered []*v1.AssignedItem

	Unassigned         []*sqlcv1.V1QueueItem
	SchedulingTimedOut []*sqlcv1.V1QueueItem
	RateLimited        []*v1.RateLimitResult
}

// ConcurrencyResults is the result set a concurrency strategy emits after a run.
type ConcurrencyResults struct {
	*v1.RunConcurrencyResult

	TenantId uuid.UUID
}

// AssignedItemWithTask pairs an optimistic scheduling assignment with its task.
type AssignedItemWithTask struct {
	AssignedItem *v1.AssignedItem
	Task         *v1.V1TaskWithPayload
}

// Sentinel errors returned from optimistic scheduling.
var ErrTenantNotFound = fmt.Errorf("tenant not found in pool")
var ErrNoOptimisticSlots = fmt.Errorf("no optimistic slots for scheduling")

// Pool is the engine-facing surface of a tenant scheduling pool.
type Pool interface {
	GetResultsCh() chan *QueueResults
	GetConcurrencyResultsCh() chan *ConcurrencyResults

	// AddExtension registers a scheduler extension (metrics, autoscaling, ...)
	// against the pool.
	AddExtension(ext SchedulerExtension)

	SetTenants(tenants []*sqlcv1.Tenant)
	Replenish(ctx context.Context, tenantId uuid.UUID)

	NotifyQueues(ctx context.Context, tenantId uuid.UUID, queueNames []string)
	NotifyConcurrency(ctx context.Context, tenantId uuid.UUID, strategyIds []int64)
	NotifyNewWorker(ctx context.Context, tenantId uuid.UUID, workerId uuid.UUID)
	NotifyNewQueue(ctx context.Context, tenantId uuid.UUID, queueName string)
	NotifyNewConcurrencyStrategy(ctx context.Context, tenantId uuid.UUID, strategyId int64)

	RunOptimisticScheduling(ctx context.Context, tenantId uuid.UUID, opts []*v1.WorkflowNameTriggerOpts, localWorkerIds map[uuid.UUID]struct{}) (map[uuid.UUID][]*AssignedItemWithTask, []*v1.V1TaskWithPayload, []*v1.DAGWithData, []v1.IdempotencyCollision, error)
	RunOptimisticSchedulingFromEvents(ctx context.Context, tenantId uuid.UUID, opts []v1.EventTriggerOpts, localWorkerIds map[uuid.UUID]struct{}) (map[uuid.UUID][]*AssignedItemWithTask, *v1.TriggerFromEventsResult, error)
}
