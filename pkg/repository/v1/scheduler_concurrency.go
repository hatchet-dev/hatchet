package v1

import (
	"context"
	"fmt"

	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"
	"github.com/jackc/pgx/v5/pgtype"
)

const PARENT_STRATEGY_LOCK_OFFSET = 1000000000000 // 1 trillion

type TaskWithQueue struct {
	*TaskIdInsertedAtRetryCount

	Queue string
}

type TaskWithCancelledReason struct {
	*TaskIdInsertedAtRetryCount

	CancelledReason string

	TaskExternalId string

	WorkflowRunId string
}

type RunConcurrencyResult struct {
	// The tasks which were enqueued
	Queued []TaskWithQueue

	// If the strategy involves cancelling a task, these are the tasks to cancel
	Cancelled []TaskWithCancelledReason

	// If the step has multiple concurrency strategies, these are the next ones to notify
	NextConcurrencyStrategies []int64
}

type ConcurrencyRepository interface {
	// Checks whether the concurrency strategy is active, and if not, sets is_active=False
	UpdateConcurrencyStrategyIsActive(ctx context.Context, tenantId pgtype.UUID, strategy *sqlcv1.V1StepConcurrency) error

	RunConcurrencyStrategy(ctx context.Context, tenantId pgtype.UUID, strategy *sqlcv1.V1StepConcurrency) (*RunConcurrencyResult, error)
}

type ConcurrencyRepositoryImpl struct {
	*sharedRepository
}

func newConcurrencyRepository(s *sharedRepository) ConcurrencyRepository {
	return &ConcurrencyRepositoryImpl{
		sharedRepository: s,
	}
}

func (c *ConcurrencyRepositoryImpl) UpdateConcurrencyStrategyIsActive(
	ctx context.Context,
	tenantId pgtype.UUID,
	strategy *sqlcv1.V1StepConcurrency,
) error {
	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, c.pool, c.l, 5000)

	if err != nil {
		return err
	}

	defer rollback()

	err = c.queries.ConcurrencyAdvisoryLock(ctx, tx, strategy.ID)

	if err != nil {
		return err
	}

	isActive, err := c.queries.CheckStrategyActive(ctx, tx, sqlcv1.CheckStrategyActiveParams{
		Tenantid:          tenantId,
		Workflowid:        strategy.WorkflowID,
		Workflowversionid: strategy.WorkflowVersionID,
		Strategyid:        strategy.ID,
	})

	if err != nil {
		return err
	}

	if !isActive {
		err = c.queries.SetConcurrencyStrategyInactive(ctx, tx, sqlcv1.SetConcurrencyStrategyInactiveParams{
			Workflowid:        strategy.WorkflowID,
			Workflowversionid: strategy.WorkflowVersionID,
			Stepid:            strategy.StepID,
			Strategyid:        strategy.ID,
		})

		if err != nil {
			return err
		}
	}

	return commit(ctx)
}

func (c *ConcurrencyRepositoryImpl) RunConcurrencyStrategy(
	ctx context.Context,
	tenantId pgtype.UUID,
	strategy *sqlcv1.V1StepConcurrency,
) (res *RunConcurrencyResult, err error) {
	switch strategy.Strategy {
	case sqlcv1.V1ConcurrencyStrategyGROUPROUNDROBIN:
		return c.runGroupRoundRobin(ctx, tenantId, strategy)
	case sqlcv1.V1ConcurrencyStrategyCANCELINPROGRESS:
		return c.runCancelInProgress(ctx, tenantId, strategy)
	case sqlcv1.V1ConcurrencyStrategyCANCELNEWEST:
		return c.runCancelNewest(ctx, tenantId, strategy)
	}

	return nil, nil
}

