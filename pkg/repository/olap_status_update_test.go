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
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/config/limits"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/validator"
)

// createEnumAwarePool builds a pool that registers the v1_readable_status_olap
// enum (and its array type) on each connection, mirroring the AfterConnect
// hook in pkg/config/loader. The OLAP queries bind []V1ReadableStatusOlap
// parameters, which pgx cannot encode without the registered types.
func createEnumAwarePool(t *testing.T, basePool *pgxpool.Pool) *pgxpool.Pool {
	ctx := context.Background()

	config, err := pgxpool.ParseConfig(basePool.Config().ConnString())
	require.NoError(t, err)

	config.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		if _, err := conn.Exec(ctx, "SET TIME ZONE 'UTC'"); err != nil {
			return err
		}

		for _, typeName := range []string{"v1_readable_status_olap", "_v1_readable_status_olap"} {
			pgType, err := conn.LoadType(ctx, typeName)
			if err != nil {
				return err
			}

			conn.TypeMap().RegisterType(pgType)
		}

		return nil
	}

	pool, err := pgxpool.NewWithConfig(ctx, config)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	return pool
}

// createOLAPRepositoryWithPayloadStore builds an OLAP repository with a fully
// initialized sharedRepository (including the payload store), which the task
// and task-event write paths require.
func createOLAPRepositoryWithPayloadStore(t *testing.T, pool *pgxpool.Pool) *OLAPRepositoryImpl {
	logger := zerolog.Nop()

	shared, cleanupShared := newSharedRepository(
		pool,
		pool,
		validator.NewDefaultValidator(),
		&logger,
		PayloadStoreRepositoryOpts{},
		limits.LimitConfigFile{},
		false,
		time.Minute,
	)
	t.Cleanup(func() { _ = cleanupShared() })

	repo, ok := newOLAPRepository(
		shared,
		24*time.Hour,
		false,
		false,
		StatusUpdateBatchSizeLimits{Task: 1000, DAG: 1000},
	).(*OLAPRepositoryImpl)
	require.True(t, ok)

	return repo
}

type replayStatusFixture struct {
	tenantId   uuid.UUID
	taskId     int64
	insertedAt pgtype.Timestamptz
	externalId uuid.UUID
	workflowId uuid.UUID
	workerId   uuid.UUID
}

func seedReplayTask(t *testing.T, ctx context.Context, repo *OLAPRepositoryImpl, taskId int64) replayStatusFixture {
	f := replayStatusFixture{
		tenantId:   uuid.New(),
		taskId:     taskId,
		insertedAt: pgtype.Timestamptz{Time: time.Now().UTC().Truncate(time.Microsecond), Valid: true},
		externalId: uuid.New(),
		workflowId: uuid.New(),
		workerId:   uuid.New(),
	}

	createReplayTask(t, ctx, repo, f)

	return f
}

// createReplayTask writes the fixture task through CreateTasks. Calling it
// again for the same fixture simulates a redelivered CreateTasks message:
// the insert is ON CONFLICT DO NOTHING and ReconcileTaskStatusesFromEvents
// runs over the task's full event history.
func createReplayTask(t *testing.T, ctx context.Context, repo *OLAPRepositoryImpl, f replayStatusFixture) {
	t.Helper()

	task := &V1TaskWithPayload{
		V1Task: &sqlcv1.V1Task{
			ID:                 f.taskId,
			InsertedAt:         f.insertedAt,
			TenantID:           f.tenantId,
			Queue:              "default",
			ActionID:           "test:replay-status",
			StepID:             uuid.New(),
			WorkflowID:         f.workflowId,
			WorkflowVersionID:  uuid.New(),
			WorkflowRunID:      f.externalId,
			ScheduleTimeout:    "5m",
			StepTimeout:        pgtype.Text{String: "60s", Valid: true},
			Priority:           pgtype.Int4{Int32: 1, Valid: true},
			Sticky:             sqlcv1.V1StickyStrategyNONE,
			ExternalID:         f.externalId,
			DisplayName:        "replay-status-test",
			Input:              []byte(`{}`),
			AdditionalMetadata: []byte(`{}`),
		},
		Payload: []byte(`{}`),
	}

	_, locksNotAcquired, err := repo.CreateTasks(ctx, f.tenantId, []*V1TaskWithPayload{task})
	require.NoError(t, err)
	require.Empty(t, locksNotAcquired)
}

