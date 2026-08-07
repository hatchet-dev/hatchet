package scheduler

import (
	"context"

	"github.com/google/uuid"

	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	schedulingv1 "github.com/hatchet-dev/hatchet/pkg/scheduling/v1"
)

func (s *Scheduler) RunOptimisticScheduling(ctx context.Context, tenantId uuid.UUID, opts []*v1.WorkflowNameTriggerOpts, localWorkerIds map[uuid.UUID]struct{}) (map[uuid.UUID][]*schedulingv1.AssignedItemWithTask, []v1.IdempotencyCollision, error) {
	// the trigger repository stages all OLAP messages on the trigger transaction and
	// runs the non-transactional signaling side effects post-commit
	localTasks, _, _, idempotencyKeyCollisions, err := s.pool.RunOptimisticScheduling(ctx, tenantId, opts, localWorkerIds)

	if err != nil {
		return nil, nil, err
	}

	return localTasks, idempotencyKeyCollisions, err
}

func (s *Scheduler) RunOptimisticSchedulingFromEvents(ctx context.Context, tenantId uuid.UUID, opts []v1.EventTriggerOpts, localWorkerIds map[uuid.UUID]struct{}) (map[uuid.UUID][]*schedulingv1.AssignedItemWithTask, error) {
	// the trigger repository stages all OLAP messages (including event triggers and
	// CEL evaluation failures) on the trigger transaction and runs the
	// non-transactional signaling side effects post-commit
	localTasks, _, err := s.pool.RunOptimisticSchedulingFromEvents(ctx, tenantId, opts, localWorkerIds)

	if err != nil {
		return nil, err
	}

	return localTasks, err
}
