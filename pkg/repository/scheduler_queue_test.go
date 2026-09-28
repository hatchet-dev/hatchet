//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func createAssignmentRepositoryForTest(pool *pgxpool.Pool) *assignmentRepository {
	logger := zerolog.New(io.Discard)

	return newAssignmentRepository(&sharedRepository{
		pool:    pool,
		l:       &logger,
		queries: sqlcv1.New(),
	})
}

func createWorkerRepositoryForTest(pool *pgxpool.Pool) WorkerRepository {
	logger := zerolog.New(io.Discard)

	return newWorkerRepository(&sharedRepository{
		pool:    pool,
		l:       &logger,
		queries: sqlcv1.New(),
	})
}

func TestListAvailableSlotsCountsBatchesOnce(t *testing.T) {
	t.Parallel()

	pool, cleanup := setupPostgresWithMigration(t)
	t.Cleanup(cleanup)

	repo := createAssignmentRepositoryForTest(pool)

	ctx := context.Background()

	tenantID := uuid.New()
	workerID := uuid.New()
	batchID := uuid.New()

	now := time.Now().UTC()
	timeoutAt := now.Add(time.Hour)

	slug := "tenant-" + strings.ReplaceAll(tenantID.String(), "-", "")

	_, err := pool.Exec(ctx, `INSERT INTO "Tenant" ("id", "name", "slug") VALUES ($1, $2, $3)`, tenantID, "Test Tenant", slug)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `INSERT INTO "Worker" ("id", "tenantId", "name", "maxRuns", "isActive") VALUES ($1, $2, $3, $4, $5)`, workerID, tenantID, "test-worker", 5, true)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `
		INSERT INTO v1_task_runtime (task_id, task_inserted_at, retry_count, worker_id, tenant_id, timeout_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, int64(1), now, int32(0), workerID, tenantID, timeoutAt)
	require.NoError(t, err)

	for idx, taskID := range []int64{2, 3, 4} {
		_, err = pool.Exec(ctx, `
			INSERT INTO v1_task_runtime (task_id, task_inserted_at, retry_count, worker_id, batch_id, batch_size, batch_index, tenant_id, timeout_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		`, taskID, now, int32(0), workerID, batchID, int32(3), int32(idx), tenantID, timeoutAt)
		require.NoError(t, err)
	}

	results, err := repo.ListAvailableSlotsForWorkers(ctx, tenantID, sqlcv1.ListAvailableSlotsForWorkersParams{
		Tenantid:  tenantID,
		Workerids: []uuid.UUID{workerID},
		Slottype:  "default",
	})
	require.NoError(t, err)
	require.Len(t, results, 1)

	assert.Equal(t, int32(3), results[0].AvailableSlots)
}

func createSharedRepositoryForTest(pool *pgxpool.Pool) *sharedRepository {
	logger := zerolog.New(io.Discard)

	return &sharedRepository{
		pool:    pool,
		ddlPool: pool,
		l:       &logger,
		queries: sqlcv1.New(),
	}
}

// insertQueuedTaskForTest inserts a v1_task row (the insert trigger creates its queue
// item) and returns the queue item.
func insertQueuedTaskForTest(t *testing.T, ctx context.Context, pool *pgxpool.Pool, tenantID uuid.UUID, taskID int64, stepTimeout string) *sqlcv1.V1QueueItem {
	t.Helper()

	_, err := pool.Exec(ctx, `
		INSERT INTO v1_task (
			id, tenant_id, queue, action_id, step_id, step_readable_id, workflow_id,
			workflow_version_id, workflow_run_id, schedule_timeout, sticky, external_id,
			display_name, input, step_index, step_timeout, initial_state
		)
		OVERRIDING SYSTEM VALUE
		VALUES (
			$1, $2, 'q', 'a', gen_random_uuid(), 's', gen_random_uuid(),
			gen_random_uuid(), gen_random_uuid(), '5m', 'NONE', gen_random_uuid(),
			't', '{}', 0, $3, 'QUEUED'
		)`, taskID, tenantID, stepTimeout)
	require.NoError(t, err)

	rows, err := pool.Query(ctx, `SELECT * FROM v1_queue_item WHERE task_id = $1`, taskID)
	require.NoError(t, err)

	items, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[sqlcv1.V1QueueItem])
	require.NoError(t, err)
	require.Len(t, items, 1)

	return items[0]
}

// A task key can reach one flush twice when a queue item a scheduler read earlier was
// replaced by a restored one before the flush. The statement deletes by task key, so
// the second copy has nothing to assign; it must be reported failed rather than abort
// the whole flush.
func TestMarkQueueItemsProcessedDuplicateTaskKey(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	require.NoError(t, createTaskRepository(pool).UpdateTablePartitions(ctx))

	repo := createSharedRepositoryForTest(pool)
	tenantID := uuid.New()

	qi := insertQueuedTaskForTest(t, ctx, pool, tenantID, 900001, "1h")
	other := insertQueuedTaskForTest(t, ctx, pool, tenantID, 900002, "1h")

	first := &AssignedItem{WorkerId: uuid.New(), QueueItem: qi}
	second := &AssignedItem{WorkerId: uuid.New(), QueueItem: qi}
	unrelated := &AssignedItem{WorkerId: uuid.New(), QueueItem: other}

	tx, err := pool.Begin(ctx)
	require.NoError(t, err)
	defer tx.Rollback(ctx) // nolint: errcheck

	succeeded, failed, err := repo.markQueueItemsProcessed(ctx, tenantID, &AssignResults{
		Assigned: []*AssignedItem{first, second, unrelated},
	}, tx, false)
	require.NoError(t, err)
	require.NoError(t, tx.Commit(ctx))

	require.Len(t, succeeded, 2)
	require.Len(t, failed, 1)
	assert.Same(t, second, failed[0])
	assert.Contains(t, succeeded, first)
	assert.Contains(t, succeeded, unrelated)

	var workerID uuid.UUID
	require.NoError(t, pool.QueryRow(ctx, `SELECT worker_id FROM v1_task_runtime WHERE task_id = $1`, qi.TaskID).Scan(&workerID))
	assert.Equal(t, first.WorkerId, workerID)

	var queued int
	require.NoError(t, pool.QueryRow(ctx, `SELECT count(*) FROM v1_queue_item WHERE task_id IN ($1, $2)`, qi.TaskID, other.TaskID).Scan(&queued))
	assert.Equal(t, 0, queued)
}

// A scheduler can hold a queue item whose row another scheduler consumed before the task
// was evicted and restored. Because restoration creates a new queue item for the same task
// key, the flush must assign that item to the worker the scheduler already reserved.
func TestMarkQueueItemsProcessedAssignsRestoredQueueItem(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	require.NoError(t, createTaskRepository(pool).UpdateTablePartitions(ctx))

	repo := createSharedRepositoryForTest(pool)
	tenantID := uuid.New()

	stale := insertQueuedTaskForTest(t, ctx, pool, tenantID, 900001, "1h")

	// another scheduler assigned the item and the task was evicted from its worker
	_, err := pool.Exec(ctx, `DELETE FROM v1_queue_item WHERE id = $1`, stale.ID)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `
		INSERT INTO v1_task_runtime (task_id, task_inserted_at, retry_count, worker_id, tenant_id, timeout_at, evicted_at)
		VALUES ($1, $2, $3, NULL, $4, now() + interval '1 hour', now())
	`, stale.TaskID, stale.TaskInsertedAt, stale.RetryCount, tenantID)
	require.NoError(t, err)

	restored, err := repo.queries.RestoreEvictedTasks(ctx, pool, sqlcv1.RestoreEvictedTasksParams{
		Taskids:         []int64{stale.TaskID},
		Taskinsertedats: []pgtype.Timestamptz{stale.TaskInsertedAt},
		Retrycounts:     []int32{stale.RetryCount},
		Tenantid:        tenantID,
	})
	require.NoError(t, err)
	require.Len(t, restored, 1)
	require.True(t, restored[0].Queued)

	var restoredID int64
	require.NoError(t, pool.QueryRow(ctx, `SELECT id FROM v1_queue_item WHERE task_id = $1`, stale.TaskID).Scan(&restoredID))
	require.NotEqual(t, stale.ID, restoredID)

	worker := uuid.New()

	tx, err := pool.Begin(ctx)
	require.NoError(t, err)
	defer tx.Rollback(ctx) // nolint: errcheck

	succeeded, failed, err := repo.markQueueItemsProcessed(ctx, tenantID, &AssignResults{
		Assigned: []*AssignedItem{{WorkerId: worker, QueueItem: stale}},
	}, tx, false)
	require.NoError(t, err)
	require.NoError(t, tx.Commit(ctx))

	require.Len(t, succeeded, 1)
	require.Empty(t, failed)

	var (
		workerID uuid.UUID
		evicted  bool
	)
	require.NoError(t, pool.QueryRow(ctx, `SELECT worker_id, evicted_at IS NOT NULL FROM v1_task_runtime WHERE task_id = $1`, stale.TaskID).Scan(&workerID, &evicted))
	assert.Equal(t, worker, workerID)
	assert.False(t, evicted)

	var queued int
	require.NoError(t, pool.QueryRow(ctx, `SELECT count(*) FROM v1_queue_item WHERE task_id = $1`, stale.TaskID).Scan(&queued))
	assert.Equal(t, 0, queued)
}

// A step timeout the duration grammar raises on (only importable rows carry one, since
// registration validation rejects it) fails its own item and leaves the rest of the batch
// flushed, so one poisoned queue item cannot hold back every assignment it shares a flush
// with.
func TestMarkQueueItemsProcessedInvalidStepTimeoutFailsOnlyThatItem(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	require.NoError(t, createTaskRepository(pool).UpdateTablePartitions(ctx))

	repo := createSharedRepositoryForTest(pool)
	tenantID := uuid.New()

	invalid := insertQueuedTaskForTest(t, ctx, pool, tenantID, 900001, "1234567890123456h")
	valid := insertQueuedTaskForTest(t, ctx, pool, tenantID, 900002, "1h")

	invalidItem := &AssignedItem{WorkerId: uuid.New(), QueueItem: invalid}
	validItem := &AssignedItem{WorkerId: uuid.New(), QueueItem: valid}

	tx, err := pool.Begin(ctx)
	require.NoError(t, err)
	defer tx.Rollback(ctx) // nolint: errcheck

	succeeded, failed, err := repo.markQueueItemsProcessed(ctx, tenantID, &AssignResults{
		Assigned: []*AssignedItem{invalidItem, validItem},
	}, tx, false)
	require.NoError(t, err)
	require.NoError(t, tx.Commit(ctx))

	require.Len(t, succeeded, 1)
	require.Len(t, failed, 1)
	assert.Same(t, validItem, succeeded[0])
	assert.Same(t, invalidItem, failed[0])

	var workerID uuid.UUID
	require.NoError(t, pool.QueryRow(ctx, `SELECT worker_id FROM v1_task_runtime WHERE task_id = $1`, valid.TaskID).Scan(&workerID))
	assert.Equal(t, validItem.WorkerId, workerID)

	var runtimes, queued int
	require.NoError(t, pool.QueryRow(ctx, `SELECT count(*) FROM v1_task_runtime WHERE task_id = $1`, invalid.TaskID).Scan(&runtimes))
	assert.Equal(t, 0, runtimes)
	require.NoError(t, pool.QueryRow(ctx, `SELECT count(*) FROM v1_queue_item WHERE task_id = $1`, invalid.TaskID).Scan(&queued))
	assert.Equal(t, 1, queued)
}

// The flush locks the existing runtime row of every key before it deletes any queue item,
// so against a transaction that holds a runtime row lock (RestoreEvictedTasks, EvictTask,
// ReleaseTasks all lock runtime rows first) it waits without holding any queue item lock
// of its own, and it cannot deadlock with them.
func TestMarkQueueItemsProcessedLocksRuntimesBeforeDeletingQueueItems(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	require.NoError(t, createTaskRepository(pool).UpdateTablePartitions(ctx))

	repo := createSharedRepositoryForTest(pool)
	tenantID := uuid.New()

	evicted := insertQueuedTaskForTest(t, ctx, pool, tenantID, 900001, "1h")
	fresh := insertQueuedTaskForTest(t, ctx, pool, tenantID, 900002, "1h")

	_, err := pool.Exec(ctx, `
		INSERT INTO v1_task_runtime (task_id, task_inserted_at, retry_count, worker_id, tenant_id, timeout_at, evicted_at)
		VALUES ($1, $2, $3, NULL, $4, now() + interval '1 hour', now())
	`, evicted.TaskID, evicted.TaskInsertedAt, evicted.RetryCount, tenantID)
	require.NoError(t, err)

	// another transaction holds the runtime row lock of the first key
	holder, err := pool.Begin(ctx)
	require.NoError(t, err)
	defer holder.Rollback(ctx) // nolint: errcheck

	var holderPid int
	require.NoError(t, holder.QueryRow(ctx, `SELECT pg_backend_pid()`).Scan(&holderPid))

	_, err = holder.Exec(ctx, `SELECT 1 FROM v1_task_runtime WHERE task_id = $1 FOR UPDATE`, evicted.TaskID)
	require.NoError(t, err)

	evictedItem := &AssignedItem{WorkerId: uuid.New(), QueueItem: evicted}
	freshItem := &AssignedItem{WorkerId: uuid.New(), QueueItem: fresh}

	flushDone := make(chan error, 1)

	go func() {
		tx, err := pool.Begin(ctx)

		if err != nil {
			flushDone <- err
			return
		}

		defer tx.Rollback(ctx) // nolint: errcheck

		succeeded, failed, err := repo.markQueueItemsProcessed(ctx, tenantID, &AssignResults{
			Assigned: []*AssignedItem{evictedItem, freshItem},
		}, tx, false)

		if err != nil {
			flushDone <- err
			return
		}

		if len(succeeded) != 2 || len(failed) != 0 {
			flushDone <- fmt.Errorf("expected both items assigned, got %d succeeded and %d failed", len(succeeded), len(failed))
			return
		}

		flushDone <- tx.Commit(ctx)
	}()

	// the flush waits on the holder's row lock
	require.Eventually(t, func() bool {
		var blocked bool
		err := pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM pg_stat_activity WHERE $1 = ANY(pg_blocking_pids(pid)))`, holderPid).Scan(&blocked)
		return err == nil && blocked
	}, 10*time.Second, 20*time.Millisecond)

	select {
	case err := <-flushDone:
		t.Fatalf("flush finished while the runtime row lock was held: %v", err)
	default:
	}

	// while it waits, the flush has deleted nothing: the second key's queue item can
	// still be locked by a third transaction
	probe, err := pool.Begin(ctx)
	require.NoError(t, err)
	defer probe.Rollback(ctx) // nolint: errcheck

	var probedID int64
	require.NoError(t, probe.QueryRow(ctx, `SELECT id FROM v1_queue_item WHERE task_id = $1 FOR UPDATE NOWAIT`, fresh.TaskID).Scan(&probedID))
	assert.Equal(t, fresh.ID, probedID)
	require.NoError(t, probe.Rollback(ctx))

	require.NoError(t, holder.Rollback(ctx))

	select {
	case err := <-flushDone:
		require.NoError(t, err)
	case <-time.After(10 * time.Second):
		t.Fatal("flush did not finish after the runtime row lock was released")
	}

	var queued int
	require.NoError(t, pool.QueryRow(ctx, `SELECT count(*) FROM v1_queue_item WHERE task_id IN ($1, $2)`, evicted.TaskID, fresh.TaskID).Scan(&queued))
	assert.Equal(t, 0, queued)

	var workerID uuid.UUID
	require.NoError(t, pool.QueryRow(ctx, `SELECT worker_id FROM v1_task_runtime WHERE task_id = $1 AND evicted_at IS NULL`, evicted.TaskID).Scan(&workerID))
	assert.Equal(t, evictedItem.WorkerId, workerID)
	require.NoError(t, pool.QueryRow(ctx, `SELECT worker_id FROM v1_task_runtime WHERE task_id = $1`, fresh.TaskID).Scan(&workerID))
	assert.Equal(t, freshItem.WorkerId, workerID)
}
