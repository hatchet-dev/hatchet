package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

type optimisticSchedulingRepositoryImpl struct {
	*sharedRepository
}

func newOptimisticSchedulingRepository(shared *sharedRepository) *optimisticSchedulingRepositoryImpl {
	return &optimisticSchedulingRepositoryImpl{
		sharedRepository: shared,
	}
}

func (r *optimisticSchedulingRepositoryImpl) StartTx(ctx context.Context) (*OptimisticTx, error) {
	return r.PrepareOptimisticTx(ctx)
}

func (r *optimisticSchedulingRepositoryImpl) TriggerFromEvents(ctx context.Context, tx *OptimisticTx, tenantId uuid.UUID, opts []EventTriggerOpts) ([]*sqlcv1.V1QueueItem, *TriggerFromEventsResult, error) {
	pre, post := r.m.Meter(ctx, tx.tx, sqlcv1.LimitResourceEVENT, tenantId, int32(len(opts))) // nolint: gosec

	if err := pre(); err != nil {
		return nil, nil, err
	}

	result, err := r.doTriggerFromEvents(ctx, tx, tenantId, opts)

	if err != nil {
		return nil, nil, err
	}

	tx.AddPostCommit(post)

	// the queue items for the created tasks come back with CreateTasks
	return queueItemsOf(result.Tasks), result, nil
}

func (r *optimisticSchedulingRepositoryImpl) TriggerFromNames(ctx context.Context, tx *OptimisticTx, tenantId uuid.UUID, opts []*WorkflowNameTriggerOpts) ([]*sqlcv1.V1QueueItem, []*V1TaskWithPayload, []*DAGWithData, []IdempotencyCollision, error) {
	triggerOpts, err := r.prepareTriggerFromWorkflowNames(ctx, tx.tx, tenantId, opts)

	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed to prepare trigger from workflow names: %w", err)
	}

	tasks, dags, idempotencyKeyCollisions, _, err := r.triggerWorkflows(ctx, tx, tenantId, triggerOpts, nil)

	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed to trigger workflows: %w", err)
	}

	// the queue items for the created tasks come back with CreateTasks
	return queueItemsOf(tasks), tasks, dags, idempotencyKeyCollisions, nil
}

func (r *optimisticSchedulingRepositoryImpl) MarkQueueItemsProcessed(ctx context.Context, tx *OptimisticTx, tenantId uuid.UUID, r2 *AssignResults) (succeeded []*AssignedItem, failed []*AssignedItem, err error) {
	return r.markQueueItemsProcessed(ctx, tenantId, r2, tx.tx, true)
}
