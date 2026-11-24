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

type pendingChildRunFixture struct {
	repos   userEventScopeTestRepositories
	tasks   *TaskRepositoryImpl
	parents []*V1TaskWithPayload
}

const pendingChildWorkflowName = "durable-child"

// newPendingChildRunFixture creates n durable parent tasks. Each parent has one RUN entry
// for a child that is ingested but not yet triggered.
func newPendingChildRunFixture(t *testing.T, ctx context.Context, n int) pendingChildRunFixture {
	t.Helper()

	pool, cleanup := setupPostgresWithMigration(t)
	t.Cleanup(cleanup)

	repos := newUserEventScopeTestRepositories(t, pool)
	workflows := &workflowRepository{sharedRepository: repos.shared}

	_, err := workflows.PutWorkflowVersion(ctx, internalTenantId, minimalWorkflowOpts("durable-parent", "v1", nil))
	require.NoError(t, err)

	_, err = workflows.PutWorkflowVersion(ctx, internalTenantId, minimalWorkflowOpts(pendingChildWorkflowName, "v1", nil))
	require.NoError(t, err)

	triggerOpts := make([]*WorkflowNameTriggerOpts, n)
	for i := range triggerOpts {
		triggerOpts[i] = &WorkflowNameTriggerOpts{
			TriggerTaskData: &TriggerTaskData{WorkflowName: "durable-parent", Data: []byte(`{}`)},
			ExternalId:      uuid.New(),
		}
	}

	parents, _, _, _, err := newTriggerRepository(repos.shared).TriggerFromWorkflowNames(ctx, internalTenantId, triggerOpts)
	require.NoError(t, err)
	require.Len(t, parents, n)

	return pendingChildRunFixture{
		repos:   repos,
		tasks:   &TaskRepositoryImpl{sharedRepository: repos.shared, maxInternalRetryCount: 3},
		parents: parents,
	}
}

func flattenedParent(parent *V1TaskWithPayload) *sqlcv1.FlattenExternalIdsRow {
	return &sqlcv1.FlattenExternalIdsRow{
		ID:         parent.ID,
		InsertedAt: parent.InsertedAt,
		ExternalID: parent.ExternalID,
	}
}

func (f pendingChildRunFixture) ingestPendingChildRun(t *testing.T, ctx context.Context, parent *V1TaskWithPayload) TriggerPendingRunEntriesOpt {
	t.Helper()

	task := flattenedParent(parent)

	reinvokeDurableTask(t, ctx, f.repos, internalTenantId, task)

	parentTaskInsertedAt := parent.InsertedAt.Time
	childIndex := int64(0)

	result, err := f.repos.durable.IngestDurableTaskEvent(ctx, IngestDurableTaskEventOpts{
		BaseIngestEventOpts: &BaseIngestEventOpts{
			Task:            task,
			Kind:            enums.V1DurableEventLogKindRUN,
			InvocationCount: 1,
			TenantId:        internalTenantId,
		},
		TriggerRuns: &IngestTriggerRunsOpts{
			TriggerOpts: []*WorkflowNameTriggerOpts{{
				TriggerTaskData: &TriggerTaskData{
					WorkflowName:         pendingChildWorkflowName,
					Data:                 []byte(`{}`),
					ParentExternalId:     &parent.ExternalID,
					ParentTaskId:         &parent.ID,
					ParentTaskInsertedAt: &parentTaskInsertedAt,
					ChildIndex:           &childIndex,
				},
			}},
		},
	})
	require.NoError(t, err)
	require.Len(t, result.TriggerRunsResult.PendingTriggers, 1)

	return TriggerPendingRunEntriesOpt{
		Task:        task,
		PendingRuns: result.TriggerRunsResult.PendingTriggers,
	}
}

