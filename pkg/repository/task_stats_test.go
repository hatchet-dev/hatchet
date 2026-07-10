//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/repository/cache"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

type taskStatsEnv struct {
	t               *testing.T
	ctx             context.Context
	repo            *TaskRepositoryImpl
	wfRepo          *workflowRepository
	queries         *sqlcv1.Queries
	concurrencyRepo ConcurrencyRepository
	tenantID        uuid.UUID
	workerID        uuid.UUID
}

type taskStatsWorkflow struct {
	workflowID        uuid.UUID
	workflowVersionID uuid.UUID
	stepID            uuid.UUID
	strategies        []*sqlcv1.V1StepConcurrency
}

type taskStatsTask struct {
	id            int64
	insertedAt    pgtype.Timestamptz
	retryCount    int32
	workflowRunID uuid.UUID
}

func newTaskStatsEnv(t *testing.T, pool *pgxpool.Pool) *taskStatsEnv {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	t.Cleanup(cancel)

	tenantID := uuid.New()
	workerID := uuid.New()
	suffix := tenantID.String()

	_, err := pool.Exec(ctx, `
		INSERT INTO "Tenant" ("id", "name", "slug", "createdAt", "updatedAt")
		VALUES ($1, 'task-stats-tenant', $2, NOW(), NOW())
	`, tenantID, "task-stats-"+suffix)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `
		INSERT INTO "Worker" ("id", "tenantId", "name", "lastHeartbeatAt", "createdAt", "updatedAt")
		VALUES ($1, $2, 'task-stats-worker', NOW(), NOW(), NOW())
	`, workerID, tenantID)
	require.NoError(t, err)

	repo := createTaskRepository(pool)
	repo.queueCache = cache.New(5 * time.Minute)
	t.Cleanup(repo.queueCache.Stop)

	wfRepo := newWorkflowTestRepository(pool)
	t.Cleanup(wfRepo.queueCache.Stop)

	return &taskStatsEnv{
		t:               t,
		ctx:             ctx,
		repo:            repo,
		wfRepo:          wfRepo,
		queries:         sqlcv1.New(),
		concurrencyRepo: newConcurrencyRepository(repo.sharedRepository),
		tenantID:        tenantID,
		workerID:        workerID,
	}
}

func (e *taskStatsEnv) registerWorkflow(
	name string,
	workflowConcurrency []CreateConcurrencyOpts,
	stepConcurrency []CreateConcurrencyOpts,
) taskStatsWorkflow {
	e.t.Helper()

	desc := name
	version, err := e.wfRepo.PutWorkflowVersion(e.ctx, e.tenantID, &CreateWorkflowVersionOpts{
		Name:        name,
		Description: &desc,
		Concurrency: workflowConcurrency,
		Tasks: []CreateStepOpts{
			{
				ReadableId:  "my-task",
				Action:      "test:run",
				Concurrency: stepConcurrency,
			},
		},
	})
	require.NoError(e.t, err)

	workflowID := version.WorkflowVersion.WorkflowId
	workflowVersionID := version.WorkflowVersion.ID

	var stepID uuid.UUID
	err = e.repo.pool.QueryRow(e.ctx, `
		SELECT s."id"
		FROM "Step" s
		JOIN "Job" j ON s."jobId" = j."id"
		WHERE j."workflowVersionId" = $1 AND s."readableId" = 'my-task'
	`, workflowVersionID).Scan(&stepID)
	require.NoError(e.t, err)

	strategies, err := e.queries.ListActiveConcurrencyStrategies(e.ctx, e.repo.pool, e.tenantID)
	require.NoError(e.t, err)

	filtered := make([]*sqlcv1.V1StepConcurrency, 0, len(strategies))
	for _, strategy := range strategies {
		if strategy.WorkflowID == workflowID && strategy.WorkflowVersionID == workflowVersionID && strategy.StepID == stepID {
			filtered = append(filtered, strategy)
		}
	}
	sortConcurrencyStrategies(filtered)

	return taskStatsWorkflow{
		workflowID:        workflowID,
		workflowVersionID: workflowVersionID,
		stepID:            stepID,
		strategies:        filtered,
	}
}

