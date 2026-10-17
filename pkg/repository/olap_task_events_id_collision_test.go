//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// idCollisionDagFixture is a regular (non-operator) DAG whose task ids are allocated from a
// sequence independent of the DAG id, as in core: v1_dag.id and v1_task.id are separate identity
// columns, so on a fresh database the first DAG gets id 1 and so does its first task.
type idCollisionDagFixture struct {
	tenantId      uuid.UUID
	dagId         int64
	insertedAt    pgtype.Timestamptz
	dagExternalId uuid.UUID
	workflowId    uuid.UUID
	tasks         []*V1TaskWithPayload
}

// seedDagWithCollidingTaskIds writes a DAG with id dagId and numTasks children whose ids start at
// dagId, so exactly one child shares the DAG's id. The DAG and its tasks share one inserted_at, as
// they do in core where both rows are written with CURRENT_TIMESTAMP in one transaction.
func seedDagWithCollidingTaskIds(t *testing.T, ctx context.Context, repo *OLAPRepositoryImpl, dagId int64, numTasks int) idCollisionDagFixture {
	t.Helper()

	f := idCollisionDagFixture{
		tenantId:      uuid.New(),
		dagId:         dagId,
		insertedAt:    pgtype.Timestamptz{Time: time.Now().UTC().Truncate(time.Microsecond), Valid: true},
		dagExternalId: uuid.New(),
		workflowId:    uuid.New(),
	}

	workflowVersionId := uuid.New()

	locksNotAcquired, err := repo.CreateDAGs(ctx, f.tenantId, []*DAGWithData{{
		V1Dag: &sqlcv1.V1Dag{
			ID:                f.dagId,
			InsertedAt:        f.insertedAt,
			TenantID:          f.tenantId,
			ExternalID:        f.dagExternalId,
			DisplayName:       "id-collision-dag",
			WorkflowID:        f.workflowId,
			WorkflowVersionID: workflowVersionId,
		},
		Input:              []byte(`{}`),
		AdditionalMetadata: []byte(`{}`),
	}})
	require.NoError(t, err)
	require.Empty(t, locksNotAcquired)

	for i := range numTasks {
		f.tasks = append(f.tasks, &V1TaskWithPayload{
			V1Task: &sqlcv1.V1Task{
				ID:                 f.dagId + int64(i),
				InsertedAt:         f.insertedAt,
				TenantID:           f.tenantId,
				Queue:              "default",
				ActionID:           "test:id-collision-dag-child",
				StepID:             uuid.New(),
				WorkflowID:         f.workflowId,
				WorkflowVersionID:  workflowVersionId,
				WorkflowRunID:      f.dagExternalId,
				ScheduleTimeout:    "5m",
				Sticky:             sqlcv1.V1StickyStrategyNONE,
				ExternalID:         uuid.New(),
				DisplayName:        "id-collision-dag-child",
				Input:              []byte(`{}`),
				AdditionalMetadata: []byte(`{}`),
				DagID:              pgtype.Int8{Int64: f.dagId, Valid: true},
				DagInsertedAt:      f.insertedAt,
			},
			Payload: []byte(`{}`),
		})
	}

	_, locksNotAcquired, err = repo.CreateTasks(ctx, f.tenantId, f.tasks)
	require.NoError(t, err)
	require.Empty(t, locksNotAcquired)

	return f
}

func (f idCollisionDagFixture) event(task *V1TaskWithPayload, eventType sqlcv1.V1EventTypeOlap, status sqlcv1.V1ReadableStatusOlap, retryCount int32) sqlcv1.CreateTaskEventsOLAPParams {
	return sqlcv1.CreateTaskEventsOLAPParams{
		TenantID:       f.tenantId,
		TaskID:         task.ID,
		TaskInsertedAt: f.insertedAt,
		EventType:      eventType,
		WorkflowID:     f.workflowId,
		EventTimestamp: pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
		ReadableStatus: status,
		RetryCount:     retryCount,
		Output:         []byte(`{}`),
		ExternalID:     uuid.New(),
	}
}

