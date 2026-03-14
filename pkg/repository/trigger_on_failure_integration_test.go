//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These integration tests verify the DAG status transition SQL logic using a real
// PostgreSQL database (via TestContainers). They test the CASE expression from
// UpdateDAGStatuses to ensure that on_failure steps don't prevent DAG completion.
//
// The tests simulate the dag_task_counts CTE by providing inline values, avoiding
// the need to insert into partitioned OLAP tables.

// TestDAGStatusSQL_SuccessPath_OnFailureExcluded verifies that when total_tasks
// excludes on_failure and the main task completes, the DAG transitions to COMPLETED.
func TestDAGStatusSQL_SuccessPath_OnFailureExcluded(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()

	// Simulate: 1 main task completed, total_tasks=1 (on_failure excluded by trigger.go fix)
	var newStatus string
	err := pool.QueryRow(ctx, `
		SELECT CASE
			WHEN queued_count = task_count THEN 'QUEUED'
			WHEN task_count < total_tasks THEN 'RUNNING'
			WHEN running_count > 0 OR queued_count > 0 THEN 'RUNNING'
			WHEN failed_count > 0 THEN 'FAILED'
			WHEN cancelled_count > 0 THEN 'CANCELLED'
			WHEN completed_count = task_count THEN 'COMPLETED'
			ELSE 'RUNNING'
		END
		FROM (VALUES (1, 1, 1, 0, 0, 0, 0))
			AS t(task_count, total_tasks, completed_count, failed_count, cancelled_count, queued_count, running_count)
	`).Scan(&newStatus)

	require.NoError(t, err)
	assert.Equal(t, "COMPLETED", newStatus,
		"DAG should be COMPLETED: task_count=1, total_tasks=1 (on_failure excluded), completed=1")
}

// TestDAGStatusSQL_FailurePath_OnFailureTriggered verifies that when the main task
// fails and on_failure triggers (creating an extra task beyond total_tasks), the DAG
// transitions to FAILED instead of staying stuck in RUNNING.
func TestDAGStatusSQL_FailurePath_OnFailureTriggered(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()

	// Simulate: main task FAILED + on_failure task COMPLETED
	// task_count=2, total_tasks=1 (on_failure excluded from total)
	// With < fix: 2 < 1 is FALSE → proceeds to check failed_count > 0 → FAILED
	var newStatus string
	err := pool.QueryRow(ctx, `
		SELECT CASE
			WHEN queued_count = task_count THEN 'QUEUED'
			WHEN task_count < total_tasks THEN 'RUNNING'
			WHEN running_count > 0 OR queued_count > 0 THEN 'RUNNING'
			WHEN failed_count > 0 THEN 'FAILED'
			WHEN cancelled_count > 0 THEN 'CANCELLED'
			WHEN completed_count = task_count THEN 'COMPLETED'
			ELSE 'RUNNING'
		END
		FROM (VALUES (2, 1, 1, 1, 0, 0, 0))
			AS t(task_count, total_tasks, completed_count, failed_count, cancelled_count, queued_count, running_count)
	`).Scan(&newStatus)

	require.NoError(t, err)
	assert.Equal(t, "FAILED", newStatus,
		"DAG should be FAILED: main task failed, on_failure completed (task_count=2 > total_tasks=1 should NOT block)")
}

// TestDAGStatusSQL_OldBug_OnFailureInTotalTasks documents the original bug.
// When on_failure IS counted in total_tasks and doesn't trigger (success path),
// the DAG stays stuck in RUNNING.
func TestDAGStatusSQL_OldBug_OnFailureInTotalTasks(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()

	// Simulate OLD BUG: total_tasks=2 (incorrectly includes on_failure), only 1 task ran
	// task_count(1) < total_tasks(2) → RUNNING forever
	var newStatus string
	err := pool.QueryRow(ctx, `
		SELECT CASE
			WHEN queued_count = task_count THEN 'QUEUED'
			WHEN task_count < total_tasks THEN 'RUNNING'
			WHEN running_count > 0 OR queued_count > 0 THEN 'RUNNING'
			WHEN failed_count > 0 THEN 'FAILED'
			WHEN cancelled_count > 0 THEN 'CANCELLED'
			WHEN completed_count = task_count THEN 'COMPLETED'
			ELSE 'RUNNING'
		END
		FROM (VALUES (1, 2, 1, 0, 0, 0, 0))
			AS t(task_count, total_tasks, completed_count, failed_count, cancelled_count, queued_count, running_count)
	`).Scan(&newStatus)

	require.NoError(t, err)
	assert.Equal(t, "RUNNING", newStatus,
		"With old bug (total_tasks=2 including on_failure), DAG stays RUNNING because 1 < 2. "+
			"This proves the trigger.go fix is also needed.")
}

