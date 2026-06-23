package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

const PARENT_STRATEGY_LOCK_OFFSET = 1000000000000 // 1 trillion

// Cancellation reasons surfaced on TaskWithCancelledReason. Kept as plain strings to match the
// values the downstream consumer (notifyAfterConcurrency) and the legacy SQL paths compare against.
const (
	CancelledReasonConcurrencyLimit   = "CONCURRENCY_LIMIT"
	CancelledReasonSchedulingTimedOut = "SCHEDULING_TIMED_OUT"
)

type TaskWithQueue struct {
	*TaskIdInsertedAtRetryCount

	Queue string
}

type TaskWithCancelledReason struct {
	*TaskIdInsertedAtRetryCount

	CancelledReason string

	TaskExternalId uuid.UUID

	WorkflowRunId uuid.UUID
}

// CancelledSlotInput identifies a concurrency slot to cancel along with the reason it's being
// cancelled, so UpdateConcurrencySlots can propagate the reason into the RunConcurrencyResult.
type CancelledSlotInput struct {
	CancelledReason string
	TaskIdInsertedAtRetryCount
}

type RunConcurrencyResult struct {
	// The tasks which were enqueued
	Queued []TaskWithQueue

	// If the strategy involves cancelling a task, these are the tasks to cancel
	Cancelled []TaskWithCancelledReason

	// If the step has multiple concurrency strategies, these are the next ones to notify
	NextConcurrencyStrategies []int64

	FailedAdvisoryLock bool
}

type ConcurrencyRepository interface {
	// Checks whether the concurrency strategy is active, and if not, sets is_active=False
	UpdateConcurrencyStrategyIsActive(ctx context.Context, tenantId uuid.UUID, strategy *sqlcv1.V1StepConcurrency) error

	RunConcurrencyStrategy(ctx context.Context, tenantId uuid.UUID, strategy *sqlcv1.V1StepConcurrency) (*RunConcurrencyResult, error)

	DeactivateStaleStepConcurrency(ctx context.Context, tenantId uuid.UUID) error

	ListTenantsWithManyStepConcurrencies(ctx context.Context, threshold int64) ([]*sqlcv1.ListTenantsWithManyStepConcurrenciesRow, error)

	ReadConcurrencySlotsForIndexing(ctx context.Context, tenantId uuid.UUID, strategyId int64, writeCh chan<- *sqlcv1.ListConcurrencySlotsForIndexingRow) error

	// UpdateConcurrencySlots manages its own transaction, for callers (e.g. the post-build queueing
	// pass) that have no transaction to attach to. Callers that already hold a transaction (e.g. the
	// WAL flush riding the outbox transaction) should use UpdateConcurrencySlotsTx instead.
	UpdateConcurrencySlots(
		ctx context.Context,
		tenantId uuid.UUID,
		strategyId int64,
		filledSlots []TaskIdInsertedAtRetryCount,
		cancelledSlots []CancelledSlotInput,
	) (*RunConcurrencyResult, error)

	// UpdateConcurrencySlotsTx runs the slot updates within the provided transaction, so the writes
	// commit (or roll back) atomically with whatever else the caller is doing in that transaction.
	UpdateConcurrencySlotsTx(
		ctx context.Context,
		tx pgx.Tx,
		tenantId uuid.UUID,
		strategyId int64,
		filledSlots []TaskIdInsertedAtRetryCount,
		cancelledSlots []CancelledSlotInput,
	) (*RunConcurrencyResult, error)
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
	tenantId uuid.UUID,
	strategy *sqlcv1.V1StepConcurrency,
) error {
	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, c.pool, c.l)

	if err != nil {
		return err
	}

	defer rollback()

	err = c.queries.AdvisoryLock(ctx, tx, strategy.ID)

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
	tenantId uuid.UUID,
	strategy *sqlcv1.V1StepConcurrency,
) (res *RunConcurrencyResult, err error) {
	switch strategy.Strategy {
	case sqlcv1.V1ConcurrencyStrategyGROUPROUNDROBIN:
		res, err = c.runGroupRoundRobin(ctx, tenantId, strategy)

		if err != nil {
			return nil, fmt.Errorf("group round robin (strategy ID: %d): %w", strategy.ID, err)
		}
	case sqlcv1.V1ConcurrencyStrategyCANCELINPROGRESS:
		res, err = c.runCancelInProgress(ctx, tenantId, strategy)

		if err != nil {
			return nil, fmt.Errorf("cancel in progress (strategy ID: %d): %w", strategy.ID, err)
		}
	case sqlcv1.V1ConcurrencyStrategyCANCELNEWEST:
		res, err = c.runCancelNewest(ctx, tenantId, strategy)

		if err != nil {
			return nil, fmt.Errorf("cancel newest (strategy ID: %d): %w", strategy.ID, err)
		}
	}

	return res, nil
}

