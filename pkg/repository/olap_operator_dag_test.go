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

func seedOperatorDag(t *testing.T, ctx context.Context, repo *OLAPRepositoryImpl, dagId int64) operatorDagFixture {
	t.Helper()

	return seedOperatorDagWithExternalId(t, ctx, repo, dagId, uuid.New())
}

func seedOperatorDagWithExternalId(t *testing.T, ctx context.Context, repo *OLAPRepositoryImpl, dagId int64, dagExternalId uuid.UUID) operatorDagFixture {
	t.Helper()

	f := operatorDagFixture{
		tenantId:      uuid.New(),
		dagId:         dagId,
		dagInsertedAt: pgtype.Timestamptz{Time: time.Now().UTC().Truncate(time.Microsecond), Valid: true},
		dagExternalId: dagExternalId,
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
		Payload:       []byte(`{}`),
		IsOperatorRun: true,
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

	_, locksNotAcquired, err := repo.CreateTaskEvents(ctx, f.tenantId, events, eventExternalIdToWorkflowRunId, nil, f.operatorRunIds())
	require.NoError(t, err)
	require.Empty(t, locksNotAcquired)
}

func (f operatorDagFixture) operatorRunIds() map[uuid.UUID]struct{} {
	return map[uuid.UUID]struct{}{f.dagExternalId: {}}
}

func (f operatorDagFixture) orchestratorUpdate(status sqlcv1.V1ReadableStatusOlap, retryCount int32) OrchestratorDAGStatusUpdateOpt {
	return OrchestratorDAGStatusUpdateOpt{
		DagId:          f.dagId,
		DagInsertedAt:  f.dagInsertedAt,
		ReadableStatus: status,
		RetryCount:     retryCount,
	}
}

func (f operatorDagFixture) applyOrchestratorEvents(t *testing.T, ctx context.Context, repo *OLAPRepositoryImpl, updates ...OrchestratorDAGStatusUpdateOpt) *StatusUpdateResult {
	t.Helper()

	result, locksNotAcquired, err := repo.CreateTaskEvents(ctx, f.tenantId, nil, nil, updates, f.operatorRunIds())
	require.NoError(t, err)
	require.Empty(t, locksNotAcquired)

	return result
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

func TestOperatorDAG_StatusComesFromOrchestratorNotChildren(t *testing.T) {
	basePool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	pool := createEnumAwarePool(t, basePool)
	repo := createOLAPRepositoryWithPayloadStore(t, pool)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	require.NoError(t, repo.UpdateTablePartitions(ctx))

	f := seedOperatorDag(t, ctx, repo, 100)

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

	var childRuns int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM v1_runs_olap
		WHERE tenant_id = $1 AND external_id IN ($2, $3)
	`, f.tenantId, childA.externalId, childB.externalId).Scan(&childRuns)
	require.NoError(t, err)
	assert.Equal(t, 0, childRuns, "children should not be v1_runs_olap rows")

	f.applyOrchestratorEvents(t, ctx, repo, f.orchestratorUpdate(sqlcv1.V1ReadableStatusOlapRUNNING, 0))
	f.assertDagStatus(t, ctx, pool, "RUNNING")

	// every child completing must NOT complete the DAG: only the orchestrator can do that
	f.applyChildEvents(t, ctx, repo, []sqlcv1.CreateTaskEventsOLAPParams{
		childA.event(sqlcv1.V1EventTypeOlapSTARTED, sqlcv1.V1ReadableStatusOlapRUNNING, 0),
		childA.event(sqlcv1.V1EventTypeOlapFINISHED, sqlcv1.V1ReadableStatusOlapCOMPLETED, 0),
		childB.event(sqlcv1.V1EventTypeOlapSTARTED, sqlcv1.V1ReadableStatusOlapRUNNING, 0),
		childB.event(sqlcv1.V1EventTypeOlapFINISHED, sqlcv1.V1ReadableStatusOlapCOMPLETED, 0),
	})
	f.assertDagStatus(t, ctx, pool, "RUNNING")

	f.applyOrchestratorEvents(t, ctx, repo, f.orchestratorUpdate(sqlcv1.V1ReadableStatusOlapCOMPLETED, 0))
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

	f := seedOperatorDag(t, ctx, repo, 200)

	result := f.applyOrchestratorEvents(t, ctx, repo, f.orchestratorUpdate(sqlcv1.V1ReadableStatusOlapRUNNING, 0))
	require.Len(t, result.DAGRows, 1)
	f.assertDagStatus(t, ctx, pool, "RUNNING")

	child := f.createChild(t, ctx, repo, 201)
	f.applyChildEvents(t, ctx, repo, []sqlcv1.CreateTaskEventsOLAPParams{
		child.event(sqlcv1.V1EventTypeOlapSTARTED, sqlcv1.V1ReadableStatusOlapRUNNING, 0),
	})

	result = f.applyOrchestratorEvents(t, ctx, repo, f.orchestratorUpdate(sqlcv1.V1ReadableStatusOlapFAILED, 0))
	require.Len(t, result.DAGRows, 1)
	f.assertDagStatus(t, ctx, pool, "FAILED")

	// a late child event must not disturb the terminal status
	f.applyChildEvents(t, ctx, repo, []sqlcv1.CreateTaskEventsOLAPParams{
		child.event(sqlcv1.V1EventTypeOlapFINISHED, sqlcv1.V1ReadableStatusOlapCOMPLETED, 0),
	})
	f.assertDagStatus(t, ctx, pool, "FAILED")

	// a stale RUNNING from the same attempt must not revive a failed DAG
	result = f.applyOrchestratorEvents(t, ctx, repo, f.orchestratorUpdate(sqlcv1.V1ReadableStatusOlapRUNNING, 0))
	require.Empty(t, result.DAGRows)
	f.assertDagStatus(t, ctx, pool, "FAILED")

	// a retry is a strictly newer attempt, so it resets the DAG
	result = f.applyOrchestratorEvents(t, ctx, repo, f.orchestratorUpdate(sqlcv1.V1ReadableStatusOlapQUEUED, 1))
	require.Len(t, result.DAGRows, 1)
	f.assertDagStatus(t, ctx, pool, "QUEUED")

	f.applyOrchestratorEvents(t, ctx, repo, f.orchestratorUpdate(sqlcv1.V1ReadableStatusOlapRUNNING, 1))
	f.assertDagStatus(t, ctx, pool, "RUNNING")

	// a straggler from the previous attempt can no longer clobber the new one
	result = f.applyOrchestratorEvents(t, ctx, repo, f.orchestratorUpdate(sqlcv1.V1ReadableStatusOlapFAILED, 0))
	require.Empty(t, result.DAGRows)
	f.assertDagStatus(t, ctx, pool, "RUNNING")
}

// Keying admission on the attempt number is what makes delivery order irrelevant, which is what
// lets these updates run without an advisory lock on the workflow run.
func TestOperatorDAG_OrchestratorEventsAreOrderIndependent(t *testing.T) {
	basePool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	pool := createEnumAwarePool(t, basePool)
	repo := createOLAPRepositoryWithPayloadStore(t, pool)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	require.NoError(t, repo.UpdateTablePartitions(ctx))

	testCases := []struct {
		name  string
		dagId int64
		apply func(f operatorDagFixture) []OrchestratorDAGStatusUpdateOpt
	}{
		{
			name:  "reversed",
			dagId: 310,
			apply: func(f operatorDagFixture) []OrchestratorDAGStatusUpdateOpt {
				return []OrchestratorDAGStatusUpdateOpt{
					f.orchestratorUpdate(sqlcv1.V1ReadableStatusOlapRUNNING, 1),
					f.orchestratorUpdate(sqlcv1.V1ReadableStatusOlapQUEUED, 1),
					f.orchestratorUpdate(sqlcv1.V1ReadableStatusOlapFAILED, 0),
					f.orchestratorUpdate(sqlcv1.V1ReadableStatusOlapRUNNING, 0),
				}
			},
		},
		{
			name:  "interleaved with duplicates",
			dagId: 320,
			apply: func(f operatorDagFixture) []OrchestratorDAGStatusUpdateOpt {
				return []OrchestratorDAGStatusUpdateOpt{
					f.orchestratorUpdate(sqlcv1.V1ReadableStatusOlapFAILED, 0),
					f.orchestratorUpdate(sqlcv1.V1ReadableStatusOlapRUNNING, 1),
					f.orchestratorUpdate(sqlcv1.V1ReadableStatusOlapRUNNING, 0),
					f.orchestratorUpdate(sqlcv1.V1ReadableStatusOlapFAILED, 0),
					f.orchestratorUpdate(sqlcv1.V1ReadableStatusOlapQUEUED, 1),
					f.orchestratorUpdate(sqlcv1.V1ReadableStatusOlapRUNNING, 1),
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			f := seedOperatorDag(t, ctx, repo, tc.dagId)

			// one batch per update, so each is its own transaction like real delivery
			for _, update := range tc.apply(f) {
				f.applyOrchestratorEvents(t, ctx, repo, update)
			}

			f.assertDagStatus(t, ctx, pool, "RUNNING")
		})
	}
}

// Only one update per DAG reaches the query, so a batch must collapse to the highest update, not the last.
func TestOperatorDAG_HighestUpdateInBatchWins(t *testing.T) {
	basePool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	pool := createEnumAwarePool(t, basePool)
	repo := createOLAPRepositoryWithPayloadStore(t, pool)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	require.NoError(t, repo.UpdateTablePartitions(ctx))

	testCases := []struct {
		name    string
		want    string
		dagId   int64
		updates func(f operatorDagFixture) []OrchestratorDAGStatusUpdateOpt
	}{
		{
			name:  "terminal outcome is not discarded by a later same-attempt RUNNING",
			dagId: 330,
			want:  "FAILED",
			updates: func(f operatorDagFixture) []OrchestratorDAGStatusUpdateOpt {
				return []OrchestratorDAGStatusUpdateOpt{
					f.orchestratorUpdate(sqlcv1.V1ReadableStatusOlapFAILED, 0),
					f.orchestratorUpdate(sqlcv1.V1ReadableStatusOlapRUNNING, 0),
				}
			},
		},
		{
			name:  "newer attempt beats an earlier attempt's terminal outcome",
			dagId: 340,
			want:  "RUNNING",
			updates: func(f operatorDagFixture) []OrchestratorDAGStatusUpdateOpt {
				return []OrchestratorDAGStatusUpdateOpt{
					f.orchestratorUpdate(sqlcv1.V1ReadableStatusOlapRUNNING, 1),
					f.orchestratorUpdate(sqlcv1.V1ReadableStatusOlapFAILED, 0),
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			f := seedOperatorDag(t, ctx, repo, tc.dagId)

			f.applyOrchestratorEvents(t, ctx, repo, tc.updates(f)...)
			f.assertDagStatus(t, ctx, pool, tc.want)
		})
	}
}

func TestOperatorDAG_UpdatesIgnoreWorkflowRunAdvisoryLock(t *testing.T) {
	basePool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	pool := createEnumAwarePool(t, basePool)
	repo := createOLAPRepositoryWithPayloadStore(t, pool)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	require.NoError(t, repo.UpdateTablePartitions(ctx))

	// held for the whole test, so nothing below may depend on taking it
	dagExternalId := uuid.New()

	blocker, err := pool.Begin(ctx)
	require.NoError(t, err)
	defer blocker.Rollback(ctx) // nolint: errcheck

	_, err = blocker.Exec(ctx, "SELECT pg_advisory_xact_lock($1)", workflowRunAdvisoryInt(dagExternalId))
	require.NoError(t, err)

	f := seedOperatorDagWithExternalId(t, ctx, repo, 500, dagExternalId)
	f.assertDagStatus(t, ctx, pool, "QUEUED")

	child := f.createChild(t, ctx, repo, 501)

	f.applyChildEvents(t, ctx, repo, []sqlcv1.CreateTaskEventsOLAPParams{
		child.event(sqlcv1.V1EventTypeOlapSTARTED, sqlcv1.V1ReadableStatusOlapRUNNING, 0),
	})

	result := f.applyOrchestratorEvents(t, ctx, repo, f.orchestratorUpdate(sqlcv1.V1ReadableStatusOlapRUNNING, 0))
	require.Len(t, result.DAGRows, 1)
	f.assertDagStatus(t, ctx, pool, "RUNNING")

	result = f.applyOrchestratorEvents(t, ctx, repo, f.orchestratorUpdate(sqlcv1.V1ReadableStatusOlapFAILED, 0))
	require.Len(t, result.DAGRows, 1)
	f.assertDagStatus(t, ctx, pool, "FAILED")
}