func (c *ConcurrencyRepositoryImpl) runGroupRoundRobin(
	ctx context.Context,
	tenantId pgtype.UUID,
	strategy *sqlcv1.V1StepConcurrency,
) (res *RunConcurrencyResult, err error) {
	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, c.pool, c.l, 5000)

	if err != nil {
		return nil, err
	}

	defer rollback()

	err = c.queries.ConcurrencyAdvisoryLock(ctx, tx, strategy.ID)

	if err != nil {
		return nil, err
	}

	var queued []TaskWithQueue
	var cancelled []TaskWithCancelledReason
	var nextConcurrencyStrategies []int64

	if strategy.ParentStrategyID.Valid {
		acquired, err := c.queries.TryConcurrencyAdvisoryLock(ctx, tx, PARENT_STRATEGY_LOCK_OFFSET+strategy.ParentStrategyID.Int64)

		if err != nil {
			return nil, err
		}

		if acquired {
			err = c.queries.RunParentGroupRoundRobin(ctx, tx, sqlcv1.RunParentGroupRoundRobinParams{
				Tenantid:   tenantId,
				Strategyid: strategy.ParentStrategyID.Int64,
				Maxruns:    strategy.MaxConcurrency,
			})

			if err != nil {
				return nil, err
			}
		}

		poppedResults, err := c.queries.RunChildGroupRoundRobin(ctx, tx, sqlcv1.RunChildGroupRoundRobinParams{
			Tenantid:         tenantId,
			Strategyid:       strategy.ID,
			Parentstrategyid: strategy.ParentStrategyID.Int64,
		})

		if err != nil {
			return nil, err
		}

		queued = make([]TaskWithQueue, 0, len(poppedResults))
		cancelled = make([]TaskWithCancelledReason, 0, len(poppedResults))
		nextConcurrencyStrategies = make([]int64, 0, len(poppedResults))

		for _, r := range poppedResults {
			idRetryCount := &TaskIdInsertedAtRetryCount{
				Id:         r.TaskID,
				InsertedAt: r.TaskInsertedAt,
				RetryCount: r.TaskRetryCount,
			}

			switch {
			case len(r.NextStrategyIds) > 0:
				nextConcurrencyStrategies = append(nextConcurrencyStrategies, r.NextStrategyIds[0])
			case r.Operation == "SCHEDULING_TIMED_OUT":
				cancelled = append(cancelled, TaskWithCancelledReason{
					TaskIdInsertedAtRetryCount: idRetryCount,
					CancelledReason:            "SCHEDULING_TIMED_OUT",
					TaskExternalId:             sqlchelpers.UUIDToStr(r.ExternalID),
					WorkflowRunId:              sqlchelpers.UUIDToStr(r.WorkflowRunID),
				})
			default:
				queued = append(queued, TaskWithQueue{
					TaskIdInsertedAtRetryCount: &TaskIdInsertedAtRetryCount{
						Id:         r.TaskID,
						InsertedAt: r.TaskInsertedAt,
						RetryCount: r.TaskRetryCount,
					},
					Queue: r.QueueToNotify,
				})
			}
		}
	} else {
		poppedResults, err := c.queries.RunGroupRoundRobin(ctx, tx, sqlcv1.RunGroupRoundRobinParams{
			Tenantid:   tenantId,
			Strategyid: strategy.ID,
			Maxruns:    strategy.MaxConcurrency,
		})

		if err != nil {
			return nil, err
		}

		queued = make([]TaskWithQueue, 0, len(poppedResults))
		cancelled = make([]TaskWithCancelledReason, 0, len(poppedResults))
		nextConcurrencyStrategies = make([]int64, 0, len(poppedResults))

		for _, r := range poppedResults {
			idRetryCount := &TaskIdInsertedAtRetryCount{
				Id:         r.TaskID,
				InsertedAt: r.TaskInsertedAt,
				RetryCount: r.TaskRetryCount,
			}

			switch {
			case len(r.NextStrategyIds) > 0:
				nextConcurrencyStrategies = append(nextConcurrencyStrategies, r.NextStrategyIds[0])
			case r.Operation == "SCHEDULING_TIMED_OUT":
				cancelled = append(cancelled, TaskWithCancelledReason{
					TaskIdInsertedAtRetryCount: idRetryCount,
					CancelledReason:            "SCHEDULING_TIMED_OUT",
					TaskExternalId:             sqlchelpers.UUIDToStr(r.ExternalID),
					WorkflowRunId:              sqlchelpers.UUIDToStr(r.WorkflowRunID),
				})
			default:
				queued = append(queued, TaskWithQueue{
					TaskIdInsertedAtRetryCount: &TaskIdInsertedAtRetryCount{
						Id:         r.TaskID,
						InsertedAt: r.TaskInsertedAt,
						RetryCount: r.TaskRetryCount,
					},
					Queue: r.QueueToNotify,
				})
			}
		}
	}

	if err = commit(ctx); err != nil {
		return nil, err
	}

	return &RunConcurrencyResult{
		Queued:                    queued,
		Cancelled:                 cancelled,
		NextConcurrencyStrategies: nextConcurrencyStrategies,
	}, nil
}