func (c *ConcurrencyRepositoryImpl) runGroupRoundRobin(
	ctx context.Context,
	tenantId uuid.UUID,
	strategy *sqlcv1.V1StepConcurrency,
) (res *RunConcurrencyResult, err error) {
	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, c.pool, c.l)

	if err != nil {
		return nil, fmt.Errorf("failed to prepare transaction (strategy ID: %d): %w", strategy.ID, err)
	}

	defer rollback()

	err = c.queries.AdvisoryLock(ctx, tx, strategy.ID)

	if err != nil {
		return nil, fmt.Errorf("failed to acquire advisory lock (strategy ID: %d): %w", strategy.ID, err)
	}

	var queued []TaskWithQueue
	var cancelled []TaskWithCancelledReason
	var nextConcurrencyStrategies []int64

	if strategy.ParentStrategyID.Valid {
		acquired, err := c.queries.TryAdvisoryLock(ctx, tx, PARENT_STRATEGY_LOCK_OFFSET+strategy.ParentStrategyID.Int64)

		if err != nil {
			return nil, fmt.Errorf("failed to acquire parent advisory lock (strategy ID: %d, parent: %d): %w", strategy.ID, strategy.ParentStrategyID.Int64, err)
		}

		if acquired {
			err = c.queries.RunParentGroupRoundRobin(ctx, tx, sqlcv1.RunParentGroupRoundRobinParams{
				Tenantid:   tenantId,
				Strategyid: strategy.ParentStrategyID.Int64,
				Maxruns:    strategy.MaxConcurrency,
			})

			if err != nil {
				return nil, fmt.Errorf("failed to run parent group round robin (strategy ID: %d, parent: %d): %w", strategy.ID, strategy.ParentStrategyID.Int64, err)
			}
		}

		poppedResults, err := c.queries.RunChildGroupRoundRobin(ctx, tx, sqlcv1.RunChildGroupRoundRobinParams{
			Tenantid:         tenantId,
			Strategyid:       strategy.ID,
			Parentstrategyid: strategy.ParentStrategyID.Int64,
		})

		if err != nil {
			return nil, fmt.Errorf("failed to run child group round robin (strategy ID: %d, parent: %d): %w", strategy.ID, strategy.ParentStrategyID.Int64, err)
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
			case r.Operation == "SCHEDULING_TIMED_OUT":
				cancelled = append(cancelled, TaskWithCancelledReason{
					TaskIdInsertedAtRetryCount: idRetryCount,
					CancelledReason:            "SCHEDULING_TIMED_OUT",
					TaskExternalId:             r.ExternalID,
					WorkflowRunId:              r.WorkflowRunID,
				})
			case len(r.NextStrategyIds) > 0:
				nextConcurrencyStrategies = append(nextConcurrencyStrategies, r.NextStrategyIds[0])
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
			return nil, fmt.Errorf("failed to run group round robin: %w", err)
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
			case r.Operation == "SCHEDULING_TIMED_OUT":
				cancelled = append(cancelled, TaskWithCancelledReason{
					TaskIdInsertedAtRetryCount: idRetryCount,
					CancelledReason:            "SCHEDULING_TIMED_OUT",
					TaskExternalId:             r.ExternalID,
					WorkflowRunId:              r.WorkflowRunID,
				})
			case len(r.NextStrategyIds) > 0:
				nextConcurrencyStrategies = append(nextConcurrencyStrategies, r.NextStrategyIds[0])
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

	err = c.upsertQueuesForQueuedTasks(ctx, tx, tenantId, queued)
	if err != nil {
		return nil, fmt.Errorf("failed to upsert queues for queued tasks (strategy ID: %d): %w", strategy.ID, err)
	}

	if err = commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction (strategy ID: %d): %w", strategy.ID, err)
	}

	return &RunConcurrencyResult{
		Queued:                    queued,
		Cancelled:                 cancelled,
		NextConcurrencyStrategies: nextConcurrencyStrategies,
		FailedAdvisoryLock:        false,
	}, nil
}

