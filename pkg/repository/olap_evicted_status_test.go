//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// evictedStatusFixture drives the OLAP status rollup the way the OLAP controller does: the task
// is written with CreateTasks and every monitoring event goes through CreateTaskEvents, which
// runs UpdateTaskStatusesFromMQ in the same transaction. Readable statuses mirror
// olapEventTypeToReadableStatus in the OLAP controller, and durable invocation counts follow
// the engine: the first assigned invocation is 1, DURABLE_EVICTED and DURABLE_RESTORING carry
// the evicted invocation's count, and the next assignment increments it.
type evictedStatusFixture struct {
	t    *testing.T
	ctx  context.Context
	repo *OLAPRepositoryImpl
	task *V1TaskWithPayload
	// base is the event time every event in a scenario is offset from.
	base time.Time
}

// olapEvent is one monitoring event: its type, the readable status the OLAP controller maps it
// to, its event_timestamp as an offset from the task's base time, and its durable invocation count.
type olapEvent struct {
	eventType  sqlcv1.V1EventTypeOlap
	status     sqlcv1.V1ReadableStatusOlap
	offset     time.Duration
	invocation int32
}

func started(offset time.Duration, invocation int32) olapEvent {
	return olapEvent{sqlcv1.V1EventTypeOlapSTARTED, sqlcv1.V1ReadableStatusOlapRUNNING, offset, invocation}
}

func assigned(offset time.Duration, invocation int32) olapEvent {
	return olapEvent{sqlcv1.V1EventTypeOlapASSIGNED, sqlcv1.V1ReadableStatusOlapRUNNING, offset, invocation}
}

func evicted(offset time.Duration, invocation int32) olapEvent {
	return olapEvent{sqlcv1.V1EventTypeOlapDURABLEEVICTED, sqlcv1.V1ReadableStatusOlapEVICTED, offset, invocation}
}

func restoring(offset time.Duration, invocation int32) olapEvent {
	return olapEvent{sqlcv1.V1EventTypeOlapDURABLERESTORING, sqlcv1.V1ReadableStatusOlapRUNNING, offset, invocation}
}

func newEvictedStatusFixture(t *testing.T, ctx context.Context, repo *OLAPRepositoryImpl, taskId int64) *evictedStatusFixture {
	t.Helper()

	// Postgres stores microseconds and the status queries join tasks and events on
	// inserted_at, so the Go side must not carry nanoseconds the database would drop.
	insertedAt := time.Now().UTC().Truncate(time.Microsecond)

	return &evictedStatusFixture{
		t:    t,
		ctx:  ctx,
		repo: repo,
		base: insertedAt,
		task: &V1TaskWithPayload{
			V1Task: &sqlcv1.V1Task{
				ID:                 taskId,
				InsertedAt:         sqlchelpers.TimestamptzFromTime(insertedAt),
				TenantID:           uuid.New(),
				Queue:              "default",
				ActionID:           "durable:sleep",
				StepID:             uuid.New(),
				WorkflowID:         uuid.New(),
				WorkflowVersionID:  uuid.New(),
				WorkflowRunID:      uuid.New(),
				ScheduleTimeout:    "5m",
				StepTimeout:        sqlchelpers.TextFromStr("60s"),
				Sticky:             sqlcv1.V1StickyStrategyNONE,
				ExternalID:         uuid.New(),
				DisplayName:        "durable-sleep",
				Input:              []byte(`{}`),
				AdditionalMetadata: []byte(`{}`),
				InitialState:       sqlcv1.V1TaskInitialStateQUEUED,
				IsDurable:          pgtype.Bool{Bool: true, Valid: true},
			},
			Payload: []byte(`{}`),
		},
	}
}

func (f *evictedStatusFixture) createTask() {
	f.t.Helper()

	_, locksNotAcquired, err := f.repo.CreateTasks(f.ctx, f.task.TenantID, []*V1TaskWithPayload{f.task})
	require.NoError(f.t, err)
	require.Empty(f.t, locksNotAcquired)
}

// ingest writes one batch of monitoring events, the way one OLAP consumer batch does, and
// returns the task rows whose status the rollup changed.
func (f *evictedStatusFixture) ingest(events ...olapEvent) []UpdateTaskStatusRow {
	f.t.Helper()

	return f.ingestWithExcluded(nil, events...)
}

