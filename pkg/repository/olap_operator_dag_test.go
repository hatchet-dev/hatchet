//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

type operatorDagFixture struct {
	tenantId          uuid.UUID
	dagId             int64
	dagInsertedAt     pgtype.Timestamptz
	dagExternalId     uuid.UUID
	workflowId        uuid.UUID
	workflowVersionId uuid.UUID
}

func seedOperatorDag(t *testing.T, ctx context.Context, repo *OLAPRepositoryImpl, dagId int64) operatorDagFixture {
	t.Helper()

	return seedOperatorDagWithExternalId(t, ctx, repo, dagId, uuid.New())
}

func newOperatorDagFixture(dagId int64, dagExternalId uuid.UUID) operatorDagFixture {
	return operatorDagFixture{
		tenantId:          uuid.New(),
		dagId:             dagId,
		dagInsertedAt:     pgtype.Timestamptz{Time: time.Now().UTC().Truncate(time.Microsecond), Valid: true},
		dagExternalId:     dagExternalId,
		workflowId:        uuid.New(),
		workflowVersionId: uuid.New(),
	}
}

// create writes the operator DAG's OLAP row (the 'created-dag' message path). Split from fixture
// construction so a test can land orchestrator events in v1_task_events_olap first, reproducing
// the ordering race where created-dag arrives last.
func (f operatorDagFixture) create(t *testing.T, ctx context.Context, repo *OLAPRepositoryImpl) {
	t.Helper()

	dag := &DAGWithData{
		V1Dag: &sqlcv1.V1Dag{
			ID:                f.dagId,
			InsertedAt:        f.dagInsertedAt,
			TenantID:          f.tenantId,
			ExternalID:        f.dagExternalId,
			DisplayName:       "operator-dag-test",
			WorkflowID:        f.workflowId,
			WorkflowVersionID: f.workflowVersionId,
		},
		Input:              []byte(`{}`),
		AdditionalMetadata: []byte(`{}`),
		IsOperatorRun:      true,
	}

	locksNotAcquired, err := repo.CreateDAGs(ctx, f.tenantId, []*DAGWithData{dag})
	require.NoError(t, err)
	require.Empty(t, locksNotAcquired)
}

// orchestratorEvent builds a monitoring event addressed to the orchestrator task, which for an
// operator DAG shares the DAG's id and inserted_at.
func (f operatorDagFixture) orchestratorEvent(eventType sqlcv1.V1EventTypeOlap, status sqlcv1.V1ReadableStatusOlap, retryCount int32) sqlcv1.CreateTaskEventsOLAPParams {
	return sqlcv1.CreateTaskEventsOLAPParams{
		TenantID:       f.tenantId,
		TaskID:         f.dagId,
		TaskInsertedAt: f.dagInsertedAt,
		EventType:      eventType,
		WorkflowID:     f.workflowId,
		EventTimestamp: pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
		ReadableStatus: status,
		RetryCount:     retryCount,
		Output:         []byte(`{}`),
		ExternalID:     uuid.New(),
	}
}

// applyOrchestratorMonitoringEvents writes raw orchestrator monitoring events to
// v1_task_events_olap without requiring the DAG row to exist yet.
func (f operatorDagFixture) applyOrchestratorMonitoringEvents(t *testing.T, ctx context.Context, repo *OLAPRepositoryImpl, events ...sqlcv1.CreateTaskEventsOLAPParams) {
	t.Helper()

	eventExternalIdToWorkflowRunId := make(map[uuid.UUID]uuid.UUID, len(events))
	for _, e := range events {
		eventExternalIdToWorkflowRunId[e.ExternalID] = f.dagExternalId
	}

	_, locksNotAcquired, err := repo.CreateTaskEvents(ctx, f.tenantId, events, eventExternalIdToWorkflowRunId, nil, f.operatorRunIds())
	require.NoError(t, err)
	require.Empty(t, locksNotAcquired)
}