func (f replayStatusFixture) event(eventType sqlcv1.V1EventTypeOlap, status sqlcv1.V1ReadableStatusOlap, retryCount int32) sqlcv1.CreateTaskEventsOLAPParams {
	e := sqlcv1.CreateTaskEventsOLAPParams{
		TenantID:       f.tenantId,
		TaskID:         f.taskId,
		TaskInsertedAt: f.insertedAt,
		EventType:      eventType,
		WorkflowID:     f.workflowId,
		EventTimestamp: pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
		ReadableStatus: status,
		RetryCount:     retryCount,
		Output:         []byte(`{}`),
		ExternalID:     f.externalId,
	}

	if eventType == sqlcv1.V1EventTypeOlapASSIGNED {
		workerId := f.workerId
		e.WorkerID = &workerId
	}

	return e
}

func assertOLAPTaskStatus(t *testing.T, ctx context.Context, pool *pgxpool.Pool, f replayStatusFixture, wantStatus string, wantRetryCount int32) {
	t.Helper()

	var status string
	var retryCount int32

	err := pool.QueryRow(ctx, `
		SELECT readable_status::text, latest_retry_count
		FROM v1_tasks_olap
		WHERE tenant_id = $1 AND id = $2
	`, f.tenantId, f.taskId).Scan(&status, &retryCount)
	require.NoError(t, err)

	// Non-fatal so a failure after one batch still surfaces failures after
	// later batches (the bug has two observable modes).
	assert.Equal(t, wantStatus, status, "v1_tasks_olap.readable_status")
	assert.Equal(t, wantRetryCount, retryCount, "v1_tasks_olap.latest_retry_count")
}

func assertOLAPRunStatus(t *testing.T, ctx context.Context, pool *pgxpool.Pool, f replayStatusFixture, wantStatus string) {
	t.Helper()

	var status string

	err := pool.QueryRow(ctx, `
		SELECT readable_status::text
		FROM v1_runs_olap
		WHERE tenant_id = $1 AND external_id = $2
	`, f.tenantId, f.externalId).Scan(&status)
	require.NoError(t, err)

	assert.Equal(t, wantStatus, status, "v1_runs_olap.readable_status")
}

// replayEventBatches returns the three ingestion batches that reproduce the
// lost-update on replay of a completed task. The task fails at retry 0,
// succeeds at retry 1, and is then replayed by the user (retry 2). Because
// OLAP ingestion is asynchronous, the replay's completion events can arrive
// in a batch before its queued/assigned events.
func replayEventBatches(f replayStatusFixture) [3][]sqlcv1.CreateTaskEventsOLAPParams {
	initialLifecycle := []sqlcv1.CreateTaskEventsOLAPParams{
		f.event(sqlcv1.V1EventTypeOlapQUEUED, sqlcv1.V1ReadableStatusOlapQUEUED, 0),
		f.event(sqlcv1.V1EventTypeOlapASSIGNED, sqlcv1.V1ReadableStatusOlapRUNNING, 0),
		f.event(sqlcv1.V1EventTypeOlapSTARTED, sqlcv1.V1ReadableStatusOlapRUNNING, 0),
		f.event(sqlcv1.V1EventTypeOlapFAILED, sqlcv1.V1ReadableStatusOlapFAILED, 0),
		f.event(sqlcv1.V1EventTypeOlapRETRYING, sqlcv1.V1ReadableStatusOlapQUEUED, 1),
		f.event(sqlcv1.V1EventTypeOlapASSIGNED, sqlcv1.V1ReadableStatusOlapRUNNING, 1),
		f.event(sqlcv1.V1EventTypeOlapSTARTED, sqlcv1.V1ReadableStatusOlapRUNNING, 1),
		f.event(sqlcv1.V1EventTypeOlapFINISHED, sqlcv1.V1ReadableStatusOlapCOMPLETED, 1),
	}

	replayCompletion := []sqlcv1.CreateTaskEventsOLAPParams{
		f.event(sqlcv1.V1EventTypeOlapSENTTOWORKER, sqlcv1.V1ReadableStatusOlapRUNNING, 2),
		f.event(sqlcv1.V1EventTypeOlapSTARTED, sqlcv1.V1ReadableStatusOlapRUNNING, 2),
		f.event(sqlcv1.V1EventTypeOlapFINISHED, sqlcv1.V1ReadableStatusOlapCOMPLETED, 2),
	}

	replayQueueing := []sqlcv1.CreateTaskEventsOLAPParams{
		f.event(sqlcv1.V1EventTypeOlapRETRIEDBYUSER, sqlcv1.V1ReadableStatusOlapQUEUED, 2),
		f.event(sqlcv1.V1EventTypeOlapQUEUED, sqlcv1.V1ReadableStatusOlapQUEUED, 2),
		f.event(sqlcv1.V1EventTypeOlapASSIGNED, sqlcv1.V1ReadableStatusOlapRUNNING, 2),
	}

	return [3][]sqlcv1.CreateTaskEventsOLAPParams{initialLifecycle, replayCompletion, replayQueueing}
}