// ingestWithExcluded is ingest with the events at the given indexes persisted to the event
// history but excluded from the status rollup, the way a retried attempt's failure is. It
// stands in for an event whose inline status update did not reach the row.
func (f *evictedStatusFixture) ingestWithExcluded(excluded map[int]struct{}, events ...olapEvent) []UpdateTaskStatusRow {
	f.t.Helper()

	params := make([]sqlcv1.CreateTaskEventsOLAPParams, 0, len(events))
	eventToRun := make(map[uuid.UUID]uuid.UUID, len(events))
	excludedIds := make(map[uuid.UUID]struct{}, len(excluded))

	for i, e := range events {
		eventId := uuid.New()
		eventToRun[eventId] = f.task.WorkflowRunID

		if _, ok := excluded[i]; ok {
			excludedIds[eventId] = struct{}{}
		}

		params = append(params, sqlcv1.CreateTaskEventsOLAPParams{
			TenantID:               f.task.TenantID,
			TaskID:                 f.task.ID,
			TaskInsertedAt:         f.task.InsertedAt,
			EventType:              e.eventType,
			WorkflowID:             f.task.WorkflowID,
			EventTimestamp:         sqlchelpers.TimestamptzFromTime(f.base.Add(e.offset)),
			ReadableStatus:         e.status,
			RetryCount:             f.task.RetryCount,
			DurableInvocationCount: e.invocation,
			ExternalID:             eventId,
		})
	}

	res, locksNotAcquired, err := f.repo.CreateTaskEvents(f.ctx, f.task.TenantID, params, eventToRun, excludedIds, nil, nil)
	require.NoError(f.t, err)
	require.Empty(f.t, locksNotAcquired)

	if res == nil {
		return nil
	}

	return res.TaskRows
}

func (f *evictedStatusFixture) requireStatus(want sqlcv1.V1ReadableStatusOlap) {
	f.t.Helper()

	row, err := f.repo.ReadTaskRun(f.ctx, f.task.ExternalID)
	require.NoError(f.t, err)
	require.Equal(f.t, want, row.ReadableStatus, "v1_tasks_olap.readable_status")
	require.Equal(f.t, f.task.RetryCount, row.LatestRetryCount, "v1_tasks_olap.latest_retry_count")
}