func (c *ConcurrencyRepositoryImpl) runCancelInProgress(
	ctx context.Context,
	tenantId uuid.UUID,
	strategy *sqlcv1.V1StepConcurrency,
) (res *RunConcurrencyResult, err error) {

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, c.pool, c.l)

	if err != nil {
		return nil, fmt.Errorf("failed to prepare transaction (strategy ID: %d): %w", strategy.ID, err)
	}

	defer rollback()

	acquired, err := c.queries.TryAdvisoryLock(ctx, tx, strategy.ID)

	if err != nil {
		return nil, fmt.Errorf("failed to acquire advisory lock (strategy ID: %d): %w", strategy.ID, err)
	}

	if !acquired {
		c.l.Warn().Ctx(ctx).Msgf("Advisory lock not acquired (strategy ID: %d). Possible lock contention.", strategy.ID)

		return &RunConcurrencyResult{
			Queued:                    []TaskWithQueue{},
			Cancelled:                 []TaskWithCancelledReason{},
			NextConcurrencyStrategies: []int64{},
			FailedAdvisoryLock:        true,
		}, nil
	}

	var queued []TaskWithQueue
	var cancelled []TaskWithCancelledReason
	var nextConcurrencyStrategies []int64

	if strategy.ParentStrategyID.Valid {
		acquired, err := c.queries.TryAdvisoryLock(ctx, tx, PARENT_STRATEGY_LOCK_OFFSET+strategy.ParentStrategyID.Int64)

		if !acquired {
			c.l.Warn().Ctx(ctx).Msgf("Advisory lock not acquired (strategy ID: %d). Possible lock contention.", strategy.ID)

			return &RunConcurrencyResult{
				Queued:                    []TaskWithQueue{},
				Cancelled:                 []TaskWithCancelledReason{},
				NextConcurrencyStrategies: []int64{},
				FailedAdvisoryLock:        true,
			}, nil
		}

		if err != nil {
			return nil, fmt.Errorf("failed to acquire parent advisory lock (strategy ID: %d, parent: %d): %w", strategy.ID, strategy.ParentStrategyID.Int64, err)
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
			return nil, fmt.Errorf("error creating parent temp table (strategy ID: %d, parent: %d): %w", strategy.ID, strategy.ParentStrategyID.Int64, err)
		}

		err = c.queries.RunParentCancelInProgress(ctx, tx, sqlcv1.RunParentCancelInProgressParams{
			Tenantid:   tenantId,
			Strategyid: strategy.ParentStrategyID.Int64,
			Maxruns:    strategy.MaxConcurrency,
		})

		if err != nil {
			return nil, fmt.Errorf("error running parent cancel in progress (strategy ID: %d, parent: %d): %w", strategy.ID, strategy.ParentStrategyID.Int64, err)
		}

		poppedResults, err := c.queries.RunChildCancelInProgress(ctx, tx, sqlcv1.RunChildCancelInProgressParams{
			Tenantid:   tenantId,
			Strategyid: strategy.ID,
			Maxruns:    strategy.MaxConcurrency,
		})

		if err != nil {
			return nil, fmt.Errorf("error running child cancel in progress (strategy ID: %d): %w", strategy.ID, err)
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
			case r.Operation == "CANCELLED":
				cancelled = append(cancelled, TaskWithCancelledReason{
					TaskIdInsertedAtRetryCount: idRetryCount,
					CancelledReason:            "CONCURRENCY_LIMIT",
					TaskExternalId:             r.ExternalID,
					WorkflowRunId:              r.WorkflowRunID,
				})
			case r.Operation == "SCHEDULING_TIMED_OUT":
				cancelled = append(cancelled, TaskWithCancelledReason{
					TaskIdInsertedAtRetryCount: idRetryCount,
					CancelledReason:            "SCHEDULING_TIMED_OUT",
					TaskExternalId:             r.ExternalID,
					WorkflowRunId:              r.WorkflowRunID,
				})
			case len(r.NextStrategyIds) > 0:
				nextConcurrencyStrategies = append(nextConcurrencyStrategies, r.NextStrategyIds[0])
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
			return nil, fmt.Errorf("error running cancel in progress (strategy ID: %d): %w", strategy.ID, err)
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
			return nil, fmt.Errorf("error deleting tasks from queue (strategy ID: %d): %w", strategy.ID, err)
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
			case r.Operation == "CANCELLED":
				cancelled = append(cancelled, TaskWithCancelledReason{
					TaskIdInsertedAtRetryCount: idRetryCount,
					CancelledReason:            "CONCURRENCY_LIMIT",
					TaskExternalId:             r.ExternalID,
					WorkflowRunId:              r.WorkflowRunID,
				})
			case r.Operation == "SCHEDULING_TIMED_OUT":
				cancelled = append(cancelled, TaskWithCancelledReason{
					TaskIdInsertedAtRetryCount: idRetryCount,
					CancelledReason:            "SCHEDULING_TIMED_OUT",
					TaskExternalId:             r.ExternalID,
					WorkflowRunId:              r.WorkflowRunID,
				})
			case len(r.NextStrategyIds) > 0:
				nextConcurrencyStrategies = append(nextConcurrencyStrategies, r.NextStrategyIds[0])
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

	err = c.upsertQueuesForQueuedTasks(ctx, tx, tenantId, queued)
	if err != nil {
		return nil, fmt.Errorf("failed to upsert queues for queued tasks (strategy ID: %d): %w", strategy.ID, err)
	}

	if err = commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction (strategy ID: %d): %w", strategy.ID, err)
	}

	return &RunConcurrencyResult{
		Queued:                    queued,
		Cancelled:                 cancelled,
		NextConcurrencyStrategies: nextConcurrencyStrategies,
		FailedAdvisoryLock:        false,
	}, nil
}

