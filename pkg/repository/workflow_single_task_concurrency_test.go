//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// Workflow-level concurrency must hold its slot for the whole run, including while the
// sole task waits out a retry backoff. A task-level slot is released during backoff, so
// with the DAG operator a single-task workflow must run through the orchestrator instead
// of having its workflow concurrency merged onto the task.
func TestPutWorkflowVersionSingleTaskWithWorkflowConcurrencyUsesDagOperator(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()
	repo := newWorkflowTestRepository(pool)

	_, err := repo.queries.UpsertTenantEntitlement(ctx, pool, sqlcv1.UpsertTenantEntitlementParams{
		Tenantid:    internalTenantId,
		Dagoperator: true,
	})
	require.NoError(t, err)

	maxRuns := int32(1)
	strategy := "GROUP_ROUND_ROBIN"
	description := "single task with workflow-level concurrency"
	opts := &CreateWorkflowVersionOpts{
		Name:        "single-task-workflow-concurrency",
		Description: &description,
		Concurrency: []CreateConcurrencyOpts{
			{Expression: "'shared'", MaxRuns: &maxRuns, LimitStrategy: &strategy},
		},
		Tasks: []CreateStepOpts{
			{ReadableId: "step1", Action: "integration:step1"},
		},
	}

	workflowVersion, err := repo.PutWorkflowVersion(ctx, internalTenantId, opts)
	require.NoError(t, err)
	require.True(t, workflowVersion.WorkflowVersion.IsUsingDagOperator)

	steps, err := repo.queries.ListStepsByWorkflowVersionIds(ctx, pool, sqlcv1.ListStepsByWorkflowVersionIdsParams{
		Tenantid: internalTenantId,
		Ids:      []uuid.UUID{workflowVersion.WorkflowVersion.ID},
	})
	require.NoError(t, err)
	require.Len(t, steps, 2)

	for _, step := range steps {
		strategies, err := repo.queries.ListConcurrencyStrategiesByStepId(ctx, pool, sqlcv1.ListConcurrencyStrategiesByStepIdParams{
			Tenantid: internalTenantId,
			Stepids:  []uuid.UUID{step.ID},
		})
		require.NoError(t, err)

		if step.IsDagOrchestrator {
			require.Len(t, strategies, 1, "workflow concurrency should live on the orchestrator")
		} else {
			require.Empty(t, strategies, "workflow concurrency should not be merged onto the task")
		}
	}
}