func (f idCollisionDagFixture) applyEvents(t *testing.T, ctx context.Context, repo *OLAPRepositoryImpl, events []sqlcv1.CreateTaskEventsOLAPParams) {
	t.Helper()

	eventExternalIdToWorkflowRunId := make(map[uuid.UUID]uuid.UUID, len(events))
	for _, e := range events {
		eventExternalIdToWorkflowRunId[e.ExternalID] = f.dagExternalId
	}

	_, locksNotAcquired, err := repo.CreateTaskEvents(ctx, f.tenantId, events, eventExternalIdToWorkflowRunId, nil, nil, nil)
	require.NoError(t, err)
	require.Empty(t, locksNotAcquired)
}

// eventTypesByTask groups the listed event types by task id.
func eventTypesByTask(rows []*TaskEventWithPayloads) map[int64][]sqlcv1.V1EventTypeOlap {
	out := make(map[int64][]sqlcv1.V1EventTypeOlap)

	for _, row := range rows {
		out[row.TaskID] = append(out[row.TaskID], row.EventType)
	}

	return out
}

// TestListTaskRunEventsByWorkflowRunId_TaskIdEqualToDagId reproduces the fresh-database case
// behind the flaky Ruby non_retryable spec: the first DAG has id 1 and its tasks have ids 1..3.
// The orchestrator-hiding predicate used to treat the task whose id equals the DAG id as the DAG
// operator's orchestrator and drop every one of its events from the run's event list.
func TestListTaskRunEventsByWorkflowRunId_TaskIdEqualToDagId(t *testing.T) {
	basePool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	pool := createEnumAwarePool(t, basePool)
	repo := createOLAPRepositoryWithPayloadStore(t, pool)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	require.NoError(t, repo.UpdateTablePartitions(ctx))

	f := seedDagWithCollidingTaskIds(t, ctx, repo, 1, 3)
	colliding, succeeding, failing := f.tasks[0], f.tasks[1], f.tasks[2]
	require.Equal(t, f.dagId, colliding.ID, "fixture must contain a task whose id equals the dag id")

	// Mirror the non_retryable example: one task fails and retries once, one succeeds, one fails
	// without retrying. The colliding task is the retrying one so the RETRYING event is at risk.
	f.applyEvents(t, ctx, repo, []sqlcv1.CreateTaskEventsOLAPParams{
		f.event(colliding, sqlcv1.V1EventTypeOlapQUEUED, sqlcv1.V1ReadableStatusOlapQUEUED, 0),
		f.event(colliding, sqlcv1.V1EventTypeOlapSTARTED, sqlcv1.V1ReadableStatusOlapRUNNING, 0),
		f.event(colliding, sqlcv1.V1EventTypeOlapFAILED, sqlcv1.V1ReadableStatusOlapFAILED, 0),
		f.event(colliding, sqlcv1.V1EventTypeOlapRETRYING, sqlcv1.V1ReadableStatusOlapQUEUED, 1),
		f.event(colliding, sqlcv1.V1EventTypeOlapQUEUED, sqlcv1.V1ReadableStatusOlapQUEUED, 1),
		f.event(colliding, sqlcv1.V1EventTypeOlapSTARTED, sqlcv1.V1ReadableStatusOlapRUNNING, 1),
		f.event(colliding, sqlcv1.V1EventTypeOlapFAILED, sqlcv1.V1ReadableStatusOlapFAILED, 1),
		f.event(succeeding, sqlcv1.V1EventTypeOlapQUEUED, sqlcv1.V1ReadableStatusOlapQUEUED, 0),
		f.event(succeeding, sqlcv1.V1EventTypeOlapSTARTED, sqlcv1.V1ReadableStatusOlapRUNNING, 0),
		f.event(succeeding, sqlcv1.V1EventTypeOlapFINISHED, sqlcv1.V1ReadableStatusOlapCOMPLETED, 0),
		f.event(failing, sqlcv1.V1EventTypeOlapQUEUED, sqlcv1.V1ReadableStatusOlapQUEUED, 0),
		f.event(failing, sqlcv1.V1EventTypeOlapSTARTED, sqlcv1.V1ReadableStatusOlapRUNNING, 0),
		f.event(failing, sqlcv1.V1EventTypeOlapFAILED, sqlcv1.V1ReadableStatusOlapFAILED, 0),
	})

	for _, includeOrchestratorEvents := range []bool{false, true} {
		rows, err := repo.ListTaskRunEventsByWorkflowRunId(ctx, f.tenantId, f.dagExternalId, includeOrchestratorEvents)
		require.NoError(t, err)

		byTask := eventTypesByTask(rows)

		assert.ElementsMatch(t, []sqlcv1.V1EventTypeOlap{
			sqlcv1.V1EventTypeOlapQUEUED, sqlcv1.V1EventTypeOlapSTARTED, sqlcv1.V1EventTypeOlapFAILED,
			sqlcv1.V1EventTypeOlapRETRYING, sqlcv1.V1EventTypeOlapQUEUED, sqlcv1.V1EventTypeOlapSTARTED, sqlcv1.V1EventTypeOlapFAILED,
		}, byTask[colliding.ID], "events of the task whose id equals the dag id (includeOrchestratorEvents=%v)", includeOrchestratorEvents)
		assert.ElementsMatch(t, []sqlcv1.V1EventTypeOlap{
			sqlcv1.V1EventTypeOlapQUEUED, sqlcv1.V1EventTypeOlapSTARTED, sqlcv1.V1EventTypeOlapFINISHED,
		}, byTask[succeeding.ID])
		assert.ElementsMatch(t, []sqlcv1.V1EventTypeOlap{
			sqlcv1.V1EventTypeOlapQUEUED, sqlcv1.V1EventTypeOlapSTARTED, sqlcv1.V1EventTypeOlapFAILED,
		}, byTask[failing.ID])
		assert.Len(t, rows, 13, "every recorded event is listed exactly once")

		for _, row := range rows {
			assert.Equal(t, "id-collision-dag-child", row.DisplayName, "task %d is reported as a task, not as the dag", row.TaskID)
		}
	}
}

