package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlchelpers"
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

func (r *optimisticSchedulingRepositoryImpl) TriggerFromEvents(ctx context.Context, tx *OptimisticTx, tenantId string, opts []EventTriggerOpts) ([]*sqlcv1.V1QueueItem, *TriggerFromEventsResult, error) {
	pre, post := r.m.Meter(ctx, sqlcv1.LimitResourceEVENT, tenantId, int32(len(opts))) // nolint: gosec

	if err := pre(); err != nil {
		return nil, nil, err
	}

	result, err := r.doTriggerFromEvents(ctx, tx, tenantId, opts)

	if err != nil {
		return nil, nil, err
	}

	tasks := result.Tasks

	// get the queue items for the tasks that were created
	taskIds := make([]int64, 0, len(tasks))
	taskInsertedAts := make([]pgtype.Timestamptz, 0, len(tasks))
	retryCounts := make([]int32, 0, len(tasks))

	for _, task := range tasks {
		taskIds = append(taskIds, task.ID)
		taskInsertedAts = append(taskInsertedAts, task.InsertedAt)
		retryCounts = append(retryCounts, task.RetryCount)
	}

	qis, err := r.queries.ListQueueItemsForTasks(ctx, tx.tx, sqlcv1.ListQueueItemsForTasksParams{
		Tenantid:        sqlchelpers.UUIDFromStr(tenantId),
		Taskids:         taskIds,
		Taskinsertedats: taskInsertedAts,
		Retrycounts:     retryCounts,
	})

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			qis = []*sqlcv1.V1QueueItem{}
		} else {
			return nil, nil, fmt.Errorf("failed to list queue items for tasks: %w", err)
		}
	}

	tx.AddPostCommit(post)

	return qis, result, nil
}

func (r *optimisticSchedulingRepositoryImpl) TriggerFromNames(ctx context.Context, tx *OptimisticTx, tenantId string, opts []*WorkflowNameTriggerOpts) ([]*sqlcv1.V1QueueItem, []*V1TaskWithPayload, []*DAGWithData, error) {
	triggerOpts, err := r.prepareTriggerFromWorkflowNames(ctx, tx.tx, tenantId, opts)

	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to prepare trigger from workflow names: %w", err)
	}

	tasks, dags, err := r.triggerWorkflows(ctx, tx, tenantId, triggerOpts, nil)

	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to trigger workflows: %w", err)
	}

	// get the queue items for the tasks that were created
	taskIds := make([]int64, 0, len(tasks))
	taskInsertedAts := make([]pgtype.Timestamptz, 0, len(tasks))
	retryCounts := make([]int32, 0, len(tasks))

	for _, task := range tasks {
		taskIds = append(taskIds, task.ID)
		taskInsertedAts = append(taskInsertedAts, task.InsertedAt)
		retryCounts = append(retryCounts, task.RetryCount)
	}

	qis, err := r.queries.ListQueueItemsForTasks(ctx, tx.tx, sqlcv1.ListQueueItemsForTasksParams{
		Tenantid:        sqlchelpers.UUIDFromStr(tenantId),
		Taskids:         taskIds,
		Taskinsertedats: taskInsertedAts,
		Retrycounts:     retryCounts,
	})

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			qis = []*sqlcv1.V1QueueItem{}
		} else {
			return nil, nil, nil, fmt.Errorf("failed to list queue items for tasks: %w", err)
		}
	}

	return qis, tasks, dags, nil
}

func (r *optimisticSchedulingRepositoryImpl) MarkQueueItemsProcessed(ctx context.Context, tx *OptimisticTx, tenantId uuid.UUID, r2 *AssignResults) (succeeded []*AssignedItem, failed []*AssignedItem, err error) {
	return r.markQueueItemsProcessed(ctx, tenantId, r2, tx.tx, true)
}