func runReplayStatusScenario(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	f replayStatusFixture,
	applyBatch func(t *testing.T, events []sqlcv1.CreateTaskEventsOLAPParams),
) {
	batches := replayEventBatches(f)

	applyBatch(t, batches[0])
	assertOLAPTaskStatus(t, ctx, pool, f, "COMPLETED", 1)

	// The replay's completion events arrive first. A newer retry count must
	// bump latest_retry_count even though the readable status is already
	// COMPLETED; otherwise the completion is swallowed.
	applyBatch(t, batches[1])
	assertOLAPTaskStatus(t, ctx, pool, f, "COMPLETED", 2)

	// The replay's queueing events arrive last. They are stale relative to
	// the already-ingested completion at the same retry count, so they must
	// not regress the status back to RUNNING.
	applyBatch(t, batches[2])
	assertOLAPTaskStatus(t, ctx, pool, f, "COMPLETED", 2)
	assertOLAPRunStatus(t, ctx, pool, f, "COMPLETED")
}

func TestPrepareDAGStatusUpdateBatchOnlyIncludesUpdatedTasks(t *testing.T) {
	repo := &OLAPRepositoryImpl{}
	tenantId := uuid.New()
	dagInsertedAt := pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true}

	batch := repo.prepareDAGStatusUpdateBatch([]*sqlcv1.UpdateTaskStatusesFromMQRow{
		{
			TenantID:      tenantId,
			DagID:         pgtype.Int8{Int64: 1, Valid: true},
			DagInsertedAt: dagInsertedAt,
			WasUpdated:    false,
		},
		{
			TenantID:      tenantId,
			DagID:         pgtype.Int8{Int64: 2, Valid: true},
			DagInsertedAt: dagInsertedAt,
			WasUpdated:    true,
		},
		{
			TenantID:      tenantId,
			DagID:         pgtype.Int8{Int64: 2, Valid: true},
			DagInsertedAt: dagInsertedAt,
			WasUpdated:    true,
		},
		{
			TenantID:   tenantId,
			DagID:      pgtype.Int8{},
			WasUpdated: true,
		},
	})

	assert.Equal(t, []uuid.UUID{tenantId}, batch.Tenantids)
	assert.Equal(t, []int64{2}, batch.Dagids)
	assert.Equal(t, []pgtype.Timestamptz{dagInsertedAt}, batch.Daginsertedats)
}