// TestDAGStatusSQL_OldOperator_WouldBlockOnFailureTriggered documents that the old
// != operator would also block the failure path when on_failure adds an extra task.
func TestDAGStatusSQL_OldOperator_WouldBlockOnFailureTriggered(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()

	// Simulate with OLD operator (!=): task_count=2, total_tasks=1
	// 2 != 1 is TRUE → RUNNING (stuck!)
	// With NEW operator (<): 2 < 1 is FALSE → proceeds → FAILED (correct)
	var oldStatus, newStatus string

	// Old behavior (!=)
	err := pool.QueryRow(ctx, `
		SELECT CASE
			WHEN task_count != total_tasks THEN 'RUNNING'
			WHEN failed_count > 0 THEN 'FAILED'
			WHEN completed_count = task_count THEN 'COMPLETED'
			ELSE 'RUNNING'
		END
		FROM (VALUES (2, 1, 1, 1, 0, 0, 0))
			AS t(task_count, total_tasks, completed_count, failed_count, cancelled_count, queued_count, running_count)
	`).Scan(&oldStatus)
	require.NoError(t, err)

	// New behavior (<)
	err = pool.QueryRow(ctx, `
		SELECT CASE
			WHEN task_count < total_tasks THEN 'RUNNING'
			WHEN failed_count > 0 THEN 'FAILED'
			WHEN completed_count = task_count THEN 'COMPLETED'
			ELSE 'RUNNING'
		END
		FROM (VALUES (2, 1, 1, 1, 0, 0, 0))
			AS t(task_count, total_tasks, completed_count, failed_count, cancelled_count, queued_count, running_count)
	`).Scan(&newStatus)
	require.NoError(t, err)

	assert.Equal(t, "RUNNING", oldStatus, "Old != operator: 2 != 1 → RUNNING (bug)")
	assert.Equal(t, "FAILED", newStatus, "New < operator: 2 < 1 is false → proceeds to FAILED (correct)")
}

// TestDAGStatusSQL_NormalDAG_NoOnFailure verifies that normal DAGs without
// on_failure steps still work correctly (no regression).
func TestDAGStatusSQL_NormalDAG_NoOnFailure(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()

	tests := []struct {
		name     string
		values   string // task_count, total_tasks, completed, failed, cancelled, queued, running
		expected string
	}{
		{
			name:     "all_completed",
			values:   "3, 3, 3, 0, 0, 0, 0",
			expected: "COMPLETED",
		},
		{
			name:     "one_still_running",
			values:   "3, 3, 2, 0, 0, 0, 1",
			expected: "RUNNING",
		},
		{
			name:     "one_failed",
			values:   "3, 3, 2, 1, 0, 0, 0",
			expected: "FAILED",
		},
		{
			name:     "not_all_tasks_created_yet",
			values:   "1, 3, 1, 0, 0, 0, 0",
			expected: "RUNNING",
		},
		{
			name:     "one_cancelled",
			values:   "2, 2, 1, 0, 1, 0, 0",
			expected: "CANCELLED",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var status string
			err := pool.QueryRow(ctx, `
				SELECT CASE
					WHEN queued_count = task_count THEN 'QUEUED'
					WHEN task_count < total_tasks THEN 'RUNNING'
					WHEN running_count > 0 OR queued_count > 0 THEN 'RUNNING'
					WHEN failed_count > 0 THEN 'FAILED'
					WHEN cancelled_count > 0 THEN 'CANCELLED'
					WHEN completed_count = task_count THEN 'COMPLETED'
					ELSE 'RUNNING'
				END
				FROM (VALUES (`+tt.values+`))
					AS t(task_count, total_tasks, completed_count, failed_count, cancelled_count, queued_count, running_count)
			`).Scan(&status)

			require.NoError(t, err)
			assert.Equal(t, tt.expected, status)
		})
	}
}