func (e *taskStatsEnv) createTask(workflow taskStatsWorkflow, keysByExpression map[string]string) taskStatsTask {
	e.t.Helper()

	strategyIDs := make([]int64, len(workflow.strategies))
	parentStrategyIDs := make([]pgtype.Int8, len(workflow.strategies))
	concurrencyKeys := make([]string, len(workflow.strategies))

	for i, strategy := range workflow.strategies {
		key, ok := keysByExpression[strategy.Expression]
		require.True(e.t, ok, "missing concurrency key for expression %q", strategy.Expression)

		strategyIDs[i] = strategy.ID
		parentStrategyIDs[i] = strategy.ParentStrategyID
		concurrencyKeys[i] = key
	}

	workflowRunID := uuid.New()
	tasks, err := e.queries.CreateTasks(e.ctx, e.repo.pool, sqlcv1.CreateTasksParams{
		Tenantids:                    []uuid.UUID{e.tenantID},
		Queues:                       []string{"default"},
		Actionids:                    []string{"test:run"},
		Stepids:                      []uuid.UUID{workflow.stepID},
		Stepreadableids:              []string{"my-task"},
		Workflowids:                  []uuid.UUID{workflow.workflowID},
		Scheduletimeouts:             []string{"5m"},
		Steptimeouts:                 []string{"30s"},
		Priorities:                   []int32{1},
		Stickies:                     []string{string(sqlcv1.V1StickyStrategyNONE)},
		Externalids:                  []uuid.UUID{uuid.New()},
		Displaynames:                 []string{"task-stats-task"},
		Inputs:                       [][]byte{[]byte(`{}`)},
		Retrycounts:                  []int32{0},
		Additionalmetadatas:          [][]byte{[]byte(`{}`)},
		InitialStates:                []string{string(sqlcv1.V1TaskInitialStateQUEUED)},
		Concurrencyparentstrategyids: [][]pgtype.Int8{parentStrategyIDs},
		ConcurrencyStrategyIds:       [][]int64{strategyIDs},
		ConcurrencyKeys:              [][]string{concurrencyKeys},
		WorkflowVersionIds:           []uuid.UUID{workflow.workflowVersionID},
		WorkflowRunIds:               []uuid.UUID{workflowRunID},
	})
	require.NoError(e.t, err)
	require.Len(e.t, tasks, 1)

	return taskStatsTask{
		id:            tasks[0].ID,
		insertedAt:    tasks[0].InsertedAt,
		retryCount:    tasks[0].RetryCount,
		workflowRunID: workflowRunID,
	}
}

func (e *taskStatsEnv) advanceConcurrency(workflow taskStatsWorkflow) {
	e.t.Helper()

	for _, strategy := range workflow.strategies {
		result, err := e.concurrencyRepo.RunConcurrencyStrategy(e.ctx, e.tenantID, strategy)
		require.NoError(e.t, err)
		require.NotNil(e.t, result)
		require.False(e.t, result.FailedAdvisoryLock)
		require.Empty(e.t, result.Cancelled)
	}
}

func (e *taskStatsEnv) assign(task taskStatsTask) {
	e.t.Helper()

	tx, err := e.repo.pool.Begin(e.ctx)
	require.NoError(e.t, err)
	defer func() { _ = tx.Rollback(e.ctx) }()

	var queueItemID int64
	err = tx.QueryRow(e.ctx, `
		SELECT id
		FROM v1_queue_item
		WHERE tenant_id = $1
			AND task_id = $2
			AND task_inserted_at = $3
			AND retry_count = $4
	`, e.tenantID, task.id, task.insertedAt, task.retryCount).Scan(&queueItemID)
	require.NoError(e.t, err)

	deletedQueueItems, err := e.queries.BulkQueueItems(e.ctx, tx, []int64{queueItemID})
	require.NoError(e.t, err)
	require.Equal(e.t, []int64{queueItemID}, deletedQueueItems)

	assigned, err := e.queries.UpdateTasksToAssigned(e.ctx, tx, sqlcv1.UpdateTasksToAssignedParams{
		Taskids:           []int64{task.id},
		Taskinsertedats:   []pgtype.Timestamptz{task.insertedAt},
		Mintaskinsertedat: task.insertedAt,
		Workerids:         []uuid.UUID{e.workerID},
		Tenantid:          e.tenantID,
	})
	require.NoError(e.t, err)
	require.Len(e.t, assigned, 1)

	require.NoError(e.t, tx.Commit(e.ctx))
}

