//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/enums"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

type cancelOnParentCancelEnv struct {
	t        *testing.T
	ctx      context.Context
	pool     *pgxpool.Pool
	repos    userEventScopeTestRepositories
	tasks    *TaskRepositoryImpl
	tenantID uuid.UUID
}

func newCancelOnParentCancelEnv(t *testing.T) *cancelOnParentCancelEnv {
	t.Helper()

	pool, cleanup := setupPostgresWithMigration(t)
	t.Cleanup(cleanup)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	t.Cleanup(cancel)

	repos := newUserEventScopeTestRepositories(t, pool)

	return &cancelOnParentCancelEnv{
		t:     t,
		ctx:   ctx,
		pool:  pool,
		repos: repos,
		tasks: &TaskRepositoryImpl{
			sharedRepository:      repos.shared,
			taskRetentionPeriod:   24 * time.Hour,
			maxInternalRetryCount: 3,
		},
		tenantID: uuid.New(),
	}
}

// insertTask inserts a v1_task row; its v1_lookup_table row is written by the insert trigger.
func (e *cancelOnParentCancelEnv) insertTask(externalID uuid.UUID, isDurable bool, dag *TaskIdInsertedAtRetryCount) TaskIdInsertedAtRetryCount {
	e.t.Helper()

	var dagID *int64
	var dagInsertedAt *time.Time

	if dag != nil {
		dagID = &dag.Id
		dagInsertedAt = &dag.InsertedAt.Time
	}

	var id int64
	var insertedAt time.Time

	err := e.pool.QueryRow(e.ctx, `
		INSERT INTO v1_task (
			tenant_id, queue, action_id, step_id, step_readable_id, workflow_id, workflow_version_id,
			workflow_run_id, schedule_timeout, step_timeout, sticky, external_id, display_name, input,
			step_index, is_durable, dag_id, dag_inserted_at
		)
		VALUES ($1, 'default', 'test:run', $2, 'step', $3, $4, $5, '5m', '30s', 'NONE', $6, 'task', '{}', 0, $7, $8, $9)
		RETURNING id, inserted_at
	`, e.tenantID, uuid.New(), uuid.New(), uuid.New(), externalID, externalID, isDurable, dagID, dagInsertedAt).Scan(&id, &insertedAt)
	require.NoError(e.t, err)

	return TaskIdInsertedAtRetryCount{Id: id, InsertedAt: pgtype.Timestamptz{Time: insertedAt, Valid: true}}
}

// insertDag inserts a v1_dag row with two tasks. The insert triggers write the DAG's
// v1_lookup_table row, which points at the DAG rather than a task, and the v1_dag_to_task rows.
func (e *cancelOnParentCancelEnv) insertDag(externalID uuid.UUID) []TaskIdInsertedAtRetryCount {
	e.t.Helper()

	var dag TaskIdInsertedAtRetryCount

	err := e.pool.QueryRow(e.ctx, `
		INSERT INTO v1_dag (tenant_id, external_id, display_name, workflow_id, workflow_version_id)
		VALUES ($1, $2, 'dag', $3, $4)
		RETURNING id, inserted_at
	`, e.tenantID, externalID, uuid.New(), uuid.New()).Scan(&dag.Id, &dag.InsertedAt)
	require.NoError(e.t, err)

	return []TaskIdInsertedAtRetryCount{
		e.insertTask(uuid.New(), false, &dag),
		e.insertTask(uuid.New(), false, &dag),
	}
}

func (e *cancelOnParentCancelEnv) completeTask(task TaskIdInsertedAtRetryCount) {
	e.t.Helper()

	_, err := e.pool.Exec(e.ctx, `
		INSERT INTO v1_task_event (tenant_id, task_id, task_inserted_at, retry_count, event_type, created_at)
		VALUES ($1, $2, $3, 0, 'COMPLETED', NOW())
	`, e.tenantID, task.Id, task.InsertedAt)
	require.NoError(e.t, err)
}

// createDurableParent inserts a durable v1_task with an event log file, as the engine does when a
// durable task starts.
func (e *cancelOnParentCancelEnv) createDurableParent() (TaskIdInsertedAtRetryCount, *sqlcv1.FlattenExternalIdsRow) {
	e.t.Helper()

	externalID := uuid.New()
	parent := e.insertTask(externalID, true, nil)

	_, err := e.repos.shared.queries.IncrementLogFileInvocationCounts(e.ctx, e.repos.shared.pool.ForShared(), sqlcv1.IncrementLogFileInvocationCountsParams{
		Durabletaskids:         []int64{parent.Id},
		Durabletaskinsertedats: []pgtype.Timestamptz{parent.InsertedAt},
		Tenantids:              []uuid.UUID{e.tenantID},
	})
	require.NoError(e.t, err)

	return parent, &sqlcv1.FlattenExternalIdsRow{
		ID:            parent.Id,
		InsertedAt:    parent.InsertedAt,
		ExternalID:    externalID,
		WorkflowRunID: externalID,
	}
}