// TestListTaskRunEventsByWorkflowRunId_OperatorDagHidesOrchestrator pins the behaviour the
// predicate exists for: an operator DAG's orchestrator shares the DAG's id and its events are
// hidden by default once children exist, shown on request, and shown while no child exists yet.
func TestListTaskRunEventsByWorkflowRunId_OperatorDagHidesOrchestrator(t *testing.T) {
	basePool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	pool := createEnumAwarePool(t, basePool)
	repo := createOLAPRepositoryWithPayloadStore(t, pool)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	require.NoError(t, repo.UpdateTablePartitions(ctx))

	f := seedOperatorDag(t, ctx, repo, 1)
	f.applyOrchestratorMonitoringEvents(t, ctx, repo,
		f.orchestratorEvent(sqlcv1.V1EventTypeOlapQUEUED, sqlcv1.V1ReadableStatusOlapQUEUED, 0),
		f.orchestratorEvent(sqlcv1.V1EventTypeOlapSTARTED, sqlcv1.V1ReadableStatusOlapRUNNING, 0),
	)

	rows, err := repo.ListTaskRunEventsByWorkflowRunId(ctx, f.tenantId, f.dagExternalId, false)
	require.NoError(t, err)
	assert.Len(t, eventTypesByTask(rows)[f.dagId], 2, "orchestrator events are listed while the dag has no children")

	child := f.createChild(t, ctx, repo, 2)
	f.applyChildEvents(t, ctx, repo, []sqlcv1.CreateTaskEventsOLAPParams{
		child.event(sqlcv1.V1EventTypeOlapQUEUED, sqlcv1.V1ReadableStatusOlapQUEUED, 0),
		child.event(sqlcv1.V1EventTypeOlapSTARTED, sqlcv1.V1ReadableStatusOlapRUNNING, 0),
		child.event(sqlcv1.V1EventTypeOlapFINISHED, sqlcv1.V1ReadableStatusOlapCOMPLETED, 0),
	})

	rows, err = repo.ListTaskRunEventsByWorkflowRunId(ctx, f.tenantId, f.dagExternalId, false)
	require.NoError(t, err)
	byTask := eventTypesByTask(rows)
	assert.Empty(t, byTask[f.dagId], "orchestrator events are hidden by default once a child exists")
	assert.Len(t, byTask[child.taskId], 3)

	rows, err = repo.ListTaskRunEventsByWorkflowRunId(ctx, f.tenantId, f.dagExternalId, true)
	require.NoError(t, err)
	byTask = eventTypesByTask(rows)
	assert.Len(t, byTask[f.dagId], 2, "orchestrator events are listed when requested")
	assert.Len(t, byTask[child.taskId], 3)
}
