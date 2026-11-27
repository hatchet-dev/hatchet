//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/enums"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// TestTriggerPendingRunEntriesCancelledParent checks that a pending child run is not created while
// its parent's current retry is cancelled, and is created for a running, evicted, replayed or
// completed parent.
func TestTriggerPendingRunEntriesCancelledParent(t *testing.T) {
	ctx := context.Background()

	pool, cleanup := setupPostgresWithMigration(t)
	t.Cleanup(cleanup)

	repos := newUserEventScopeTestRepositories(t, pool)
	tasks := &TaskRepositoryImpl{sharedRepository: repos.shared, maxInternalRetryCount: 3}
	triggers := newTriggerRepository(repos.shared)
	workflows := &workflowRepository{sharedRepository: repos.shared}

	const childWorkflowName = "durable-child"

	_, err := workflows.PutWorkflowVersion(ctx, internalTenantId, minimalWorkflowOpts("durable-parent", "v1", nil))
	require.NoError(t, err)

	_, err = workflows.PutWorkflowVersion(ctx, internalTenantId, minimalWorkflowOpts(childWorkflowName, "v1", nil))
	require.NoError(t, err)

	newPendingParent := func(t *testing.T) (*V1TaskWithPayload, TriggerPendingRunEntriesOpt) {
		t.Helper()

		parents, _, _, _, err := triggers.TriggerFromWorkflowNames(ctx, internalTenantId, []*WorkflowNameTriggerOpts{{
			TriggerTaskData: &TriggerTaskData{WorkflowName: "durable-parent", Data: []byte(`{}`)},
			ExternalId:      uuid.New(),
		}})
		require.NoError(t, err)
		require.Len(t, parents, 1)

		parent := parents[0]
		task := &sqlcv1.FlattenExternalIdsRow{
			ID:         parent.ID,
			InsertedAt: parent.InsertedAt,
			ExternalID: parent.ExternalID,
		}

		reinvokeDurableTask(t, ctx, repos, internalTenantId, task)

		// a parent which spawns children is assigned to a worker, so it has a runtime on its current retry
		_, err = repos.shared.pool.ForShared().Exec(ctx,
			`INSERT INTO v1_task_runtime (task_id, task_inserted_at, retry_count, worker_id, tenant_id, timeout_at)
			 VALUES ($1, $2, 0, $3, $4, $5)`,
			parent.ID, parent.InsertedAt, uuid.New(), internalTenantId, time.Now().Add(time.Hour),
		)
		require.NoError(t, err)

		parentInsertedAt := parent.InsertedAt.Time
		childIndex := int64(0)

		result, err := repos.durable.IngestDurableTaskEvent(ctx, IngestDurableTaskEventOpts{
			BaseIngestEventOpts: &BaseIngestEventOpts{
				Task:            task,
				Kind:            enums.V1DurableEventLogKindRUN,
				InvocationCount: 1,
				TenantId:        internalTenantId,
			},
			TriggerRuns: &IngestTriggerRunsOpts{
				TriggerOpts: []*WorkflowNameTriggerOpts{{
					TriggerTaskData: &TriggerTaskData{
						WorkflowName:         childWorkflowName,
						Data:                 []byte(`{}`),
						ParentExternalId:     &parent.ExternalID,
						ParentTaskId:         &parent.ID,
						ParentTaskInsertedAt: &parentInsertedAt,
						ChildIndex:           &childIndex,
					},
				}},
			},
		})
		require.NoError(t, err)
		require.Len(t, result.TriggerRunsResult.PendingTriggers, 1)

		return parent, TriggerPendingRunEntriesOpt{
			Task:        task,
			PendingRuns: result.TriggerRunsResult.PendingTriggers,
		}
	}

	cancelParent := func(t *testing.T, parent *V1TaskWithPayload) {
		t.Helper()

		_, err := tasks.CancelTasks(ctx, internalTenantId, []TaskIdInsertedAtRetryCount{{
			Id:         parent.ID,
			InsertedAt: parent.InsertedAt,
			RetryCount: 0,
		}})
		require.NoError(t, err)
	}

	requireMaterialized := func(t *testing.T, pending TriggerPendingRunEntriesOpt, expected bool) {
		t.Helper()

		created, _, _, err := repos.durable.TriggerPendingRunEntries(ctx, internalTenantId, []TriggerPendingRunEntriesOpt{pending})
		require.NoError(t, err)

		run := pending.PendingRuns[0]
		childExternalId := run.TriggerOpts.ExternalId

		var childTaskCount int
		require.NoError(t, repos.shared.pool.ForShared().QueryRow(ctx,
			`SELECT count(*) FROM v1_task WHERE external_id = $1`, childExternalId,
		).Scan(&childTaskCount))

		entry, err := repos.shared.queries.GetDurableEventLogEntry(ctx, repos.shared.pool.ForShared(), sqlcv1.GetDurableEventLogEntryParams{
			Durabletaskid:         pending.Task.ID,
			Durabletaskinsertedat: pending.Task.InsertedAt,
			Nodeid:                run.NodeId,
			Branchid:              run.BranchId,
		})
		require.NoError(t, err)

		if expected {
			require.Len(t, created, 1)
			require.Equal(t, childExternalId, created[0].ExternalID)
			require.Equal(t, 1, childTaskCount)
			require.True(t, entry.TriggeredAt.Valid)
		} else {
			require.Empty(t, created)
			require.Zero(t, childTaskCount)
			require.False(t, entry.TriggeredAt.Valid, "the entry stays untriggered so a replay of the parent can trigger it")
		}
	}

	t.Run("skips cancelled parent without blocking a running sibling in the same batch", func(t *testing.T) {
		cancelled, cancelledPending := newPendingParent(t)
		cancelParent(t, cancelled)

		_, runningPending := newPendingParent(t)

		created, _, _, err := repos.durable.TriggerPendingRunEntries(ctx, internalTenantId, []TriggerPendingRunEntriesOpt{cancelledPending, runningPending})
		require.NoError(t, err)
		require.Len(t, created, 1)
		require.Equal(t, runningPending.PendingRuns[0].TriggerOpts.ExternalId, created[0].ExternalID)

		requireMaterialized(t, cancelledPending, false)
	})

	t.Run("materializes child of an evicted parent", func(t *testing.T) {
		parent, pending := newPendingParent(t)

		evicted, err := tasks.EvictTask(ctx, internalTenantId, TaskIdInsertedAtRetryCount{
			Id:         parent.ID,
			InsertedAt: parent.InsertedAt,
			RetryCount: 0,
		})
		require.NoError(t, err)
		require.True(t, evicted.WasEvicted)

		requireMaterialized(t, pending, true)
	})

	t.Run("materializes child after a cancelled parent is replayed", func(t *testing.T) {
		parent, pending := newPendingParent(t)
		cancelParent(t, parent)

		requireMaterialized(t, pending, false)

		replayed, err := repos.shared.replayTasks(ctx, repos.shared.pool.ForShared(), internalTenantId, []ReplayTaskOpts{{
			TaskId:       parent.ID,
			InsertedAt:   parent.InsertedAt,
			ExternalId:   parent.ExternalID,
			StepId:       parent.StepID,
			Input:        repos.shared.newTaskInputFromTaskInputPayload(taskInputPayloadVersion(parent.TaskInputPayloadVersion), parent.TaskInputPayload),
			InitialState: sqlcv1.V1TaskInitialStateQUEUED,
		}})
		require.NoError(t, err)
		require.Len(t, replayed, 1)
		require.EqualValues(t, 1, replayed[0].RetryCount)

		requireMaterialized(t, pending, true)
	})

	t.Run("materializes child of a completed parent", func(t *testing.T) {
		parent, pending := newPendingParent(t)

		_, err := tasks.CompleteTasks(ctx, internalTenantId, []CompleteTaskOpts{{
			TaskIdInsertedAtRetryCount: &TaskIdInsertedAtRetryCount{
				Id:         parent.ID,
				InsertedAt: parent.InsertedAt,
				RetryCount: 0,
			},
			Output: []byte(`{}`),
		}})
		require.NoError(t, err)

		requireMaterialized(t, pending, true)
	})
}