func (c *ConcurrencyRepositoryImpl) runCancelNewest(
	ctx context.Context,
	tenantId uuid.UUID,
	strategy *sqlcv1.V1StepConcurrency,
) (res *RunConcurrencyResult, err error) {
	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, c.pool, c.l)

	if err != nil {
		return nil, fmt.Errorf("failed to prepare transaction (strategy ID: %d): %w", strategy.ID, err)
	}

	defer rollback()

	// Use TryAdvisoryLock instead of blocking lock to reduce contention
	acquired, err := c.queries.TryAdvisoryLock(ctx, tx, strategy.ID)

	if err != nil {
		return nil, fmt.Errorf("failed to try advisory lock (strategy ID: %d): %w", strategy.ID, err)
	}

	if !acquired {
		// Log lock contention issue
		c.l.Warn().Ctx(ctx).Msgf("Advisory lock not acquired (strategy ID: %d). Possible lock contention.", strategy.ID)
		// Lock not available, return empty result to avoid blocking
		return &RunConcurrencyResult{
			Queued:                    []TaskWithQueue{},
			Cancelled:                 []TaskWithCancelledReason{},
			NextConcurrencyStrategies: []int64{},
			FailedAdvisoryLock:        true,
		}, nil
	}

	var queued []TaskWithQueue
	var cancelled []TaskWithCancelledReason
	var nextConcurrencyStrategies []int64

	if strategy.ParentStrategyID.Valid {
		// Also use TryAdvisoryLock for parent strategy
		parentAcquired, err := c.queries.TryAdvisoryLock(ctx, tx, PARENT_STRATEGY_LOCK_OFFSET+strategy.ParentStrategyID.Int64)

		if err != nil {
			return nil, fmt.Errorf("failed to try parent advisory lock (strategy ID: %d, parent: %d): %w", strategy.ID, strategy.ParentStrategyID.Int64, err)
		}

		if !parentAcquired {
			// Log the event when the parent advisory lock is not acquired
			c.l.Warn().Ctx(ctx).Msgf("Parent advisory lock not acquired (strategy ID: %d, parent: %d)", strategy.ID, strategy.ParentStrategyID.Int64)
			// Parent lock not available, return empty result
			return &RunConcurrencyResult{
				Queued:                    []TaskWithQueue{},
				Cancelled:                 []TaskWithCancelledReason{},
				NextConcurrencyStrategies: []int64{},
				FailedAdvisoryLock:        true,
			}, nil
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
			return nil, fmt.Errorf("error creating parent temp table (strategy ID: %d, parent: %d): %w", strategy.ID, strategy.ParentStrategyID.Int64, err)
		}

		err = c.queries.RunParentCancelNewest(ctx, tx, sqlcv1.RunParentCancelNewestParams{
			Tenantid:   tenantId,
			Strategyid: strategy.ParentStrategyID.Int64,
			Maxruns:    strategy.MaxConcurrency,
		})

		if err != nil {
			return nil, fmt.Errorf("error running parent cancel newest (strategy ID: %d, parent: %d): %w", strategy.ID, strategy.ParentStrategyID.Int64, err)
		}

		poppedResults, err := c.queries.RunChildCancelNewest(ctx, tx, sqlcv1.RunChildCancelNewestParams{
			Tenantid:   tenantId,
			Strategyid: strategy.ID,
			Maxruns:    strategy.MaxConcurrency,
		})

		if err != nil {
			return nil, fmt.Errorf("error running child cancel newest (strategy ID: %d): %w", strategy.ID, err)
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
			return nil, fmt.Errorf("error deleting tasks from queue (strategy ID: %d): %w", strategy.ID, err)
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
			case r.Operation == "CANCELLED":
				cancelled = append(cancelled, TaskWithCancelledReason{
					TaskIdInsertedAtRetryCount: idRetryCount,
					CancelledReason:            "CONCURRENCY_LIMIT",
					TaskExternalId:             r.ExternalID,
					WorkflowRunId:              r.WorkflowRunID,
				})
			case r.Operation == "SCHEDULING_TIMED_OUT":
				cancelled = append(cancelled, TaskWithCancelledReason{
					TaskIdInsertedAtRetryCount: idRetryCount,
					CancelledReason:            "SCHEDULING_TIMED_OUT",
					TaskExternalId:             r.ExternalID,
					WorkflowRunId:              r.WorkflowRunID,
				})
			case len(r.NextStrategyIds) > 0:
				nextConcurrencyStrategies = append(nextConcurrencyStrategies, r.NextStrategyIds[0])
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
			return nil, fmt.Errorf("error running cancel newest (strategy ID: %d): %w", strategy.ID, err)
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
			return nil, fmt.Errorf("error deleting tasks from queue (strategy ID: %d): %w", strategy.ID, err)
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
			case r.Operation == "CANCELLED":
				cancelled = append(cancelled, TaskWithCancelledReason{
					TaskIdInsertedAtRetryCount: idRetryCount,
					CancelledReason:            "CONCURRENCY_LIMIT",
					TaskExternalId:             r.ExternalID,
					WorkflowRunId:              r.WorkflowRunID,
				})
			case r.Operation == "SCHEDULING_TIMED_OUT":
				cancelled = append(cancelled, TaskWithCancelledReason{
					TaskIdInsertedAtRetryCount: idRetryCount,
					CancelledReason:            "SCHEDULING_TIMED_OUT",
					TaskExternalId:             r.ExternalID,
					WorkflowRunId:              r.WorkflowRunID,
				})
			case len(r.NextStrategyIds) > 0:
				nextConcurrencyStrategies = append(nextConcurrencyStrategies, r.NextStrategyIds[0])
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

	err = c.upsertQueuesForQueuedTasks(ctx, tx, tenantId, queued)
	if err != nil {
		return nil, fmt.Errorf("failed to upsert queues for queued tasks (strategy ID: %d): %w", strategy.ID, err)
	}

	if err = commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction (strategy ID: %d): %w", strategy.ID, err)
	}

	return &RunConcurrencyResult{
		Queued:                    queued,
		Cancelled:                 cancelled,
		NextConcurrencyStrategies: nextConcurrencyStrategies,
		FailedAdvisoryLock:        false,
	}, nil
}