func seedOperatorDagWithExternalId(t *testing.T, ctx context.Context, repo *OLAPRepositoryImpl, dagId int64, dagExternalId uuid.UUID) operatorDagFixture {
	t.Helper()

	f := newOperatorDagFixture(dagId, dagExternalId)
	f.create(t, ctx, repo)

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
		DagId:              f.dagId,
		DagInsertedAt:      f.dagInsertedAt,
		ReadableStatus:     status,
		RetryCount:         retryCount,
		ExternalId:         f.dagExternalId,
		DisplayName:        "orchestrator-task",
		WorkflowId:         f.workflowId,
		WorkflowVersionId:  f.workflowVersionId,
		AdditionalMetadata: []byte(`{}`),
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

// TestOperatorDAG_CatchesUpWhenOrchestratorEventsPrecedeDagRow reproduces the ordering race: the
// orchestrator's ASSIGNED/STARTED/FINISHED monitoring events land in OLAP before the independent
// 'created-dag' message. Without the catch-up, UpdateDAGStatusesFromOrchestratorEvents no-ops
// (no DAG row to join) and the DAG is stranded non-terminal. writeDAGBatch must reconcile the
// freshly-inserted row against the events already on record.
func TestOperatorDAG_CatchesUpWhenOrchestratorEventsPrecedeDagRow(t *testing.T) {
	basePool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	pool := createEnumAwarePool(t, basePool)
	repo := createOLAPRepositoryWithPayloadStore(t, pool)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	require.NoError(t, repo.UpdateTablePartitions(ctx))

	t.Run("terminal event before dag row", func(t *testing.T) {
		f := newOperatorDagFixture(600, uuid.New())

		// orchestrator lifecycle events arrive first, while there is no v1_dags_olap row
		f.applyOrchestratorMonitoringEvents(t, ctx, repo,
			f.orchestratorEvent(sqlcv1.V1EventTypeOlapASSIGNED, sqlcv1.V1ReadableStatusOlapRUNNING, 0),
			f.orchestratorEvent(sqlcv1.V1EventTypeOlapSTARTED, sqlcv1.V1ReadableStatusOlapRUNNING, 0),
			f.orchestratorEvent(sqlcv1.V1EventTypeOlapFINISHED, sqlcv1.V1ReadableStatusOlapCOMPLETED, 0),
		)

		// the created-dag message finally lands
		f.create(t, ctx, repo)

		f.assertDagStatus(t, ctx, pool, "COMPLETED")
	})

	t.Run("only running events before dag row", func(t *testing.T) {
		f := newOperatorDagFixture(601, uuid.New())

		f.applyOrchestratorMonitoringEvents(t, ctx, repo,
			f.orchestratorEvent(sqlcv1.V1EventTypeOlapASSIGNED, sqlcv1.V1ReadableStatusOlapRUNNING, 0),
			f.orchestratorEvent(sqlcv1.V1EventTypeOlapSTARTED, sqlcv1.V1ReadableStatusOlapRUNNING, 0),
		)

		f.create(t, ctx, repo)
		f.assertDagStatus(t, ctx, pool, "RUNNING")

		// the later FINISHED still applies through the normal path now that the row exists
		result := f.applyOrchestratorEvents(t, ctx, repo, f.orchestratorUpdate(sqlcv1.V1ReadableStatusOlapCOMPLETED, 0))
		require.Len(t, result.DAGRows, 1)
		f.assertDagStatus(t, ctx, pool, "COMPLETED")
	})

	t.Run("higher retry terminal event before dag row", func(t *testing.T) {
		f := newOperatorDagFixture(602, uuid.New())

		f.applyOrchestratorMonitoringEvents(t, ctx, repo,
			f.orchestratorEvent(sqlcv1.V1EventTypeOlapSTARTED, sqlcv1.V1ReadableStatusOlapRUNNING, 0),
			f.orchestratorEvent(sqlcv1.V1EventTypeOlapFAILED, sqlcv1.V1ReadableStatusOlapFAILED, 0),
			f.orchestratorEvent(sqlcv1.V1EventTypeOlapSTARTED, sqlcv1.V1ReadableStatusOlapRUNNING, 1),
			f.orchestratorEvent(sqlcv1.V1EventTypeOlapFINISHED, sqlcv1.V1ReadableStatusOlapCOMPLETED, 1),
		)

		f.create(t, ctx, repo)

		f.assertDagStatus(t, ctx, pool, "COMPLETED")

		var latestRetry int32
		require.NoError(t, pool.QueryRow(ctx, `
			SELECT latest_retry_count FROM v1_dags_olap WHERE tenant_id = $1 AND id = $2
		`, f.tenantId, f.dagId).Scan(&latestRetry))
		assert.Equal(t, int32(1), latestRetry)
	})

	t.Run("no orchestrator events yet is a no-op", func(t *testing.T) {
		f := newOperatorDagFixture(603, uuid.New())

		f.create(t, ctx, repo)

		f.assertDagStatus(t, ctx, pool, "QUEUED")
	})
}

func (f operatorDagFixture) createDagsParams() sqlcv1.CreateDAGsOLAPOverwriteParams {
	return sqlcv1.CreateDAGsOLAPOverwriteParams{
		Tenantids:             []uuid.UUID{f.tenantId},
		Ids:                   []int64{f.dagId},
		Insertedats:           []pgtype.Timestamptz{f.dagInsertedAt},
		Externalids:           []uuid.UUID{f.dagExternalId},
		Displaynames:          []string{"operator-dag-test"},
		Workflowids:           []uuid.UUID{f.workflowId},
		Workflowversionids:    []uuid.UUID{f.workflowVersionId},
		Additionalmetadatas:   [][]byte{[]byte(`{}`)},
		Parenttaskexternalids: []*uuid.UUID{nil},
		Totaltasks:            []int32{1},
		IdempotencyKeys:       []pgtype.Text{{}},
		IsDagOperators:        []bool{true},
	}
}

func (f operatorDagFixture) orchestratorUpdateParams(status sqlcv1.V1ReadableStatusOlap) sqlcv1.UpdateDAGStatusesFromOrchestratorEventsParams {
	return sqlcv1.UpdateDAGStatusesFromOrchestratorEventsParams{
		Tenantid:            f.tenantId,
		Dagids:              []int64{f.dagId},
		Daginsertedats:      []pgtype.Timestamptz{f.dagInsertedAt},
		Statuses:            []sqlcv1.V1ReadableStatusOlap{status},
		Retrycounts:         []int32{0},
		Externalids:         []uuid.UUID{f.dagExternalId},
		Displaynames:        []string{"orchestrator-task"},
		Workflowids:         []uuid.UUID{f.workflowId},
		Workflowversionids:  []uuid.UUID{f.workflowVersionId},
		Additionalmetadatas: [][]byte{[]byte(`{}`)},
	}
}

// Runs first in an open transaction, then second in its own transaction, and asserts that second
// only completes once first commits.
func runOverlappingTransactions(t *testing.T, ctx context.Context, pool *pgxpool.Pool, first, second func(pgx.Tx) error) {
	t.Helper()

	firstTx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	require.NoError(t, err)
	defer firstTx.Rollback(ctx) // nolint: errcheck

	require.NoError(t, first(firstTx))

	secondDone := make(chan error, 1)

	go func() {
		secondTx, err := pool.BeginTx(ctx, pgx.TxOptions{})
		if err != nil {
			secondDone <- err
			return
		}
		defer secondTx.Rollback(ctx) // nolint: errcheck

		if err := second(secondTx); err != nil {
			secondDone <- err
			return
		}

		secondDone <- secondTx.Commit(ctx)
	}()

	select {
	case err := <-secondDone:
		t.Fatalf("second transaction did not wait for the first to commit: %v", err)
	case <-time.After(500 * time.Millisecond):
	}

	require.NoError(t, firstTx.Commit(ctx))

	select {
	case err := <-secondDone:
		require.NoError(t, err)
	case <-time.After(10 * time.Second):
		t.Fatal("second transaction never finished after the first committed")
	}
}

// The create and the status update run in separate transactions on separate message queue
// consumers. Under READ COMMITTED an UPDATE finds no row while the create is uncommitted, and the
// create's reconcile finds no events while the update is uncommitted, so both orderings lost the
// terminal status. The upsert makes the second transaction wait on the first one's row.
func TestOperatorDAG_ConcurrentCreateAndStatusUpdateSerialize(t *testing.T) {
	basePool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	pool := createEnumAwarePool(t, basePool)
	repo := createOLAPRepositoryWithPayloadStore(t, pool)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	require.NoError(t, repo.UpdateTablePartitions(ctx))

	t.Run("create first", func(t *testing.T) {
		f := newOperatorDagFixture(800, uuid.New())

		runOverlappingTransactions(t, ctx, pool,
			func(tx pgx.Tx) error {
				return repo.queries.CreateDAGsOLAP(ctx, tx, f.createDagsParams())
			},
			func(tx pgx.Tx) error {
				_, err := repo.queries.UpdateDAGStatusesFromOrchestratorEvents(ctx, tx, f.orchestratorUpdateParams(sqlcv1.V1ReadableStatusOlapCOMPLETED))
				return err
			},
		)

		f.assertDagStatus(t, ctx, pool, "COMPLETED")
	})

	t.Run("update first", func(t *testing.T) {
		f := newOperatorDagFixture(801, uuid.New())

		runOverlappingTransactions(t, ctx, pool,
			func(tx pgx.Tx) error {
				_, err := repo.queries.UpdateDAGStatusesFromOrchestratorEvents(ctx, tx, f.orchestratorUpdateParams(sqlcv1.V1ReadableStatusOlapCOMPLETED))
				return err
			},
			func(tx pgx.Tx) error {
				return repo.queries.CreateDAGsOLAP(ctx, tx, f.createDagsParams())
			},
		)

		f.assertDagStatus(t, ctx, pool, "COMPLETED")
	})
}

// A status update that lands first inserts the DAG row, and the insert trigger copies it to
// v1_runs_olap without the parent task or idempotency key the update cannot know. The later create
// only takes the conflict path on v1_dags_olap, so the runs row depends on the update trigger to
// carry those columns over.
func TestOperatorDAG_RunsRowGetsParentFromLateCreate(t *testing.T) {
	basePool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	pool := createEnumAwarePool(t, basePool)
	repo := createOLAPRepositoryWithPayloadStore(t, pool)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	require.NoError(t, repo.UpdateTablePartitions(ctx))

	f := newOperatorDagFixture(900, uuid.New())
	parentTaskExternalId := uuid.New()

	f.applyOrchestratorEvents(t, ctx, repo, f.orchestratorUpdate(sqlcv1.V1ReadableStatusOlapCOMPLETED, 0))

	_, err := repo.CreateDAGs(ctx, f.tenantId, []*DAGWithData{{
		V1Dag: &sqlcv1.V1Dag{
			ID:                f.dagId,
			InsertedAt:        f.dagInsertedAt,
			TenantID:          f.tenantId,
			ExternalID:        f.dagExternalId,
			DisplayName:       "operator-dag-test",
			WorkflowID:        f.workflowId,
			WorkflowVersionID: f.workflowVersionId,
			IdempotencyKey:    pgtype.Text{String: "idempotency-key", Valid: true},
		},
		Input:                []byte(`{}`),
		AdditionalMetadata:   []byte(`{}`),
		ParentTaskExternalID: &parentTaskExternalId,
		IsOperatorRun:        true,
	}})
	require.NoError(t, err)

	var runParent *uuid.UUID
	var runIdempotencyKey *string

	require.NoError(t, pool.QueryRow(ctx, `
		SELECT parent_task_external_id, idempotency_key
		FROM v1_runs_olap
		WHERE tenant_id = $1 AND external_id = $2
	`, f.tenantId, f.dagExternalId).Scan(&runParent, &runIdempotencyKey))

	require.NotNil(t, runParent)
	assert.Equal(t, parentTaskExternalId, *runParent)
	require.NotNil(t, runIdempotencyKey)
	assert.Equal(t, "idempotency-key", *runIdempotencyKey)

	f.assertDagStatus(t, ctx, pool, "COMPLETED")
}
