//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

const taskStatsGroupRoundRobin = "GROUP_ROUND_ROBIN"

type taskStatsTestEnv struct {
	t        *testing.T
	ctx      context.Context
	pool     *pgxpool.Pool
	repo     *TaskRepositoryImpl
	tenantID uuid.UUID
	workerID uuid.UUID
}

type taskStatsStrategySeed struct {
	expression string
	key        string
	parent     bool
	filled     bool
}

func newTaskStatsTestEnv(t *testing.T, pool *pgxpool.Pool) *taskStatsTestEnv {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	t.Cleanup(cancel)

	env := &taskStatsTestEnv{
		t:        t,
		ctx:      ctx,
		pool:     pool,
		repo:     createTaskRepository(pool),
		tenantID: uuid.New(),
		workerID: uuid.New(),
	}

	_, err := pool.Exec(ctx, `
		INSERT INTO "Tenant" ("id", "name", "slug", "createdAt", "updatedAt")
		VALUES ($1, 'task-stats-tenant', $2, NOW(), NOW())
	`, env.tenantID, "task-stats-"+env.tenantID.String())
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `
		INSERT INTO "Worker" ("id", "tenantId", "name", "lastHeartbeatAt", "createdAt", "updatedAt")
		VALUES ($1, $2, 'task-stats-worker', NOW(), NOW(), NOW())
	`, env.workerID, env.tenantID)
	require.NoError(t, err)

	return env
}

func (e *taskStatsTestEnv) seedTask(stepReadableID string, running bool, seeds ...taskStatsStrategySeed) {
	e.t.Helper()

	workflowID := uuid.New()
	workflowVersionID := uuid.New()
	workflowRunID := uuid.New()
	stepID := uuid.New()
	externalID := uuid.New()

	strategyIDs := make([]int64, len(seeds))
	parentStrategyIDs := make([]any, len(seeds))
	keys := make([]string, len(seeds))

	for i, seed := range seeds {
		if seed.parent {
			var parentStrategyID int64
			err := e.pool.QueryRow(e.ctx, `
				INSERT INTO v1_workflow_concurrency (
					workflow_id,
					workflow_version_id,
					strategy,
					expression,
					tenant_id,
					max_concurrency
				)
				VALUES ($1, $2, $3, $4, $5, 5)
				RETURNING id
			`, workflowID, workflowVersionID, taskStatsGroupRoundRobin, seed.expression, e.tenantID).Scan(&parentStrategyID)
			require.NoError(e.t, err)
			parentStrategyIDs[i] = parentStrategyID
		}

		err := e.pool.QueryRow(e.ctx, `
			INSERT INTO v1_step_concurrency (
				parent_strategy_id,
				workflow_id,
				workflow_version_id,
				step_id,
				strategy,
				expression,
				tenant_id,
				max_concurrency
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, 5)
			RETURNING id
		`, parentStrategyIDs[i], workflowID, workflowVersionID, stepID, taskStatsGroupRoundRobin, seed.expression, e.tenantID).Scan(&strategyIDs[i])
		require.NoError(e.t, err)

		if parentStrategyIDs[i] != nil {
			_, err = e.pool.Exec(e.ctx, `
				UPDATE v1_workflow_concurrency
				SET child_strategy_ids = ARRAY[$1]::bigint[]
				WHERE workflow_id = $2
					AND workflow_version_id = $3
					AND id = $4
			`, strategyIDs[i], workflowID, workflowVersionID, parentStrategyIDs[i])
			require.NoError(e.t, err)
		}

		keys[i] = seed.key
	}

	var taskID int64
	var taskInsertedAt time.Time
	err := e.pool.QueryRow(e.ctx, `
		INSERT INTO v1_task (
			tenant_id,
			queue,
			action_id,
			step_id,
			step_readable_id,
			workflow_id,
			workflow_version_id,
			workflow_run_id,
			schedule_timeout,
			step_timeout,
			sticky,
			external_id,
			display_name,
			input,
			step_index
		)
		VALUES ($1, 'default', 'test:run', $2, $3, $4, $5, $6, '5m', '30s', 'NONE', $7, 'task-stats-task', '{}', 0)
		RETURNING id, inserted_at
	`, e.tenantID, stepID, stepReadableID, workflowID, workflowVersionID, workflowRunID, externalID).Scan(&taskID, &taskInsertedAt)
	require.NoError(e.t, err)

	if len(strategyIDs) > 0 {
		_, err = e.pool.Exec(e.ctx, `
			UPDATE v1_task
			SET concurrency_strategy_ids = $1,
				concurrency_keys = $2
			WHERE id = $3 AND inserted_at = $4
		`, strategyIDs, keys, taskID, taskInsertedAt)
		require.NoError(e.t, err)
	}

	if running || len(strategyIDs) > 0 {
		_, err = e.pool.Exec(e.ctx, `
			DELETE FROM v1_queue_item
			WHERE task_id = $1 AND task_inserted_at = $2 AND retry_count = 0
		`, taskID, taskInsertedAt)
		require.NoError(e.t, err)
	}

	for i, seed := range seeds {
		_, err = e.pool.Exec(e.ctx, `
			INSERT INTO v1_concurrency_slot (
				task_id,
				task_inserted_at,
				task_retry_count,
				external_id,
				tenant_id,
				workflow_id,
				workflow_version_id,
				workflow_run_id,
				strategy_id,
				parent_strategy_id,
				priority,
				key,
				is_filled,
				queue_to_notify,
				schedule_timeout_at
			)
			VALUES ($1, $2, 0, $3, $4, $5, $6, $7, $8, $9, 1, $10, $11, 'default', NOW() + INTERVAL '5 minutes')
		`, taskID, taskInsertedAt, externalID, e.tenantID, workflowID, workflowVersionID, workflowRunID, strategyIDs[i], parentStrategyIDs[i], seed.key, seed.filled)
		require.NoError(e.t, err)
	}

	if running {
		_, err = e.pool.Exec(e.ctx, `
			INSERT INTO v1_task_runtime (
				task_id,
				task_inserted_at,
				retry_count,
				worker_id,
				tenant_id,
				timeout_at
			)
			VALUES ($1, $2, 0, $3, $4, NOW() + INTERVAL '30 seconds')
		`, taskID, taskInsertedAt, e.workerID, e.tenantID)
		require.NoError(e.t, err)
	}
}

