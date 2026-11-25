//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// A worker's stale heartbeat only proves the worker is gone while the engine could record
// heartbeats. This walks a dispatcher through a database outage: reassignment stays paused while
// the dispatcher's own heartbeat is stale and for a minute after it resumes, then goes ahead.
func TestProcessTaskReassignmentsPausesWhileDispatchersCouldNotHeartbeat(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()

	repo := newBatchTestRepository(pool)
	repo.reassignLimit = 100

	dispatchers := newDispatcherRepository(repo.sharedRepository)

	// the migrated database has no dispatchers, so the one below is the only one
	dispatcherId := uuid.New()
	_, err := pool.Exec(ctx, `
		INSERT INTO "Dispatcher" ("id", "lastHeartbeatAt", "heartbeatingSince")
		VALUES ($1, NOW(), NOW() - INTERVAL '10 minutes')
	`, dispatcherId)
	require.NoError(t, err)

	task := insertTaskWithRetries(t, ctx, repo, "reassign-pause", 0)

	workerId := uuid.New()
	_, err = pool.Exec(ctx, `
		INSERT INTO "Worker" ("id", "tenantId", "name", "lastHeartbeatAt", "createdAt", "updatedAt")
		VALUES ($1, $2, 'reassign-pause-worker', NOW() - INTERVAL '45 seconds', NOW(), NOW())
	`, workerId, internalTenantId)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `
		INSERT INTO v1_task_runtime (task_id, task_inserted_at, retry_count, worker_id, tenant_id, timeout_at, assigned_at, acked_at)
		VALUES ($1, $2, $3, $4, $5, NOW() + INTERVAL '1 hour', NOW() - INTERVAL '50 seconds', NOW() - INTERVAL '50 seconds')
	`, task.ID, task.InsertedAt, task.RetryCount, workerId, internalTenantId)
	require.NoError(t, err)

	reassigned := func() int {
		t.Helper()

		res, _, err := repo.ProcessTaskReassignments(ctx, internalTenantId)
		require.NoError(t, err)

		return len(res.ReleasedTasks)
	}

	// the outage: the dispatcher has not written a heartbeat for longer than the worker has
	_, err = pool.Exec(ctx, `UPDATE "Dispatcher" SET "lastHeartbeatAt" = NOW() - INTERVAL '45 seconds' WHERE "id" = $1`, dispatcherId)
	require.NoError(t, err)

	require.Zero(t, reassigned(), "no dispatcher is live, so the stale worker heartbeat proves nothing")

	// the database is back and the dispatcher heartbeats before the worker does
	require.NoError(t, dispatchers.Heartbeat(ctx, dispatcherId))

	require.Zero(t, reassigned(), "the dispatcher has only just resumed, so the worker has not had its chance to heartbeat")

	// a heartbeat that follows no gap leaves the run alone
	_, err = pool.Exec(ctx, `UPDATE "Dispatcher" SET "heartbeatingSince" = NOW() - INTERVAL '61 seconds' WHERE "id" = $1`, dispatcherId)
	require.NoError(t, err)
	require.NoError(t, dispatchers.Heartbeat(ctx, dispatcherId))

	require.Equal(t, 1, reassigned(), "the dispatcher recorded heartbeats for a full minute and the worker sent none")
}

// A draining dispatcher keeps heartbeating but takes no new worker streams, so it does not vouch
// for worker heartbeats.
func TestHasDispatcherHeartbeatingThroughoutIgnoresInactiveDispatchers(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()

	repo := newBatchTestRepository(pool)

	dispatcherId := uuid.New()
	_, err := pool.Exec(ctx, `
		INSERT INTO "Dispatcher" ("id", "lastHeartbeatAt", "heartbeatingSince", "isActive")
		VALUES ($1, NOW(), NOW() - INTERVAL '10 minutes', false)
	`, dispatcherId)
	require.NoError(t, err)

	recording, err := repo.queries.HasDispatcherHeartbeatingThroughout(ctx, pool)
	require.NoError(t, err)
	require.False(t, recording)

	_, err = pool.Exec(ctx, `UPDATE "Dispatcher" SET "isActive" = true WHERE "id" = $1`, dispatcherId)
	require.NoError(t, err)

	recording, err = repo.queries.HasDispatcherHeartbeatingThroughout(ctx, pool)
	require.NoError(t, err)
	require.True(t, recording)
}

// A stalled database can accept heartbeat statements without finishing them. A heartbeat that
// queued behind the stalled one started within seconds of it, but nothing was recorded until the
// stall ended, so it starts a new run.
func TestDispatcherHeartbeatCountsTimeSpentWaitingAsAGap(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()

	repo := newBatchTestRepository(pool)
	dispatchers := newDispatcherRepository(repo.sharedRepository)

	dispatcherId := uuid.New()
	_, err := pool.Exec(ctx, `
		INSERT INTO "Dispatcher" ("id", "lastHeartbeatAt", "heartbeatingSince")
		VALUES ($1, NOW(), NOW() - INTERVAL '10 minutes')
	`, dispatcherId)
	require.NoError(t, err)

	// the stalled heartbeat: written 14 seconds ago and still not committed
	stalled, err := pool.Begin(ctx)
	require.NoError(t, err)
	defer stalled.Rollback(ctx) // nolint:errcheck

	_, err = stalled.Exec(ctx, `UPDATE "Dispatcher" SET "lastHeartbeatAt" = NOW() - INTERVAL '14 seconds' WHERE "id" = $1`, dispatcherId)
	require.NoError(t, err)

	queued := make(chan error, 1)

	go func() {
		queued <- dispatchers.Heartbeat(ctx, dispatcherId)
	}()

	require.Eventually(t, func() bool {
		var waiting bool

		err := pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM pg_stat_activity
				WHERE wait_event_type = 'Lock' AND query LIKE '%heartbeatingSince%'
			)
		`).Scan(&waiting)

		return err == nil && waiting
	}, 10*time.Second, 50*time.Millisecond, "the queued heartbeat should wait on the row the stalled one holds")

	// the stall ends 16 seconds after the stalled heartbeat was written
	time.Sleep(2 * time.Second)

	require.NoError(t, stalled.Commit(ctx))
	require.NoError(t, <-queued)

	var restarted bool
	err = pool.QueryRow(ctx, `SELECT "heartbeatingSince" > NOW() - INTERVAL '1 minute' FROM "Dispatcher" WHERE "id" = $1`, dispatcherId).Scan(&restarted)
	require.NoError(t, err)
	require.True(t, restarted, "no heartbeat was recorded for 16 seconds, so the run starts over")
}