func (e *taskStatsEnv) advanceAndAssign(task taskStatsTask, workflow taskStatsWorkflow) {
	e.t.Helper()

	e.advanceConcurrency(workflow)
	e.assign(task)
}

func (e *taskStatsEnv) retryAndAssign(task taskStatsTask, workflow taskStatsWorkflow) taskStatsTask {
	e.t.Helper()

	tx, err := e.repo.pool.Begin(e.ctx)
	require.NoError(e.t, err)
	defer func() { _ = tx.Rollback(e.ctx) }()

	retried, err := e.queries.FailTaskInternalFailure(e.ctx, tx, sqlcv1.FailTaskInternalFailureParams{
		Maxinternalretries: 3,
		Taskids:            []int64{task.id},
		Taskinsertedats:    []pgtype.Timestamptz{task.insertedAt},
		Taskretrycounts:    []int32{task.retryCount},
		Tenantid:           e.tenantID,
	})
	require.NoError(e.t, err)
	require.Len(e.t, retried, 1)

	released, err := e.repo.releaseTasks(e.ctx, tx, e.tenantID, []TaskIdInsertedAtRetryCount{{
		Id:         task.id,
		InsertedAt: task.insertedAt,
		RetryCount: task.retryCount,
	}})
	require.NoError(e.t, err)
	require.Len(e.t, released, 1)
	require.NoError(e.t, tx.Commit(e.ctx))

	next := taskStatsTask{
		id:            retried[0].ID,
		insertedAt:    retried[0].InsertedAt,
		retryCount:    retried[0].RetryCount,
		workflowRunID: task.workflowRunID,
	}
	e.advanceAndAssign(next, workflow)

	return next
}

func (e *taskStatsEnv) clearAssignedWorker(task taskStatsTask) {
	e.t.Helper()

	// Production release paths delete the runtime, so clear the assigned worker in place to
	// exercise the legacy NULL-worker state without reintroducing a queue item.
	tag, err := e.repo.pool.Exec(e.ctx, `
		UPDATE v1_task_runtime
		SET worker_id = NULL
		WHERE task_id = $1
			AND task_inserted_at = $2
			AND retry_count = $3
			AND tenant_id = $4
	`, task.id, task.insertedAt, task.retryCount, e.tenantID)
	require.NoError(e.t, err)
	require.Equal(e.t, int64(1), tag.RowsAffected())

	var queueItems int
	err = e.repo.pool.QueryRow(e.ctx, `
		SELECT COUNT(*)
		FROM v1_queue_item
		WHERE tenant_id = $1
			AND task_id = $2
			AND task_inserted_at = $3
			AND retry_count = $4
	`, e.tenantID, task.id, task.insertedAt, task.retryCount).Scan(&queueItems)
	require.NoError(e.t, err)
	require.Zero(e.t, queueItems)
}

func (e *taskStatsEnv) stats() map[string]TaskStat {
	e.t.Helper()

	stats, err := e.repo.GetTaskStats(e.ctx, e.tenantID)
	require.NoError(e.t, err)
	return stats
}

func concurrencyKeyCount(status *TaskStatusStat, expression, strategyType, key string) int64 {
	if status == nil {
		return 0
	}

	for _, entry := range status.Concurrency {
		if entry.Expression == expression && entry.Type == strategyType {
			return entry.Keys[key]
		}
	}

	return 0
}