func (f pendingChildRunFixture) requireChildMaterialized(t *testing.T, ctx context.Context, pending TriggerPendingRunEntriesOpt, expected bool) {
	t.Helper()

	created, _, _, err := f.repos.durable.TriggerPendingRunEntries(ctx, internalTenantId, []TriggerPendingRunEntriesOpt{pending})
	require.NoError(t, err)

	run := pending.PendingRuns[0]
	childExternalId := run.TriggerOpts.ExternalId

	var childTaskCount int
	require.NoError(t, f.repos.shared.pool.ForShared().QueryRow(ctx,
		`SELECT count(*) FROM v1_task WHERE external_id = $1`, childExternalId,
	).Scan(&childTaskCount))

	entry, err := f.repos.shared.queries.GetDurableEventLogEntry(ctx, f.repos.shared.pool.ForShared(), sqlcv1.GetDurableEventLogEntryParams{
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

func (f pendingChildRunFixture) cancel(t *testing.T, ctx context.Context, parent *V1TaskWithPayload, retryCount int32) {
	t.Helper()

	_, err := f.tasks.CancelTasks(ctx, internalTenantId, []TaskIdInsertedAtRetryCount{{
		Id:         parent.ID,
		InsertedAt: parent.InsertedAt,
		RetryCount: retryCount,
	}})
	require.NoError(t, err)
}

func TestTriggerPendingRunEntriesSkipsAChildOfACancelledParent(t *testing.T) {
	ctx := context.Background()
	f := newPendingChildRunFixture(t, ctx, 2)

	cancelled := f.parents[0]
	cancelledPending := f.ingestPendingChildRun(t, ctx, cancelled)
	f.cancel(t, ctx, cancelled, 0)

	// a cancelled parent in the same batch doesn't hold back another parent's child
	running := f.parents[1]
	runningPending := f.ingestPendingChildRun(t, ctx, running)

	created, _, _, err := f.repos.durable.TriggerPendingRunEntries(ctx, internalTenantId, []TriggerPendingRunEntriesOpt{cancelledPending, runningPending})
	require.NoError(t, err)
	require.Len(t, created, 1)
	require.Equal(t, runningPending.PendingRuns[0].TriggerOpts.ExternalId, created[0].ExternalID)

	f.requireChildMaterialized(t, ctx, cancelledPending, false)
}

func TestTriggerPendingRunEntriesMaterializesAChildOfAnEvictedParent(t *testing.T) {
	ctx := context.Background()
	f := newPendingChildRunFixture(t, ctx, 1)
	parent := f.parents[0]

	pending := f.ingestPendingChildRun(t, ctx, parent)

	_, err := f.repos.shared.pool.ForShared().Exec(ctx,
		`INSERT INTO v1_task_runtime (task_id, task_inserted_at, retry_count, worker_id, tenant_id, timeout_at)
		 VALUES ($1, $2, 0, $3, $4, $5)`,
		parent.ID, parent.InsertedAt, uuid.New(), internalTenantId, time.Now().Add(time.Hour),
	)
	require.NoError(t, err)

	evicted, err := f.tasks.EvictTask(ctx, internalTenantId, TaskIdInsertedAtRetryCount{
		Id:         parent.ID,
		InsertedAt: parent.InsertedAt,
		RetryCount: 0,
	})
	require.NoError(t, err)
	require.True(t, evicted.WasEvicted)

	f.requireChildMaterialized(t, ctx, pending, true)
}

func TestTriggerPendingRunEntriesMaterializesAChildOfAReplayedParent(t *testing.T) {
	ctx := context.Background()
	f := newPendingChildRunFixture(t, ctx, 1)
	parent := f.parents[0]

	pending := f.ingestPendingChildRun(t, ctx, parent)
	f.cancel(t, ctx, parent, 0)

	f.requireChildMaterialized(t, ctx, pending, false)

	replayed, err := f.repos.shared.replayTasks(ctx, f.repos.shared.pool.ForShared(), internalTenantId, []ReplayTaskOpts{{
		TaskId:       parent.ID,
		InsertedAt:   parent.InsertedAt,
		ExternalId:   parent.ExternalID,
		StepId:       parent.StepID,
		Input:        f.repos.shared.newTaskInputFromTaskInputPayload(taskInputPayloadVersion(parent.TaskInputPayloadVersion), parent.TaskInputPayload),
		InitialState: sqlcv1.V1TaskInitialStateQUEUED,
	}})
	require.NoError(t, err)
	require.Len(t, replayed, 1)
	require.EqualValues(t, 1, replayed[0].RetryCount)

	f.requireChildMaterialized(t, ctx, pending, true)
}

func TestTriggerPendingRunEntriesMaterializesAChildOfACompletedParent(t *testing.T) {
	ctx := context.Background()
	f := newPendingChildRunFixture(t, ctx, 1)
	parent := f.parents[0]

	pending := f.ingestPendingChildRun(t, ctx, parent)

	_, err := f.tasks.CompleteTasks(ctx, internalTenantId, []CompleteTaskOpts{{
		TaskIdInsertedAtRetryCount: &TaskIdInsertedAtRetryCount{
			Id:         parent.ID,
			InsertedAt: parent.InsertedAt,
			RetryCount: 0,
		},
		Output: []byte(`{}`),
	}})
	require.NoError(t, err)

	f.requireChildMaterialized(t, ctx, pending, true)
}