func (c *ConcurrencyRepositoryImpl) runCancelInProgress(
	ctx context.Context,
	tenantId pgtype.UUID,
	strategy *sqlcv1.V1StepConcurrency,
) (res *RunConcurrencyResult, err error) {
	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, c.pool, c.l, 5000)

	if err != nil {
		return nil, err
	}

	defer rollback()

	err = c.queries.ConcurrencyAdvisoryLock(ctx, tx, strategy.ID)

	if err != nil {
		return nil, err
	}

	var queued []TaskWithQueue
	var cancelled []TaskWithCancelledReason
	var nextConcurrencyStrategies []int64

	if strategy.ParentStrategyID.Valid {
		err := c.queries.ConcurrencyAdvisoryLock(ctx, tx, PARENT_STRATEGY_LOCK_OFFSET+strategy.ParentStrategyID.Int64)

		if err != nil {
			return nil, err
		}

		_, err = tx.Exec(
			ctx,
			`
-- name: CreateParentTempTable :exec
CREATE TEMP TABLE tmp_workflow_concurrency_slot ON COMMIT DROP AS
SELECT *
FROM v1_workflow_concurrency_slot
WHERE tenant_id = $1::uuid AND strategy_id = $2::bigint;`,
			tenantId,
			strategy.ParentStrategyID.Int64,
		)

		if err != nil {
			return nil, fmt.Errorf("error creating parent temp table: %w", err)
		}

		err = c.queries.RunParentCancelInProgress(ctx, tx, sqlcv1.RunParentCancelInProgressParams{
			Tenantid:   tenantId,
			Strategyid: strategy.ParentStrategyID.Int64,
			Maxruns:    strategy.MaxConcurrency,
		})

		if err != nil {
			return nil, fmt.Errorf("error running parent cancel in progress: %w", err)
		}

		poppedResults, err := c.queries.RunChildCancelInProgress(ctx, tx, sqlcv1.RunChildCancelInProgressParams{
			Tenantid:   tenantId,
			Strategyid: strategy.ID,
			Maxruns:    strategy.MaxConcurrency,
		})

		if err != nil {
			return nil, fmt.Errorf("error running child cancel in progress: %w", err)
		}

		queued = make([]TaskWithQueue, 0, len(poppedResults))
		cancelled = make([]TaskWithCancelledReason, 0, len(poppedResults))
		nextConcurrencyStrategies = make([]int64, 0, len(poppedResults))

		for _, r := range poppedResults {
			idRetryCount := &TaskIdInsertedAtRetryCount{
				Id:         r.TaskID,
				InsertedAt: r.TaskInsertedAt,
				RetryCount: r.TaskRetryCount,
			}

			switch {
			case len(r.NextStrategyIds) > 0:
				nextConcurrencyStrategies = append(nextConcurrencyStrategies, r.NextStrategyIds[0])
			case r.Operation == "CANCELLED":
				cancelled = append(cancelled, TaskWithCancelledReason{
					TaskIdInsertedAtRetryCount: idRetryCount,
					CancelledReason:            "CONCURRENCY_LIMIT",
					TaskExternalId:             sqlchelpers.UUIDToStr(r.ExternalID),
					WorkflowRunId:              sqlchelpers.UUIDToStr(r.WorkflowRunID),
				})
			case r.Operation == "SCHEDULING_TIMED_OUT":
				cancelled = append(cancelled, TaskWithCancelledReason{
					TaskIdInsertedAtRetryCount: idRetryCount,
					CancelledReason:            "SCHEDULING_TIMED_OUT",
					TaskExternalId:             sqlchelpers.UUIDToStr(r.ExternalID),
					WorkflowRunId:              sqlchelpers.UUIDToStr(r.WorkflowRunID),
				})
			default:
				queued = append(queued, TaskWithQueue{
					TaskIdInsertedAtRetryCount: &TaskIdInsertedAtRetryCount{
						Id:         r.TaskID,
						InsertedAt: r.TaskInsertedAt,
						RetryCount: r.TaskRetryCount,
					},
					Queue: r.QueueToNotify,
				})
			}
		}
	} else {
		poppedResults, err := c.queries.RunCancelInProgress(ctx, tx, sqlcv1.RunCancelInProgressParams{
			Tenantid:   tenantId,
			Strategyid: strategy.ID,
			Maxruns:    strategy.MaxConcurrency,
		})

		if err != nil {
			return nil, err
		}

		// for any cancelled tasks, call cancelTasks
		cancelledTasks := make([]TaskIdInsertedAtRetryCount, 0, len(poppedResults))

		for _, r := range poppedResults {
			if r.Operation == "CANCELLED" {
				cancelledTasks = append(cancelledTasks, TaskIdInsertedAtRetryCount{
					Id:         r.TaskID,
					InsertedAt: r.TaskInsertedAt,
					RetryCount: r.TaskRetryCount,
				})
			}
		}

		taskIds := make([]int64, len(cancelledTasks))
		retryCounts := make([]int32, len(cancelledTasks))

		for i, task := range cancelledTasks {
			taskIds[i] = task.Id
			retryCounts[i] = task.RetryCount
		}

		// remove tasks from queue
		err = c.queries.DeleteTasksFromQueue(ctx, tx, sqlcv1.DeleteTasksFromQueueParams{
			Taskids:     taskIds,
			Retrycounts: retryCounts,
		})

		if err != nil {
			return nil, err
		}

		queued = make([]TaskWithQueue, 0, len(poppedResults))
		cancelled = make([]TaskWithCancelledReason, 0, len(poppedResults))
		nextConcurrencyStrategies = make([]int64, 0, len(poppedResults))

		for _, r := range poppedResults {
			idRetryCount := &TaskIdInsertedAtRetryCount{
				Id:         r.TaskID,
				InsertedAt: r.TaskInsertedAt,
				RetryCount: r.TaskRetryCount,
			}

			switch {
			case len(r.NextStrategyIds) > 0:
				nextConcurrencyStrategies = append(nextConcurrencyStrategies, r.NextStrategyIds[0])
			case r.Operation == "CANCELLED":
				cancelled = append(cancelled, TaskWithCancelledReason{
					TaskIdInsertedAtRetryCount: idRetryCount,
					CancelledReason:            "CONCURRENCY_LIMIT",
					TaskExternalId:             sqlchelpers.UUIDToStr(r.ExternalID),
					WorkflowRunId:              sqlchelpers.UUIDToStr(r.WorkflowRunID),
				})
			case r.Operation == "SCHEDULING_TIMED_OUT":
				cancelled = append(cancelled, TaskWithCancelledReason{
					TaskIdInsertedAtRetryCount: idRetryCount,
					CancelledReason:            "SCHEDULING_TIMED_OUT",
					TaskExternalId:             sqlchelpers.UUIDToStr(r.ExternalID),
					WorkflowRunId:              sqlchelpers.UUIDToStr(r.WorkflowRunID),
				})
			default:
				queued = append(queued, TaskWithQueue{
					TaskIdInsertedAtRetryCount: &TaskIdInsertedAtRetryCount{
						Id:         r.TaskID,
						InsertedAt: r.TaskInsertedAt,
						RetryCount: r.TaskRetryCount,
					},
					Queue: r.QueueToNotify,
				})
			}
		}
	}

	if err = commit(ctx); err != nil {
		return nil, err
	}

	return &RunConcurrencyResult{
		Queued:                    queued,
		Cancelled:                 cancelled,
		NextConcurrencyStrategies: nextConcurrencyStrategies,
	}, nil
}