func (c *ConcurrencyRepositoryImpl) upsertQueuesForQueuedTasks(ctx context.Context, tx sqlcv1.DBTX, tenantId uuid.UUID, queuedTasks []TaskWithQueue) error {
	uniqueQueues := make(map[string]bool, len(queuedTasks))
	queueList := make([]string, 0, len(queuedTasks))
	for _, queue := range queuedTasks {
		if _, ok := uniqueQueues[queue.Queue]; ok {
			continue
		}
		uniqueQueues[queue.Queue] = true
		queueList = append(queueList, queue.Queue)
	}

	_, err := c.upsertQueues(ctx, tx, tenantId, queueList)
	if err != nil {
		return fmt.Errorf("failed to upsert queues: %w", err)
	}

	return nil
}

func (c *ConcurrencyRepositoryImpl) DeactivateStaleStepConcurrency(ctx context.Context, tenantId uuid.UUID) error {
	return c.queries.DeactivateStaleStepConcurrency(ctx, c.pool, tenantId)
}

func (c *ConcurrencyRepositoryImpl) ListTenantsWithManyStepConcurrencies(ctx context.Context, threshold int64) ([]*sqlcv1.ListTenantsWithManyStepConcurrenciesRow, error) {
	return c.queries.ListTenantsWithManyStepConcurrencies(ctx, c.pool, threshold)
}