func (e *taskStatsTestEnv) taskStats(stepReadableID string) TaskStat {
	e.t.Helper()

	stats, err := e.repo.GetTaskStats(e.ctx, e.tenantID)
	require.NoError(e.t, err)
	return stats[stepReadableID]
}

func TestGetTaskStats(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	t.Run("duplicate strategies count one running attempt", func(t *testing.T) {
		env := newTaskStatsTestEnv(t, pool)
		env.seedTask("duplicate-strategies", true,
			taskStatsStrategySeed{expression: "input.my_id", key: "shared-key", parent: true, filled: true},
			taskStatsStrategySeed{expression: "input.my_id", key: "shared-key", filled: true},
		)

		running := env.taskStats("duplicate-strategies").Running
		require.NotNil(t, running)
		require.Equal(t, int64(1), running.Total)
		require.Equal(t, []ConcurrencyStat{{
			Expression: "input.my_id",
			Type:       taskStatsGroupRoundRobin,
			Keys:       map[string]int64{"shared-key": 1},
		}}, running.Concurrency)
	})

	t.Run("distinct strategies do not inflate running total", func(t *testing.T) {
		env := newTaskStatsTestEnv(t, pool)
		env.seedTask("distinct-strategies", true,
			taskStatsStrategySeed{expression: "input.key_a", key: "key-a", filled: true},
			taskStatsStrategySeed{expression: "input.key_b", key: "key-b", filled: true},
		)

		running := env.taskStats("distinct-strategies").Running
		require.NotNil(t, running)
		require.Equal(t, int64(1), running.Total)
		require.ElementsMatch(t, []ConcurrencyStat{
			{
				Expression: "input.key_a",
				Type:       taskStatsGroupRoundRobin,
				Keys:       map[string]int64{"key-a": 1},
			},
			{
				Expression: "input.key_b",
				Type:       taskStatsGroupRoundRobin,
				Keys:       map[string]int64{"key-b": 1},
			},
		}, running.Concurrency)
	})

	t.Run("running task without concurrency", func(t *testing.T) {
		env := newTaskStatsTestEnv(t, pool)
		env.seedTask("no-concurrency", true)

		running := env.taskStats("no-concurrency").Running
		require.NotNil(t, running)
		require.Equal(t, int64(1), running.Total)
		require.Empty(t, running.Concurrency)
	})

	t.Run("queued concurrency aggregation is unchanged", func(t *testing.T) {
		env := newTaskStatsTestEnv(t, pool)
		env.seedTask("queued-concurrency", false,
			taskStatsStrategySeed{expression: "input.my_id", key: "queued-key"},
		)

		stat := env.taskStats("queued-concurrency")
		require.Nil(t, stat.Running)
		require.NotNil(t, stat.Queued)
		require.Equal(t, int64(1), stat.Queued.Total)
		require.Equal(t, []ConcurrencyStat{{
			Expression: "input.my_id",
			Type:       taskStatsGroupRoundRobin,
			Keys:       map[string]int64{"queued-key": 1},
		}}, stat.Queued.Concurrency)
	})
}
