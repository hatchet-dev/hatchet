//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// One DAG run with two tasks and one standalone task run, so a filter can be checked for both
// run kinds and for the DAG-to-task expansion of the task-only list.
func TestListRunsAndTasks_ConcurrencySlotExternalIdsFilter(t *testing.T) {
	ctx, _, repo := setupOLAPRepository(t)
	f := seedDAGWithFinishedChildren(t, ctx, repo, 2)

	standaloneExternalId := uuid.New()

	_, locksNotAcquired, err := repo.CreateTasks(ctx, f.tenantId, []*V1TaskWithPayload{{
		V1Task: &sqlcv1.V1Task{
			ID:                 100,
			InsertedAt:         f.insertedAt,
			TenantID:           f.tenantId,
			Queue:              "default",
			ActionID:           "test:standalone",
			StepID:             uuid.New(),
			WorkflowID:         uuid.New(),
			WorkflowVersionID:  uuid.New(),
			WorkflowRunID:      standaloneExternalId,
			ScheduleTimeout:    "5m",
			Sticky:             sqlcv1.V1StickyStrategyNONE,
			ExternalID:         standaloneExternalId,
			DisplayName:        "standalone",
			Input:              []byte(`{}`),
			AdditionalMetadata: []byte(`{"team":"a"}`),
		},
		Payload: []byte(`{}`),
	}})
	require.NoError(t, err)
	require.Empty(t, locksNotAcquired)

	since := f.insertedAt.Time.Add(-time.Hour)

	listRuns := func(ids *[]uuid.UUID, metadata map[string]interface{}) []uuid.UUID {
		t.Helper()

		opts := ListWorkflowRunOpts{
			CreatedAfter:                          since,
			Limit:                                 50,
			IncludeTotalCount:                     true,
			ConcurrencySlotWorkflowRunExternalIds: ids,
			AdditionalMetadata:                    metadata,
			AdditionalMetadataOperator:            AdditionalMetadataOperatorAnd,
		}

		runs, total, err := repo.ListWorkflowRuns(ctx, f.tenantId, opts)
		require.NoError(t, err)

		count, err := repo.CountWorkflowRuns(ctx, f.tenantId, opts)
		require.NoError(t, err)
		require.Equal(t, len(runs), total)
		require.Equal(t, len(runs), count)

		res := make([]uuid.UUID, 0, len(runs))
		for _, run := range runs {
			res = append(res, run.ExternalID)
		}

		return res
	}

	listTasks := func(ids *[]uuid.UUID, metadata map[string]interface{}) []uuid.UUID {
		t.Helper()

		opts := ListTaskRunOpts{
			CreatedAfter:                   since,
			Limit:                          50,
			IncludeTotalCount:              true,
			ConcurrencySlotTaskExternalIds: ids,
			AdditionalMetadata:             metadata,
			AdditionalMetadataOperator:     AdditionalMetadataOperatorAnd,
		}

		tasks, total, err := repo.ListTasks(ctx, f.tenantId, opts)
		require.NoError(t, err)

		count, err := repo.CountTasks(ctx, f.tenantId, opts)
		require.NoError(t, err)
		require.Equal(t, len(tasks), total)
		require.Equal(t, len(tasks), count)

		res := make([]uuid.UUID, 0, len(tasks))
		for _, task := range tasks {
			res = append(res, task.ExternalID)
		}

		return res
	}

	ids := func(v ...uuid.UUID) *[]uuid.UUID { return &v }

	t.Run("runs without filter", func(t *testing.T) {
		require.ElementsMatch(t, []uuid.UUID{f.dagExternalId, standaloneExternalId}, listRuns(nil, nil))
	})

	t.Run("runs by DAG id", func(t *testing.T) {
		require.ElementsMatch(t, []uuid.UUID{f.dagExternalId}, listRuns(ids(f.dagExternalId), nil))
	})

	t.Run("runs by standalone task id", func(t *testing.T) {
		require.ElementsMatch(t, []uuid.UUID{standaloneExternalId}, listRuns(ids(standaloneExternalId), nil))
	})

	t.Run("runs with metadata filter", func(t *testing.T) {
		got := listRuns(ids(f.dagExternalId, standaloneExternalId), map[string]interface{}{"team": "a"})
		require.ElementsMatch(t, []uuid.UUID{standaloneExternalId}, got)
	})

	t.Run("tasks by DAG id expand to its tasks", func(t *testing.T) {
		require.ElementsMatch(t, f.childExternalIds, listTasks(ids(f.dagExternalId), nil))
	})

	t.Run("tasks by task id", func(t *testing.T) {
		require.ElementsMatch(t, []uuid.UUID{f.childExternalIds[0]}, listTasks(ids(f.childExternalIds[0]), nil))
	})

	t.Run("tasks with metadata filter", func(t *testing.T) {
		got := listTasks(ids(f.dagExternalId, standaloneExternalId), map[string]interface{}{"team": "a"})
		require.ElementsMatch(t, []uuid.UUID{standaloneExternalId}, got)
	})

	t.Run("empty id list matches nothing", func(t *testing.T) {
		require.Empty(t, listRuns(ids(), nil))
		require.Empty(t, listTasks(ids(), nil))
	})
}