func (c *ConcurrencyRepositoryImpl) ReadConcurrencySlotsForIndexing(ctx context.Context, tenantId uuid.UUID, strategyId int64, writeCh chan<- *sqlcv1.ListConcurrencySlotsForIndexingRow) error {
	// we don't want to hold the transaction open if we're blocked on the write channel, so we use this
	// context to escape the <- writeCh loop and close the tx
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	tx, commit, rollback, err := sqlchelpers.PrepareTxWithStatementTimeout(ctx, c.pool, c.l, 1000*60*5) // 5 minute timeout

	if err != nil {
		return fmt.Errorf("failed to prepare transaction: %w", err)
	}

	defer rollback()

	offset := 0

	for {
		slots, err := c.queries.ListConcurrencySlotsForIndexing(ctx, tx, sqlcv1.ListConcurrencySlotsForIndexingParams{
			Tenantid:   tenantId,
			Strategyid: strategyId,
			Offset:     int32(offset),
			Limit:      10000,
		})

		if err != nil {
			return fmt.Errorf("error reading concurrency slots for indexing: %w", err)
		}

		if len(slots) == 0 {
			break
		}

		for _, slot := range slots {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case writeCh <- slot:
			}
		}

		offset += len(slots)
	}

	if err := commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (c *ConcurrencyRepositoryImpl) UpdateConcurrencySlotsTx(
	ctx context.Context,
	tx pgx.Tx,
	tenantId uuid.UUID,
	strategyId int64,
	filledSlots []TaskIdInsertedAtRetryCount,
	cancelledSlots []CancelledSlotInput,
) (*RunConcurrencyResult, error) {
	updateArgs := make([]sqlcv1.UpdateConcurrencySlotIsFilledParams, len(filledSlots))

	for i, slot := range filledSlots {
		updateArgs[i] = sqlcv1.UpdateConcurrencySlotIsFilledParams{
			IsFilled:       true,
			TaskID:         slot.Id,
			TaskInsertedAt: slot.InsertedAt,
			TaskRetryCount: slot.RetryCount,
			StrategyID:     strategyId,
		}
	}

	filledRows, err := c.queries.UpdateConcurrencySlotIsFilledBatch(ctx, tx, updateArgs)

	if err != nil {
		return nil, fmt.Errorf("failed to update concurrency slots: %w", err)
	}

	// build the queued / next-strategy results from the slots we just filled, mirroring
	// the output contract of runGroupRoundRobin
	queued := make([]TaskWithQueue, 0, len(filledRows))
	nextConcurrencyStrategies := make([]int64, 0, len(filledRows))

	for _, row := range filledRows {
		// if the slot hands off to a downstream strategy, notify it instead of enqueuing the task
		if len(row.NextStrategyIds) > 0 {
			nextConcurrencyStrategies = append(nextConcurrencyStrategies, row.NextStrategyIds[0])
			continue
		}

		queued = append(queued, TaskWithQueue{
			TaskIdInsertedAtRetryCount: &TaskIdInsertedAtRetryCount{
				Id:         row.TaskID,
				InsertedAt: row.TaskInsertedAt,
				RetryCount: row.TaskRetryCount,
			},
			Queue: row.QueueToNotify,
		})
	}

	// flatten to the plain task identifiers releaseTasks expects, and keep a lookup of each task's
	// cancellation reason so we can propagate it onto the result below.
	type reasonKey struct {
		id         int64
		retryCount int32
	}

	tasksToCancel := make([]TaskIdInsertedAtRetryCount, 0, len(cancelledSlots))
	reasonByTask := make(map[reasonKey]string, len(cancelledSlots))

	for _, slot := range cancelledSlots {
		tasksToCancel = append(tasksToCancel, slot.TaskIdInsertedAtRetryCount)
		reasonByTask[reasonKey{id: slot.Id, retryCount: slot.RetryCount}] = slot.CancelledReason
	}

	// releaseTasks requires a 1:1 match between input tasks and returned rows, so dedupe first.
	tasksToCancel = uniqueSet(tasksToCancel)

	// note: we'd prefer to call cancelTasks here, but keeping this consistent with the previous concurrency
	// implementation. the only important thing is that we delete v1_concurrency_slot, but we need to release
	// other scheduling resources in a precise order which releaseTasks respects, otherwise we deadlock.
	releasedTasks, err := c.releaseTasks(ctx, tx, tenantId, tasksToCancel)

	if err != nil {
		return nil, fmt.Errorf("failed to release tasks: %w", err)
	}

	cancelled := make([]TaskWithCancelledReason, 0, len(releasedTasks))

	for _, released := range releasedTasks {
		reason, ok := reasonByTask[reasonKey{id: released.ID, retryCount: released.RetryCount}]
		if !ok {
			reason = CancelledReasonConcurrencyLimit
		}

		cancelled = append(cancelled, TaskWithCancelledReason{
			TaskIdInsertedAtRetryCount: &TaskIdInsertedAtRetryCount{
				Id:         released.ID,
				InsertedAt: released.InsertedAt,
				RetryCount: released.RetryCount,
			},
			CancelledReason: reason,
			TaskExternalId:  released.ExternalID,
			WorkflowRunId:   released.WorkflowRunID,
		})
	}

	if err := c.upsertQueuesForQueuedTasks(ctx, tx, tenantId, queued); err != nil {
		return nil, fmt.Errorf("failed to upsert queues for queued tasks: %w", err)
	}

	return &RunConcurrencyResult{
		Queued:                    queued,
		Cancelled:                 cancelled,
		NextConcurrencyStrategies: nextConcurrencyStrategies,
		FailedAdvisoryLock:        false,
	}, nil
}

func (c *ConcurrencyRepositoryImpl) UpdateConcurrencySlots(
	ctx context.Context,
	tenantId uuid.UUID,
	strategyId int64,
	filledSlots []TaskIdInsertedAtRetryCount,
	cancelledSlots []CancelledSlotInput,
) (*RunConcurrencyResult, error) {
	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, c.pool, c.l)

	if err != nil {
		return nil, fmt.Errorf("failed to prepare transaction: %w", err)
	}

	defer rollback()

	res, err := c.UpdateConcurrencySlotsTx(ctx, tx, tenantId, strategyId, filledSlots, cancelledSlots)

	if err != nil {
		return nil, err
	}

	if err := commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return res, nil
}