// TestOLAPStatusUpdate_ReplayOfCompletedTask is a regression test for runs
// permanently stuck in RUNNING (or with a stale retry count) after a bulk
// replay of completed tasks, when the replay's OLAP events are ingested
// out of order across batches.
func TestOLAPStatusUpdate_ReplayOfCompletedTask(t *testing.T) {
	basePool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	pool := createEnumAwarePool(t, basePool)
	repo := createOLAPRepositoryWithPayloadStore(t, pool)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// Creates the hash partitions of v1_task_events_olap_tmp and the date
	// partitions of the OLAP tables, which the write paths rely on.
	require.NoError(t, repo.UpdateTablePartitions(ctx))

	t.Run("mq_path", func(t *testing.T) {
		f := seedReplayTask(t, ctx, repo, 1)

		applyBatch := func(t *testing.T, events []sqlcv1.CreateTaskEventsOLAPParams) {
			t.Helper()

			eventExternalIdToWorkflowRunId := map[uuid.UUID]uuid.UUID{f.externalId: f.externalId}

			_, locksNotAcquired, err := repo.CreateTaskEvents(ctx, f.tenantId, events, eventExternalIdToWorkflowRunId)
			require.NoError(t, err)
			require.Empty(t, locksNotAcquired)
		}

		runReplayStatusScenario(t, ctx, pool, f, applyBatch)
	})

	t.Run("tmp_table_path", func(t *testing.T) {
		f := seedReplayTask(t, ctx, repo, 2)

		applyBatch := func(t *testing.T, events []sqlcv1.CreateTaskEventsOLAPParams) {
			t.Helper()

			for _, e := range events {
				_, err := pool.Exec(ctx, `
					INSERT INTO v1_task_events_olap_tmp (
						tenant_id, task_id, task_inserted_at, event_type, readable_status, retry_count, worker_id
					) VALUES ($1, $2, $3, $4::v1_event_type_olap, $5::v1_readable_status_olap, $6, $7)
				`, e.TenantID, e.TaskID, e.TaskInsertedAt, string(e.EventType), string(e.ReadableStatus), e.RetryCount, e.WorkerID)
				require.NoError(t, err)
			}

			_, _, err := repo.UpdateTaskStatuses(ctx, []uuid.UUID{f.tenantId})
			require.NoError(t, err)
		}

		runReplayStatusScenario(t, ctx, pool, f, applyBatch)
	})

	// ReconcileTaskStatusesFromEvents carries the same guard but aggregates
	// over the task's full event history rather than one batch, so the
	// stuck-RUNNING mode cannot occur — only the stale-retry-count mode can.
	t.Run("reconcile_path", func(t *testing.T) {
		f := seedReplayTask(t, ctx, repo, 3)
		batches := replayEventBatches(f)

		eventExternalIdToWorkflowRunId := map[uuid.UUID]uuid.UUID{f.externalId: f.externalId}
		_, locksNotAcquired, err := repo.CreateTaskEvents(ctx, f.tenantId, batches[0], eventExternalIdToWorkflowRunId)
		require.NoError(t, err)
		require.Empty(t, locksNotAcquired)
		assertOLAPTaskStatus(t, ctx, pool, f, "COMPLETED", 1)

		// Write the replay's events (retry 2) into the events table without
		// going through a status-update path, mimicking events whose inline
		// status update did not reach this row.
		for _, batch := range [][]sqlcv1.CreateTaskEventsOLAPParams{batches[1], batches[2]} {
			for _, e := range batch {
				_, err := pool.Exec(ctx, `
					INSERT INTO v1_task_events_olap (
						tenant_id, task_id, task_inserted_at, event_type, workflow_id, readable_status, retry_count, worker_id
					) VALUES ($1, $2, $3, $4::v1_event_type_olap, $5, $6::v1_readable_status_olap, $7, $8)
				`, e.TenantID, e.TaskID, e.TaskInsertedAt, string(e.EventType), e.WorkflowID, string(e.ReadableStatus), e.RetryCount, e.WorkerID)
				require.NoError(t, err)
			}
		}

		// A redelivered CreateTasks message reconciles the row against the
		// event history: retry 2's highest-priority status is COMPLETED, the
		// same readable status the row already has, so only latest_retry_count
		// needs to advance.
		createReplayTask(t, ctx, repo, f)
		assertOLAPTaskStatus(t, ctx, pool, f, "COMPLETED", 2)
		assertOLAPRunStatus(t, ctx, pool, f, "COMPLETED")
	})
}
