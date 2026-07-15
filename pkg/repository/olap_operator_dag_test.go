//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

type operatorDagFixture struct {
	tenantId      uuid.UUID
	dagId         int64
	dagInsertedAt pgtype.Timestamptz
	dagExternalId uuid.UUID
	workflowId    uuid.UUID
}

func seedOperatorDag(t *testing.T, ctx context.Context, repo *OLAPRepositoryImpl, dagId int64, totalTasks int) operatorDagFixture {
	t.Helper()

	f := operatorDagFixture{
		tenantId:      uuid.New(),
		dagId:         dagId,
		dagInsertedAt: pgtype.Timestamptz{Time: time.Now().UTC().Truncate(time.Microsecond), Valid: true},
		dagExternalId: uuid.New(),
		workflowId:    uuid.New(),
	}

	dag := &DAGWithData{
		V1Dag: &sqlcv1.V1Dag{
			ID:                f.dagId,
			InsertedAt:        f.dagInsertedAt,
			TenantID:          f.tenantId,
			ExternalID:        f.dagExternalId,
			DisplayName:       "operator-dag-test",
			WorkflowID:        f.workflowId,
			WorkflowVersionID: uuid.New(),
		},
		Input:              []byte(`{}`),
		AdditionalMetadata: []byte(`{}`),
		TotalTasks:         totalTasks,
		IsOperatorRun:      true,
	}

	locksNotAcquired, err := repo.CreateDAGs(ctx, f.tenantId, []*DAGWithData{dag})
	require.NoError(t, err)
	require.Empty(t, locksNotAcquired)

	return f
}

func (f operatorDagFixture) createChild(t *testing.T, ctx context.Context, repo *OLAPRepositoryImpl, taskId int64) replayStatusFixture {
	t.Helper()

	child := replayStatusFixture{
		tenantId:   f.tenantId,
		taskId:     taskId,
		insertedAt: pgtype.Timestamptz{Time: time.Now().UTC().Truncate(time.Microsecond), Valid: true},
		externalId: uuid.New(),
		workflowId: f.workflowId,
		workerId:   uuid.New(),
	}

	task := &V1TaskWithPayload{
		V1Task: &sqlcv1.V1Task{
			ID:                 child.taskId,
			InsertedAt:         child.insertedAt,
			TenantID:           child.tenantId,
			Queue:              "default",
			ActionID:           "test:operator-dag-child",
			StepID:             uuid.New(),
			WorkflowID:         child.workflowId,
			WorkflowVersionID:  uuid.New(),
			WorkflowRunID:      f.dagExternalId,
			ScheduleTimeout:    "5m",
			StepTimeout:        pgtype.Text{String: "60s", Valid: true},
			Priority:           pgtype.Int4{Int32: 1, Valid: true},
			Sticky:             sqlcv1.V1StickyStrategyNONE,
			ExternalID:         child.externalId,
			DisplayName:        "operator-dag-child",
			Input:              []byte(`{}`),
			AdditionalMetadata: []byte(`{}`),
			DagID:              pgtype.Int8{Int64: f.dagId, Valid: true},
			DagInsertedAt:      f.dagInsertedAt,
		},
		Payload: []byte(`{}`),
	}

	_, locksNotAcquired, err := repo.CreateTasks(ctx, child.tenantId, []*V1TaskWithPayload{task})
	require.NoError(t, err)
	require.Empty(t, locksNotAcquired)

	return child
}

func (f operatorDagFixture) applyChildEvents(t *testing.T, ctx context.Context, repo *OLAPRepositoryImpl, events []sqlcv1.CreateTaskEventsOLAPParams) {
	t.Helper()

	eventExternalIdToWorkflowRunId := make(map[uuid.UUID]uuid.UUID)
	for _, e := range events {
		eventExternalIdToWorkflowRunId[e.ExternalID] = f.dagExternalId
	}

	_, locksNotAcquired, err := repo.CreateTaskEvents(ctx, f.tenantId, events, eventExternalIdToWorkflowRunId)
	require.NoError(t, err)
	require.Empty(t, locksNotAcquired)
}

func (f operatorDagFixture) assertDagStatus(t *testing.T, ctx context.Context, pool *pgxpool.Pool, wantStatus string) {
	t.Helper()

	var status string

	err := pool.QueryRow(ctx, `
		SELECT readable_status::text
		FROM v1_dags_olap
		WHERE tenant_id = $1 AND id = $2
	`, f.tenantId, f.dagId).Scan(&status)
	require.NoError(t, err)

	assert.Equal(t, wantStatus, status, "v1_dags_olap.readable_status")

	var runStatus string

	err = pool.QueryRow(ctx, `
		SELECT readable_status::text
		FROM v1_runs_olap
		WHERE tenant_id = $1 AND external_id = $2
	`, f.tenantId, f.dagExternalId).Scan(&runStatus)
	require.NoError(t, err)

	assert.Equal(t, wantStatus, runStatus, "v1_runs_olap.readable_status")
}