// TestOLAPStatusUpdate_EvictedTaskFollowsDurableLifecycle is a regression test for an evicted
// durable task shown as RUNNING with isEvicted=false, and for the reverse. A durable task's
// STARTED, DURABLE_EVICTED and DURABLE_RESTORING events reach OLAP through independent publish
// buffers (the dispatcher's action event path, the dispatcher's eviction handler and the task
// controller), so OLAP can ingest them in any order. The rollup must decide EVICTED from the
// durable lifecycle events and their invocation counts, not from ingestion order.
func TestOLAPStatusUpdate_EvictedTaskFollowsDurableLifecycle(t *testing.T) {
	basePool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	pool := createEnumAwarePool(t, basePool)
	repo := createOLAPRepositoryWithPayloadStore(t, pool)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// a fresh database has no partitions for today yet
	require.NoError(t, repo.UpdateTablePartitions(ctx))

	// The CI failure: the STARTED event from before the eviction is ingested after it.
	t.Run("stale_started_after_eviction", func(t *testing.T) {
		f := newEvictedStatusFixture(t, ctx, repo, 1)
		f.createTask()

		updated := f.ingest(evicted(60*time.Millisecond, 1))
		require.Len(t, updated, 1)
		f.requireStatus(sqlcv1.V1ReadableStatusOlapEVICTED)

		updated = f.ingest(started(0, 1))
		require.Empty(t, updated, "a STARTED event from before the eviction must not change the status")
		f.requireStatus(sqlcv1.V1ReadableStatusOlapEVICTED)

		// worker-side events stamped after the eviction carry no invocation count and are not a restore
		updated = f.ingest(olapEvent{sqlcv1.V1EventTypeOlapTIMEOUTREFRESHED, sqlcv1.V1ReadableStatusOlapRUNNING, 120 * time.Millisecond, 0})
		require.Empty(t, updated)
		f.requireStatus(sqlcv1.V1ReadableStatusOlapEVICTED)

		// the restore of the evicted invocation leaves EVICTED
		updated = f.ingest(restoring(5*time.Second, 1))
		require.Len(t, updated, 1)
		require.Equal(t, sqlcv1.V1ReadableStatusOlapRUNNING, updated[0].ReadableStatus)
		f.requireStatus(sqlcv1.V1ReadableStatusOlapRUNNING)

		// the restored invocation is assigned, starts and is evicted again
		f.ingest(assigned(5050*time.Millisecond, 2))
		f.ingest(started(5100*time.Millisecond, 2))
		f.requireStatus(sqlcv1.V1ReadableStatusOlapRUNNING)

		updated = f.ingest(evicted(5200*time.Millisecond, 2))
		require.Len(t, updated, 1)
		f.requireStatus(sqlcv1.V1ReadableStatusOlapEVICTED)

		// a straggling STARTED from the first invocation cannot clear the second eviction
		require.Empty(t, f.ingest(started(10*time.Millisecond, 1)))
		f.requireStatus(sqlcv1.V1ReadableStatusOlapEVICTED)

		// restore, run and complete
		f.ingest(restoring(10*time.Second, 2))
		f.requireStatus(sqlcv1.V1ReadableStatusOlapRUNNING)
		f.ingest(assigned(10050*time.Millisecond, 3), started(10100*time.Millisecond, 3))

		updated = f.ingest(olapEvent{sqlcv1.V1EventTypeOlapFINISHED, sqlcv1.V1ReadableStatusOlapCOMPLETED, 11 * time.Second, 3})
		require.Len(t, updated, 1)
		f.requireStatus(sqlcv1.V1ReadableStatusOlapCOMPLETED)
	})

	// When a task is evicted while all its durable entries are already satisfied, the dispatcher
	// requests the restore before it publishes DURABLE_EVICTED, so DURABLE_RESTORING can be
	// ingested first. The late eviction must not evict the task the controller has restored.
	t.Run("restore_ingested_before_eviction", func(t *testing.T) {
		f := newEvictedStatusFixture(t, ctx, repo, 2)
		f.createTask()

		f.ingest(started(0, 1))
		f.requireStatus(sqlcv1.V1ReadableStatusOlapRUNNING)

		require.Empty(t, f.ingest(restoring(70*time.Millisecond, 1)))
		require.Empty(t, f.ingest(evicted(60*time.Millisecond, 1)), "an eviction already followed by its restore must not apply")
		f.requireStatus(sqlcv1.V1ReadableStatusOlapRUNNING)

		require.Empty(t, f.ingest(assigned(90*time.Millisecond, 2), started(100*time.Millisecond, 2)))
		f.requireStatus(sqlcv1.V1ReadableStatusOlapRUNNING)

		// the restored invocation can still be evicted
		require.Len(t, f.ingest(evicted(5*time.Second, 2)), 1)
		f.requireStatus(sqlcv1.V1ReadableStatusOlapEVICTED)
	})

	// The same events can share one ingestion batch, which collapses to a single status per
	// task (EVICTED outranks RUNNING). The history still decides.
	t.Run("eviction_and_restore_in_one_batch", func(t *testing.T) {
		f := newEvictedStatusFixture(t, ctx, repo, 3)
		f.createTask()

		f.ingest(started(0, 1))
		require.Empty(t, f.ingest(evicted(60*time.Millisecond, 1), restoring(70*time.Millisecond, 1)))
		f.requireStatus(sqlcv1.V1ReadableStatusOlapRUNNING)

		// and the reverse: an evicted task whose restore shares a batch with a stale STARTED
		g := newEvictedStatusFixture(t, ctx, repo, 4)
		g.createTask()

		g.ingest(evicted(60*time.Millisecond, 1))
		g.requireStatus(sqlcv1.V1ReadableStatusOlapEVICTED)
		require.Len(t, g.ingest(started(0, 1), restoring(5*time.Second, 1)), 1)
		g.requireStatus(sqlcv1.V1ReadableStatusOlapRUNNING)
	})

	// A batch holding the whole evict and restore cycle collapses to EVICTED by priority. The
	// history must decide the resulting status, whatever the row held before the batch.
	t.Run("collapsed_batch_with_restore_evidence", func(t *testing.T) {
		cycle := []olapEvent{evicted(60*time.Millisecond, 1), restoring(70*time.Millisecond, 1), assigned(90*time.Millisecond, 2), started(100*time.Millisecond, 2)}

		queued := newEvictedStatusFixture(t, ctx, repo, 12)
		queued.createTask()
		require.Len(t, queued.ingest(cycle...), 1)
		queued.requireStatus(sqlcv1.V1ReadableStatusOlapRUNNING)

		// the row is already EVICTED and the eviction is redelivered inside the batch
		alreadyEvicted := newEvictedStatusFixture(t, ctx, repo, 13)
		alreadyEvicted.createTask()
		alreadyEvicted.ingest(evicted(60*time.Millisecond, 1))
		alreadyEvicted.requireStatus(sqlcv1.V1ReadableStatusOlapEVICTED)
		require.Len(t, alreadyEvicted.ingest(cycle...), 1)
		alreadyEvicted.requireStatus(sqlcv1.V1ReadableStatusOlapRUNNING)

		// a newer retry count always applies, but still with the status the history decides
		retried := newEvictedStatusFixture(t, ctx, repo, 14)
		retried.createTask()
		retried.ingest(started(0, 1))
		require.Len(t, retried.ingest(olapEvent{sqlcv1.V1EventTypeOlapFAILED, sqlcv1.V1ReadableStatusOlapFAILED, time.Second, 1}), 1)
		retried.requireStatus(sqlcv1.V1ReadableStatusOlapFAILED)
		retried.task.RetryCount = 1
		require.Len(t, retried.ingest(evicted(2*time.Second, 2), restoring(2100*time.Millisecond, 2), assigned(2200*time.Millisecond, 3), started(2300*time.Millisecond, 3)), 1)
		retried.requireStatus(sqlcv1.V1ReadableStatusOlapRUNNING)

		// a QUEUED event on an EVICTED row whose persisted history already says restored moves
		// it to RUNNING, not back to QUEUED
		lateQueued := newEvictedStatusFixture(t, ctx, repo, 15)
		lateQueued.createTask()
		lateQueued.ingest(evicted(60*time.Millisecond, 1))
		lateQueued.requireStatus(sqlcv1.V1ReadableStatusOlapEVICTED)
		require.Empty(t, lateQueued.ingestWithExcluded(map[int]struct{}{0: {}}, restoring(70*time.Millisecond, 1)))
		lateQueued.requireStatus(sqlcv1.V1ReadableStatusOlapEVICTED)
		require.Len(t, lateQueued.ingest(olapEvent{sqlcv1.V1EventTypeOlapQUEUED, sqlcv1.V1ReadableStatusOlapQUEUED, 80 * time.Millisecond, 0}), 1)
		lateQueued.requireStatus(sqlcv1.V1ReadableStatusOlapRUNNING)
	})

	// DURABLE_RESTORING is published best effort after the restore commits. If it never arrives,
	// the restored invocation's assignment carries a higher invocation count than the eviction,
	// which only happens after a restore, so it clears EVICTED on its own.
	t.Run("lost_restore_recovered_by_reassignment", func(t *testing.T) {
		f := newEvictedStatusFixture(t, ctx, repo, 5)
		f.createTask()

		f.ingest(started(0, 1))
		f.ingest(evicted(60*time.Millisecond, 1))
		f.requireStatus(sqlcv1.V1ReadableStatusOlapEVICTED)

		updated := f.ingest(assigned(5*time.Second, 2))
		require.Len(t, updated, 1)
		require.Equal(t, sqlcv1.V1ReadableStatusOlapRUNNING, updated[0].ReadableStatus)
		f.requireStatus(sqlcv1.V1ReadableStatusOlapRUNNING)
	})

	// A delayed or redelivered restore of an earlier invocation must not clear a newer eviction.
	t.Run("stale_restore_does_not_clear_newer_eviction", func(t *testing.T) {
		f := newEvictedStatusFixture(t, ctx, repo, 6)
		f.createTask()

		f.ingest(started(0, 1))
		f.ingest(evicted(60*time.Millisecond, 1))
		f.ingest(restoring(5*time.Second, 1))
		f.ingest(assigned(5050*time.Millisecond, 2), started(5100*time.Millisecond, 2))
		f.ingest(evicted(6*time.Second, 2))
		f.requireStatus(sqlcv1.V1ReadableStatusOlapEVICTED)

		require.Empty(t, f.ingest(restoring(5*time.Second, 1)), "a redelivered restore of invocation 1 must not clear the eviction of invocation 2")
		f.requireStatus(sqlcv1.V1ReadableStatusOlapEVICTED)

		require.Len(t, f.ingest(restoring(10*time.Second, 2)), 1)
		f.requireStatus(sqlcv1.V1ReadableStatusOlapRUNNING)
	})

	// Terminal outcomes outrank EVICTED by priority and must keep applying to an evicted task.
	t.Run("terminal_events_override_eviction", func(t *testing.T) {
		cancelled := newEvictedStatusFixture(t, ctx, repo, 7)
		cancelled.createTask()
		cancelled.ingest(started(0, 1))
		cancelled.ingest(evicted(50*time.Millisecond, 1))
		cancelled.requireStatus(sqlcv1.V1ReadableStatusOlapEVICTED)
		require.Len(t, cancelled.ingest(olapEvent{sqlcv1.V1EventTypeOlapCANCELLED, sqlcv1.V1ReadableStatusOlapCANCELLED, time.Second, 0}), 1)
		cancelled.requireStatus(sqlcv1.V1ReadableStatusOlapCANCELLED)

		failed := newEvictedStatusFixture(t, ctx, repo, 8)
		failed.createTask()
		failed.ingest(evicted(50*time.Millisecond, 1))
		failed.requireStatus(sqlcv1.V1ReadableStatusOlapEVICTED)
		require.Len(t, failed.ingest(olapEvent{sqlcv1.V1EventTypeOlapTIMEDOUT, sqlcv1.V1ReadableStatusOlapFAILED, time.Second, 0}), 1)
		failed.requireStatus(sqlcv1.V1ReadableStatusOlapFAILED)

		// an eviction event never moves a terminal task back
		require.Empty(t, failed.ingest(evicted(2*time.Second, 1)))
		failed.requireStatus(sqlcv1.V1ReadableStatusOlapFAILED)
	})

	// ReconcileTaskStatusesFromEvents derives the status from the whole event history when the
	// task row is written after its events, and again on a redelivered created-task message. It
	// must follow the same lifecycle rule instead of letting the eviction win by priority.
	t.Run("reconcile_from_event_history", func(t *testing.T) {
		// events land before the task row exists, so the rollup has nothing to update yet
		f := newEvictedStatusFixture(t, ctx, repo, 9)
		require.Empty(t, f.ingest(evicted(60*time.Millisecond, 1)))
		require.Empty(t, f.ingest(started(0, 1)))

		f.createTask()
		f.requireStatus(sqlcv1.V1ReadableStatusOlapEVICTED)

		// a late RUNNING event of the evicted invocation still cannot clear it
		require.Empty(t, f.ingest(olapEvent{sqlcv1.V1EventTypeOlapSENTTOWORKER, sqlcv1.V1ReadableStatusOlapRUNNING, 5 * time.Millisecond, 1}))
		f.requireStatus(sqlcv1.V1ReadableStatusOlapEVICTED)

		require.Len(t, f.ingest(restoring(5*time.Second, 1)), 1)
		f.requireStatus(sqlcv1.V1ReadableStatusOlapRUNNING)

		// a redelivered created-task message after an evict and restore cycle must not re-evict
		g := newEvictedStatusFixture(t, ctx, repo, 10)
		g.createTask()
		g.ingest(started(0, 1))
		g.ingest(evicted(60*time.Millisecond, 1))
		g.ingest(restoring(5*time.Second, 1))
		g.ingest(assigned(5050*time.Millisecond, 2), started(5100*time.Millisecond, 2))
		g.requireStatus(sqlcv1.V1ReadableStatusOlapRUNNING)

		g.createTask()
		g.requireStatus(sqlcv1.V1ReadableStatusOlapRUNNING)

		// a task whose whole evict and restore history precedes its row is running
		h := newEvictedStatusFixture(t, ctx, repo, 11)
		require.Empty(t, h.ingest(evicted(60*time.Millisecond, 1), restoring(70*time.Millisecond, 1)))
		h.createTask()
		h.requireStatus(sqlcv1.V1ReadableStatusOlapRUNNING)
	})
}
