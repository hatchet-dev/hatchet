//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSingleTaskWorkflowConcurrencyPersistsAsTaskLevel(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()
	repo := newWorkflowTestRepository(pool)

	maxRuns := int32(3)
	strategy := "GROUP_ROUND_ROBIN"
	taskMaxRuns := int32(1)
	taskStrategy := "CANCEL_IN_PROGRESS"
	desc := "single-task-workflow-concurrency"

	opts := &CreateWorkflowVersionOpts{
		Name:        "single-task-wf-concurrency",
		Description: &desc,
		Concurrency: []CreateConcurrencyOpts{
			{
				MaxRuns:       &maxRuns,
				LimitStrategy: &strategy,
				Expression:    "input.user_id",
			},
		},
		Tasks: []CreateStepOpts{
			{
				ReadableId: "only",
				Action:     "integration:only",
				Concurrency: []CreateConcurrencyOpts{
					{
						MaxRuns:       &taskMaxRuns,
						LimitStrategy: &taskStrategy,
						Expression:    "input.task_id",
					},
				},
			},
		},
	}

	wf, err := repo.PutWorkflowVersion(ctx, internalTenantId, opts)
	require.NoError(t, err)

	var workflowConcCount int
	err = pool.QueryRow(ctx, `
		SELECT count(*)
		FROM v1_workflow_concurrency
		WHERE workflow_version_id = $1
	`, wf.WorkflowVersion.ID).Scan(&workflowConcCount)
	require.NoError(t, err)
	assert.Equal(t, 0, workflowConcCount, "single-task workflow concurrency should not create a parent strategy")

	rows, err := pool.Query(ctx, `
		SELECT sc.expression, sc.max_concurrency, sc.strategy, sc.parent_strategy_id
		FROM v1_step_concurrency sc
		JOIN "Step" s ON s.id = sc.step_id
		WHERE sc.workflow_version_id = $1
		ORDER BY sc.id
	`, wf.WorkflowVersion.ID)
	require.NoError(t, err)
	defer rows.Close()

	type concRow struct {
		expression       string
		maxConcurrency   int32
		strategy         string
		parentStrategyID *int64
	}

	var got []concRow
	for rows.Next() {
		var row concRow
		require.NoError(t, rows.Scan(&row.expression, &row.maxConcurrency, &row.strategy, &row.parentStrategyID))
		got = append(got, row)
	}
	require.NoError(t, rows.Err())
	require.Len(t, got, 2)

	assert.Equal(t, "input.user_id", got[0].expression)
	assert.Equal(t, maxRuns, got[0].maxConcurrency)
	assert.Equal(t, strategy, got[0].strategy)
	assert.Nil(t, got[0].parentStrategyID, "merged workflow concurrency must not have a parent strategy")

	assert.Equal(t, "input.task_id", got[1].expression)
	assert.Equal(t, taskMaxRuns, got[1].maxConcurrency)
	assert.Equal(t, taskStrategy, got[1].strategy)
	assert.Nil(t, got[1].parentStrategyID)
}

func TestMultiTaskWorkflowConcurrencyKeepsParentStrategy(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()
	repo := newWorkflowTestRepository(pool)

	maxRuns := int32(2)
	strategy := "GROUP_ROUND_ROBIN"
	desc := "multi-task-workflow-concurrency"

	opts := &CreateWorkflowVersionOpts{
		Name:        "multi-task-wf-concurrency",
		Description: &desc,
		Concurrency: []CreateConcurrencyOpts{
			{
				MaxRuns:       &maxRuns,
				LimitStrategy: &strategy,
				Expression:    "input.user_id",
			},
		},
		Tasks: []CreateStepOpts{
			{ReadableId: "first", Action: "integration:first"},
			{ReadableId: "second", Action: "integration:second", Parents: []string{"first"}},
		},
	}

	wf, err := repo.PutWorkflowVersion(ctx, internalTenantId, opts)
	require.NoError(t, err)

	var workflowConcCount int
	err = pool.QueryRow(ctx, `
		SELECT count(*)
		FROM v1_workflow_concurrency
		WHERE workflow_version_id = $1
	`, wf.WorkflowVersion.ID).Scan(&workflowConcCount)
	require.NoError(t, err)
	assert.Equal(t, 1, workflowConcCount, "multi-task workflow concurrency should keep a parent strategy")

	var parentCount int
	err = pool.QueryRow(ctx, `
		SELECT count(*)
		FROM v1_step_concurrency
		WHERE workflow_version_id = $1
		  AND parent_strategy_id IS NOT NULL
	`, wf.WorkflowVersion.ID).Scan(&parentCount)
	require.NoError(t, err)
	assert.Equal(t, 2, parentCount, "each DEFAULT step should get a child strategy")
}