func TestOperatorDAG_ChildrenConvergeStatus(t *testing.T) {
	basePool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	pool := createEnumAwarePool(t, basePool)
	repo := createOLAPRepositoryWithPayloadStore(t, pool)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	require.NoError(t, repo.UpdateTablePartitions(ctx))

	f := seedOperatorDag(t, ctx, repo, 100, 2)

	// the DAG row exists, plus a self-mapping junction row for the orchestrator's events
	var selfMappings int
	err := pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM v1_dag_to_task_olap
		WHERE (dag_id, task_id) = ($1, $1)
	`, f.dagId).Scan(&selfMappings)
	require.NoError(t, err)
	assert.Equal(t, 1, selfMappings, "self-mapping junction row")

	f.assertDagStatus(t, ctx, pool, "QUEUED")

	childA := f.createChild(t, ctx, repo, 101)
	childB := f.createChild(t, ctx, repo, 102)

	// children must not appear as standalone runs
	var childRuns int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM v1_runs_olap
		WHERE tenant_id = $1 AND external_id IN ($2, $3)
	`, f.tenantId, childA.externalId, childB.externalId).Scan(&childRuns)
	require.NoError(t, err)
	assert.Equal(t, 0, childRuns, "children should not be v1_runs_olap rows")

	f.applyChildEvents(t, ctx, repo, []sqlcv1.CreateTaskEventsOLAPParams{
		childA.event(sqlcv1.V1EventTypeOlapSTARTED, sqlcv1.V1ReadableStatusOlapRUNNING, 0),
		childA.event(sqlcv1.V1EventTypeOlapFINISHED, sqlcv1.V1ReadableStatusOlapCOMPLETED, 0),
	})

	// one of two children finished: the DAG is still running
	f.assertDagStatus(t, ctx, pool, "RUNNING")

	f.applyChildEvents(t, ctx, repo, []sqlcv1.CreateTaskEventsOLAPParams{
		childB.event(sqlcv1.V1EventTypeOlapSTARTED, sqlcv1.V1ReadableStatusOlapRUNNING, 0),
		childB.event(sqlcv1.V1EventTypeOlapFINISHED, sqlcv1.V1ReadableStatusOlapCOMPLETED, 0),
	})

	f.assertDagStatus(t, ctx, pool, "COMPLETED")
}

func TestOperatorDAG_OrchestratorFailureOverride(t *testing.T) {
	basePool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	pool := createEnumAwarePool(t, basePool)
	repo := createOLAPRepositoryWithPayloadStore(t, pool)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	require.NoError(t, repo.UpdateTablePartitions(ctx))

	// total_tasks = 3, but only one child will ever be created: the operator died mid-run
	f := seedOperatorDag(t, ctx, repo, 200, 3)

	// a non-reset RUNNING event (orchestrator started) moves the DAG out of QUEUED
	result, err := repo.ApplyOrchestratorEventsToDAGs(ctx, f.tenantId, []OrchestratorDAGStatusUpdate{
		{DagId: f.dagId, DagInsertedAt: f.dagInsertedAt, ReadableStatus: sqlcv1.V1ReadableStatusOlapRUNNING},
	})
	require.NoError(t, err)
	require.Len(t, result.DAGRows, 1)
	f.assertDagStatus(t, ctx, pool, "RUNNING")

	child := f.createChild(t, ctx, repo, 201)
	f.applyChildEvents(t, ctx, repo, []sqlcv1.CreateTaskEventsOLAPParams{
		child.event(sqlcv1.V1EventTypeOlapSTARTED, sqlcv1.V1ReadableStatusOlapRUNNING, 0),
	})

	// orchestrator failed terminally (e.g. timed out) while the run is incomplete
	result, err = repo.ApplyOrchestratorEventsToDAGs(ctx, f.tenantId, []OrchestratorDAGStatusUpdate{
		{DagId: f.dagId, DagInsertedAt: f.dagInsertedAt, ReadableStatus: sqlcv1.V1ReadableStatusOlapFAILED},
	})
	require.NoError(t, err)
	require.Len(t, result.DAGRows, 1)
	f.assertDagStatus(t, ctx, pool, "FAILED")

	// a late child event recomputes the DAG status from counts; with only 1 of 3
	// tasks created, the recompute must not downgrade the terminal status
	f.applyChildEvents(t, ctx, repo, []sqlcv1.CreateTaskEventsOLAPParams{
		child.event(sqlcv1.V1EventTypeOlapFINISHED, sqlcv1.V1ReadableStatusOlapCOMPLETED, 0),
	})
	f.assertDagStatus(t, ctx, pool, "FAILED")

	// a non-reset RUNNING event must not revive a failed DAG
	result, err = repo.ApplyOrchestratorEventsToDAGs(ctx, f.tenantId, []OrchestratorDAGStatusUpdate{
		{DagId: f.dagId, DagInsertedAt: f.dagInsertedAt, ReadableStatus: sqlcv1.V1ReadableStatusOlapRUNNING},
	})
	require.NoError(t, err)
	require.Empty(t, result.DAGRows)
	f.assertDagStatus(t, ctx, pool, "FAILED")

	// a retry (user replay) resets the DAG back to RUNNING
	result, err = repo.ApplyOrchestratorEventsToDAGs(ctx, f.tenantId, []OrchestratorDAGStatusUpdate{
		{DagId: f.dagId, DagInsertedAt: f.dagInsertedAt, ReadableStatus: sqlcv1.V1ReadableStatusOlapRUNNING, IsReset: true},
	})
	require.NoError(t, err)
	require.Len(t, result.DAGRows, 1)
	f.assertDagStatus(t, ctx, pool, "RUNNING")
}