func (c *ConcurrencyRepositoryImpl) runCancelNewest(
	ctx context.Context,
	tenantId pgtype.UUID,
	strategy *sqlcv1.V1StepConcurrency,
) (res *RunConcurrencyResult, err error) {
	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, c.pool, c.l, 5000)

	if err != nil {
		return nil, err
	}

	defer rollback()

	err = c.queries.ConcurrencyAdvisoryLock(ctx, tx, strategy.ID)

	if err != nil {
		return nil, err
	}

	var queued []TaskWithQueue
	var cancelled []TaskWithCancelledReason
	var nextConcurrencyStrategies []int64

	if strategy.ParentStrategyID.Valid {
		err := c.queries.ConcurrencyAdvisoryLock(ctx, tx, PARENT_STRATEGY_LOCK_OFFSET+strategy.ParentStrategyID.Int64)

		if err != nil {
			return nil, err
		}

		_, err = tx.Exec(
			ctx,
			`
-- name: CreateParentTempTable :exec
CREATE TEMP TABLE tmp_workflow_concurrency_slot ON COMMIT DROP AS
SELECT *
FROM v1_workflow_concurrency_slot
WHERE tenant_id = $1::uuid AND strategy_id = $2::bigint;`,
			tenantId,
			strategy.ParentStrategyID.Int64,
		)

		if err != nil {
			return nil, fmt.Errorf("error creating parent temp table: %w", err)
		}

		err = c.queries.RunParentCancelNewest(ctx, tx, sqlcv1.RunParentCancelNewestParams{
			Tenantid:   tenantId,
			Strategyid: strategy.ParentStrategyID.Int64,
			Maxruns:    strategy.MaxConcurrency,
		})

		if err != nil {
			return nil, fmt.Errorf("error running parent cancel newest: %w", err)
		}

		poppedResults, err := c.queries.RunChildCancelNewest(ctx, tx, sqlcv1.RunChildCancelNewestParams{
			Tenantid:   tenantId,
			Strategyid: strategy.ID,
			Maxruns:    strategy.MaxConcurrency,
		})

		if err != nil {
			return nil, err
		}

		// for any cancelled tasks, call cancelTasks
		cancelledTasks := make([]TaskIdInsertedAtRetryCount, 0, len(poppedResults))

		for _, r := range poppedResults {
			if r.Operation == "CANCELLED" {
				cancelledTasks = append(cancelledTasks, TaskIdInsertedAtRetryCount{
					Id:         r.TaskID,
					InsertedAt: r.TaskInsertedAt,
					RetryCount: r.TaskRetryCount,
				})
			}
		}

		taskIds := make([]int64, len(cancelledTasks))
		retryCounts := make([]int32, len(cancelledTasks))

		for i, task := range cancelledTasks {
			taskIds[i] = task.Id
			retryCounts[i] = task.RetryCount
		}

		// remove tasks from queue
		err = c.queries.DeleteTasksFromQueue(ctx, tx, sqlcv1.DeleteTasksFromQueueParams{
			Taskids:     taskIds,
			Retrycounts: retryCounts,
		})

		if err != nil {
			return nil, err
		}

		queued = make([]TaskWithQueue, 0, len(poppedResults))
		cancelled = make([]TaskWithCancelledReason, 0, len(poppedResults))
		nextConcurrencyStrategies = make([]int64, 0, len(poppedResults))

		for _, r := range poppedResults {
			idRetryCount := &TaskIdInsertedAtRetryCount{
				Id:         r.TaskID,
				InsertedAt: r.TaskInsertedAt,
				RetryCount: r.TaskRetryCount,
			}

			switch {
			case len(r.NextStrategyIds) > 0:
				nextConcurrencyStrategies = append(nextConcurrencyStrategies, r.NextStrategyIds[0])
			case r.Operation == "CANCELLED":
				cancelled = append(cancelled, TaskWithCancelledReason{
					TaskIdInsertedAtRetryCount: idRetryCount,
					CancelledReason:            "CONCURRENCY_LIMIT",
					TaskExternalId:             sqlchelpers.UUIDToStr(r.ExternalID),
					WorkflowRunId:              sqlchelpers.UUIDToStr(r.WorkflowRunID),
				})
			case r.Operation == "SCHEDULING_TIMED_OUT":
				cancelled = append(cancelled, TaskWithCancelledReason{
					TaskIdInsertedAtRetryCount: idRetryCount,
					CancelledReason:            "SCHEDULING_TIMED_OUT",
					TaskExternalId:             sqlchelpers.UUIDToStr(r.ExternalID),
					WorkflowRunId:              sqlchelpers.UUIDToStr(r.WorkflowRunID),
				})
			default:
				queued = append(queued, TaskWithQueue{
					TaskIdInsertedAtRetryCount: &TaskIdInsertedAtRetryCount{
						Id:         r.TaskID,
						InsertedAt: r.TaskInsertedAt,
						RetryCount: r.TaskRetryCount,
					},
					Queue: r.QueueToNotify,
				})
			}
		}
	} else {
		poppedResults, err := c.queries.RunCancelNewest(ctx, tx, sqlcv1.RunCancelNewestParams{
			Tenantid:   tenantId,
			Strategyid: strategy.ID,
			Maxruns:    strategy.MaxConcurrency,
		})

		if err != nil {
			return nil, err
		}

		// for any cancelled tasks, call cancelTasks
		cancelledTasks := make([]TaskIdInsertedAtRetryCount, 0, len(poppedResults))

		for _, r := range poppedResults {
			if r.Operation == "CANCELLED" {
				cancelledTasks = append(cancelledTasks, TaskIdInsertedAtRetryCount{
					Id:         r.TaskID,
					InsertedAt: r.TaskInsertedAt,
					RetryCount: r.TaskRetryCount,
				})
			}
		}

		taskIds := make([]int64, len(cancelledTasks))
		retryCounts := make([]int32, len(cancelledTasks))

		for i, task := range cancelledTasks {
			taskIds[i] = task.Id
			retryCounts[i] = task.RetryCount
		}

		// remove tasks from queue
		err = c.queries.DeleteTasksFromQueue(ctx, tx, sqlcv1.DeleteTasksFromQueueParams{
			Taskids:     taskIds,
			Retrycounts: retryCounts,
		})

		if err != nil {
			return nil, err
		}

		queued = make([]TaskWithQueue, 0, len(poppedResults))
		cancelled = make([]TaskWithCancelledReason, 0, len(poppedResults))
		nextConcurrencyStrategies = make([]int64, 0, len(poppedResults))

		for _, r := range poppedResults {
			idRetryCount := &TaskIdInsertedAtRetryCount{
				Id:         r.TaskID,
				InsertedAt: r.TaskInsertedAt,
				RetryCount: r.TaskRetryCount,
			}

			switch {
			case len(r.NextStrategyIds) > 0:
				nextConcurrencyStrategies = append(nextConcurrencyStrategies, r.NextStrategyIds[0])
			case r.Operation == "CANCELLED":
				cancelled = append(cancelled, TaskWithCancelledReason{
					TaskIdInsertedAtRetryCount: idRetryCount,
					CancelledReason:            "CONCURRENCY_LIMIT",
					TaskExternalId:             sqlchelpers.UUIDToStr(r.ExternalID),
					WorkflowRunId:              sqlchelpers.UUIDToStr(r.WorkflowRunID),
				})
			case r.Operation == "SCHEDULING_TIMED_OUT":
				cancelled = append(cancelled, TaskWithCancelledReason{
					TaskIdInsertedAtRetryCount: idRetryCount,
					CancelledReason:            "SCHEDULING_TIMED_OUT",
					TaskExternalId:             sqlchelpers.UUIDToStr(r.ExternalID),
					WorkflowRunId:              sqlchelpers.UUIDToStr(r.WorkflowRunID),
				})
			default:
				queued = append(queued, TaskWithQueue{
					TaskIdInsertedAtRetryCount: &TaskIdInsertedAtRetryCount{
						Id:         r.TaskID,
						InsertedAt: r.TaskInsertedAt,
						RetryCount: r.TaskRetryCount,
					},
					Queue: r.QueueToNotify,
				})
			}
		}
	}

	if err = commit(ctx); err != nil {
		return nil, err
	}

	return &RunConcurrencyResult{
		Queued:                    queued,
		Cancelled:                 cancelled,
		NextConcurrencyStrategies: nextConcurrencyStrategies,
	}, nil
}