func TestGetTaskStats(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	groupRoundRobin := "GROUP_ROUND_ROBIN"
	maxRuns := int32(5)
	concurrency := func(expression string) CreateConcurrencyOpts {
		return CreateConcurrencyOpts{
			MaxRuns:       &maxRuns,
			LimitStrategy: &groupRoundRobin,
			Expression:    expression,
		}
	}

	t.Run("running task without concurrency", func(t *testing.T) {
		env := newTaskStatsEnv(t, pool)
		workflow := env.registerWorkflow("no-concurrency", nil, nil)
		task := env.createTask(workflow, nil)
		env.advanceAndAssign(task, workflow)

		running := env.stats()["my-task"].Running
		require.NotNil(t, running)
		assert.Equal(t, int64(1), running.Total)
		assert.Empty(t, running.Concurrency)
	})

	t.Run("one filled strategy", func(t *testing.T) {
		env := newTaskStatsEnv(t, pool)
		workflow := env.registerWorkflow("one-strategy", nil, []CreateConcurrencyOpts{concurrency("input.my_id")})
		require.Len(t, workflow.strategies, 1)
		task := env.createTask(workflow, map[string]string{"input.my_id": "shared-key"})
		env.advanceAndAssign(task, workflow)

		running := env.stats()["my-task"].Running
		require.NotNil(t, running)
		assert.Equal(t, int64(1), running.Total)
		assert.Equal(t, int64(1), concurrencyKeyCount(running, "input.my_id", groupRoundRobin, "shared-key"))
	})

	t.Run("duplicate workflow-child and direct strategies collapse", func(t *testing.T) {
		env := newTaskStatsEnv(t, pool)
		duplicate := []CreateConcurrencyOpts{concurrency("input.my_id")}
		workflow := env.registerWorkflow("duplicate-strategies", duplicate, duplicate)
		require.Len(t, workflow.strategies, 2)
		require.True(t, workflow.strategies[0].ParentStrategyID.Valid)
		require.False(t, workflow.strategies[1].ParentStrategyID.Valid)
		task := env.createTask(workflow, map[string]string{"input.my_id": "shared-key"})
		env.advanceConcurrency(workflow)

		var filledParentSlots int
		err := pool.QueryRow(env.ctx, `
			SELECT COUNT(*)
			FROM v1_workflow_concurrency_slot
			WHERE tenant_id = $1
				AND workflow_version_id = $2
				AND workflow_run_id = $3
				AND strategy_id = $4
				AND is_filled = TRUE
		`, env.tenantID, workflow.workflowVersionID, task.workflowRunID, workflow.strategies[0].ParentStrategyID.Int64).Scan(&filledParentSlots)
		require.NoError(t, err)
		require.Equal(t, 1, filledParentSlots)

		var filledSlots int
		var filledStrategyIDs int
		err = pool.QueryRow(env.ctx, `
			SELECT COUNT(*), COUNT(DISTINCT strategy_id)
			FROM v1_concurrency_slot
			WHERE task_id = $1
				AND task_inserted_at = $2
				AND task_retry_count = $3
				AND strategy_id = ANY($4::bigint[])
				AND is_filled = TRUE
		`, task.id, task.insertedAt, task.retryCount, []int64{
			workflow.strategies[0].ID,
			workflow.strategies[1].ID,
		}).Scan(&filledSlots, &filledStrategyIDs)
		require.NoError(t, err)
		require.Equal(t, 2, filledSlots)
		require.Equal(t, 2, filledStrategyIDs)

		env.assign(task)

		running := env.stats()["my-task"].Running
		require.NotNil(t, running)
		assert.Equal(t, int64(1), running.Total)
		assert.Equal(t, int64(1), concurrencyKeyCount(running, "input.my_id", groupRoundRobin, "shared-key"))
	})

	t.Run("distinct strategies each include the attempt", func(t *testing.T) {
		env := newTaskStatsEnv(t, pool)
		workflow := env.registerWorkflow("distinct-strategies", nil, []CreateConcurrencyOpts{
			concurrency("input.key_a"),
			concurrency("input.key_b"),
		})
		require.Len(t, workflow.strategies, 2)
		task := env.createTask(workflow, map[string]string{
			"input.key_a": "key-a",
			"input.key_b": "key-b",
		})
		env.advanceAndAssign(task, workflow)

		running := env.stats()["my-task"].Running
		require.NotNil(t, running)
		assert.Equal(t, int64(1), running.Total)
		assert.Equal(t, int64(1), concurrencyKeyCount(running, "input.key_a", groupRoundRobin, "key-a"))
		assert.Equal(t, int64(1), concurrencyKeyCount(running, "input.key_b", groupRoundRobin, "key-b"))
	})

	t.Run("same key counts distinct attempts", func(t *testing.T) {
		env := newTaskStatsEnv(t, pool)
		workflow := env.registerWorkflow("same-key", nil, []CreateConcurrencyOpts{concurrency("input.my_id")})

		for range 3 {
			task := env.createTask(workflow, map[string]string{"input.my_id": "shared-key"})
			env.advanceAndAssign(task, workflow)
		}

		running := env.stats()["my-task"].Running
		require.NotNil(t, running)
		assert.Equal(t, int64(3), running.Total)
		assert.Equal(t, int64(3), concurrencyKeyCount(running, "input.my_id", groupRoundRobin, "shared-key"))
	})

	t.Run("canonical retry transition keeps current attempt only", func(t *testing.T) {
		env := newTaskStatsEnv(t, pool)
		workflow := env.registerWorkflow("retry", nil, nil)
		task := env.createTask(workflow, nil)
		env.advanceAndAssign(task, workflow)

		retried := env.retryAndAssign(task, workflow)
		require.Equal(t, task.id, retried.id)
		require.Equal(t, int32(1), retried.retryCount)

		running := env.stats()["my-task"].Running
		require.NotNil(t, running)
		assert.Equal(t, int64(1), running.Total)
		assert.Nil(t, running.OldestExcludingRetries)
	})

	t.Run("assigned runtime with cleared worker is not running", func(t *testing.T) {
		env := newTaskStatsEnv(t, pool)
		workflow := env.registerWorkflow("null-worker", nil, nil)
		task := env.createTask(workflow, nil)
		env.advanceAndAssign(task, workflow)
		env.clearAssignedWorker(task)

		assert.Nil(t, env.stats()["my-task"].Running)
	})

	t.Run("missing runtime is not running", func(t *testing.T) {
		env := newTaskStatsEnv(t, pool)
		workflow := env.registerWorkflow("missing-runtime", nil, nil)
		env.createTask(workflow, nil)

		assert.Nil(t, env.stats()["my-task"].Running)
	})

	t.Run("unfilled is queued and filled assignment is running", func(t *testing.T) {
		env := newTaskStatsEnv(t, pool)
		workflow := env.registerWorkflow("slot-lifecycle", nil, []CreateConcurrencyOpts{concurrency("input.my_id")})
		task := env.createTask(workflow, map[string]string{"input.my_id": "slot-key"})

		queuedStats := env.stats()["my-task"]
		require.NotNil(t, queuedStats.Queued)
		assert.Equal(t, int64(1), queuedStats.Queued.Total)
		assert.Equal(t, int64(1), concurrencyKeyCount(queuedStats.Queued, "input.my_id", groupRoundRobin, "slot-key"))
		assert.Nil(t, queuedStats.Running)

		env.advanceAndAssign(task, workflow)

		runningStats := env.stats()["my-task"]
		require.NotNil(t, runningStats.Running)
		assert.Equal(t, int64(1), runningStats.Running.Total)
		assert.Equal(t, int64(1), concurrencyKeyCount(runningStats.Running, "input.my_id", groupRoundRobin, "slot-key"))
	})

	t.Run("oldest comes from assigned attempts", func(t *testing.T) {
		env := newTaskStatsEnv(t, pool)
		workflow := env.registerWorkflow("oldest", nil, nil)

		older := env.createTask(workflow, nil)
		env.advanceAndAssign(older, workflow)
		time.Sleep(2 * time.Millisecond)
		newer := env.createTask(workflow, nil)
		env.advanceAndAssign(newer, workflow)
		require.True(t, older.insertedAt.Time.Before(newer.insertedAt.Time))

		running := env.stats()["my-task"].Running
		require.NotNil(t, running)
		require.NotNil(t, running.Oldest)
		assert.True(t, running.Oldest.Equal(older.insertedAt.Time))
		require.NotNil(t, running.OldestExcludingRetries)
		assert.True(t, running.OldestExcludingRetries.Equal(older.insertedAt.Time))
	})

	t.Run("queued concurrency aggregation remains unchanged", func(t *testing.T) {
		env := newTaskStatsEnv(t, pool)
		workflow := env.registerWorkflow("queued-concurrency", nil, []CreateConcurrencyOpts{concurrency("input.my_id")})
		env.createTask(workflow, map[string]string{"input.my_id": "queued-key"})

		queued := env.stats()["my-task"].Queued
		require.NotNil(t, queued)
		assert.Equal(t, int64(1), queued.Total)
		assert.Equal(t, int64(1), concurrencyKeyCount(queued, "input.my_id", groupRoundRobin, "queued-key"))
	})
}