// spawnChildren ingests one RUN event with a child per cancelOnParentCancel value and returns the
// external ids the engine reserved for the children.
func (e *cancelOnParentCancelEnv) spawnChildren(parent *sqlcv1.FlattenExternalIdsRow, cancelOnParentCancel ...bool) []uuid.UUID {
	e.t.Helper()

	triggerOpts := make([]*WorkflowNameTriggerOpts, len(cancelOnParentCancel))

	for i, cancel := range cancelOnParentCancel {
		childIndex := int64(i)
		triggerOpts[i] = &WorkflowNameTriggerOpts{
			TriggerTaskData: &TriggerTaskData{
				WorkflowName:         "child",
				Data:                 []byte(`{}`),
				ParentExternalId:     &parent.WorkflowRunID,
				ParentTaskId:         &parent.ID,
				ParentTaskInsertedAt: &parent.InsertedAt.Time,
				ChildIndex:           &childIndex,
				CancelOnParentCancel: cancel,
			},
		}
	}

	result, err := e.repos.durable.IngestDurableTaskEvent(e.ctx, IngestDurableTaskEventOpts{
		BaseIngestEventOpts: &BaseIngestEventOpts{
			Task:            parent,
			Kind:            enums.V1DurableEventLogKindRUN,
			InvocationCount: 1,
			TenantId:        e.tenantID,
		},
		TriggerRuns: &IngestTriggerRunsOpts{TriggerOpts: triggerOpts},
	})
	require.NoError(e.t, err)
	require.Len(e.t, result.TriggerRunsResult.Entries, len(cancelOnParentCancel))

	ids := make([]uuid.UUID, len(cancelOnParentCancel))

	for i, entry := range result.TriggerRunsResult.Entries {
		ids[i] = entry.WorkflowRunExternalId
	}

	return ids
}

func sortedTaskIds(tasks []TaskIdInsertedAtRetryCount) []int64 {
	ids := make([]int64, len(tasks))

	for i, t := range tasks {
		ids[i] = t.Id
	}

	slices.Sort(ids)

	return ids
}

// A cancelled durable task releases a row marked durable, and the lookup from that row returns the
// unfinished tasks of exactly the children spawned with cancel_on_parent_cancel: a standalone
// child's task and every task of a DAG child, but not an opted-out child or a finished one.
func TestCancelledDurableParentListsOptedInChildren(t *testing.T) {
	e := newCancelOnParentCancelEnv(t)

	parent, parentRow := e.createDurableParent()
	childIds := e.spawnChildren(parentRow, true, true, false, true)

	standaloneChild := e.insertTask(childIds[0], false, nil)
	dagChildTasks := e.insertDag(childIds[1])
	e.insertTask(childIds[2], false, nil)
	finishedChild := e.insertTask(childIds[3], false, nil)
	e.completeTask(finishedChild)

	res, err := e.tasks.CancelTasks(e.ctx, e.tenantID, []TaskIdInsertedAtRetryCount{parent})
	require.NoError(t, err)
	require.Len(t, res.ReleasedTasks, 1)
	require.True(t, res.ReleasedTasks[0].IsDurable)
	require.True(t, res.ReleasedTasks[0].IsCurrentRetry)

	children, err := e.tasks.ListUnfinishedCancelOnParentCancelChildren(e.ctx, e.tenantID, []TaskIdInsertedAtRetryCount{{
		Id:         res.ReleasedTasks[0].ID,
		InsertedAt: res.ReleasedTasks[0].InsertedAt,
		RetryCount: res.ReleasedTasks[0].RetryCount,
	}})
	require.NoError(t, err)

	want := append([]TaskIdInsertedAtRetryCount{standaloneChild}, dagChildTasks...)
	require.Equal(t, sortedTaskIds(want), sortedTaskIds(children))
}

// cancel_on_parent_cancel defaults to false: children spawned without it are never listed.
func TestCancelledDurableParentWithoutOptInListsNoChildren(t *testing.T) {
	e := newCancelOnParentCancelEnv(t)

	parent, parentRow := e.createDurableParent()
	childIds := e.spawnChildren(parentRow, false, false)

	e.insertTask(childIds[0], false, nil)
	e.insertDag(childIds[1])

	children, err := e.tasks.ListUnfinishedCancelOnParentCancelChildren(e.ctx, e.tenantID, []TaskIdInsertedAtRetryCount{parent})
	require.NoError(t, err)
	require.Empty(t, children)
}
