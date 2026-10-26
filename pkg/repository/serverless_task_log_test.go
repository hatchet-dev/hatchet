//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServerlessTaskLogs(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()
	repo := newServerlessTestRepository(t, pool)

	tenantId := uuid.New()
	externalId := uuid.New()
	stepId, workflowId := uuid.New(), uuid.New()

	var taskId int64
	var taskInsertedAt time.Time

	err := pool.QueryRow(ctx, `
		INSERT INTO v1_task (
			tenant_id, queue, action_id, step_id, step_readable_id, workflow_id, workflow_version_id,
			workflow_run_id, schedule_timeout, step_timeout, sticky, external_id, display_name, input,
			step_index
		)
		VALUES ($1, 'default', 'wf:task', $2, 'task', $3, $4, $5, '5m', '30s', 'NONE', $6, 'task', '{}', 0)
		RETURNING id, inserted_at
	`, tenantId, stepId, workflowId, uuid.New(), uuid.New(), externalId).Scan(&taskId, &taskInsertedAt)
	require.NoError(t, err)

	err = repo.TaskLogs().Insert(ctx, tenantId, []ServerlessTaskLogOpts{
		{
			TaskExternalId: externalId,
			RetryCount:     1,
			Message:        "Delivered to serverless endpoint orders",
			Metadata:       []byte(`{"hatchet__serverless_endpoint_id":"x"}`),
		},
		// a task that does not exist is skipped, not an error
		{TaskExternalId: uuid.New(), Message: "missing"},
	})
	require.NoError(t, err)

	rows, err := pool.Query(ctx, `
		SELECT message, retry_count, level::text, metadata->>'hatchet__serverless_endpoint_id', workflow_id, step_id
		FROM v1_log_line
		WHERE tenant_id = $1 AND task_id = $2 AND task_inserted_at = $3
	`, tenantId, taskId, taskInsertedAt)
	require.NoError(t, err)
	defer rows.Close()

	count := 0

	for rows.Next() {
		var message, level, endpointId string
		var retryCount int32
		var gotWorkflow, gotStep uuid.UUID

		require.NoError(t, rows.Scan(&message, &retryCount, &level, &endpointId, &gotWorkflow, &gotStep))

		assert.Equal(t, "Delivered to serverless endpoint orders", message)
		assert.Equal(t, int32(1), retryCount)
		assert.Equal(t, "INFO", level)
		assert.Equal(t, "x", endpointId)
		assert.Equal(t, workflowId, gotWorkflow)
		assert.Equal(t, stepId, gotStep)

		count++
	}

	require.NoError(t, rows.Err())
	assert.Equal(t, 1, count)

	// nothing to write is a no-op
	require.NoError(t, repo.TaskLogs().Insert(ctx, tenantId, nil))
}
