//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
	"maps"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hatchet-dev/pgoutbox"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func seedDeliveryWorker(t *testing.T, ctx context.Context, pool *pgxpool.Pool, tenantId uuid.UUID) uuid.UUID {
	t.Helper()

	workerId := uuid.New()

	_, err := pool.Exec(ctx, `
		INSERT INTO "Worker" ("id", "tenantId", "name", "lastHeartbeatAt", "createdAt", "updatedAt")
		VALUES ($1, $2, 'delivery-worker', NOW(), NOW(), NOW())
	`, workerId, tenantId)
	require.NoError(t, err)

	return workerId
}

type runtimeSeed struct {
	taskId     int64
	workerId   uuid.UUID
	assignedAt time.Time
	acked      bool
	evicted    bool
	batched    bool
}

func seedRuntime(t *testing.T, ctx context.Context, pool *pgxpool.Pool, tenantId uuid.UUID, r runtimeSeed) {
	t.Helper()

	var ackedAt, evictedAt *time.Time
	var batchId *uuid.UUID

	if r.acked {
		ackedAt = &r.assignedAt
	}

	if r.evicted {
		evictedAt = &r.assignedAt
	}

	if r.batched {
		id := uuid.New()
		batchId = &id
	}

	_, err := pool.Exec(ctx, `
		INSERT INTO v1_task_runtime (task_id, task_inserted_at, retry_count, worker_id, tenant_id, timeout_at, assigned_at, acked_at, evicted_at, batch_id)
		VALUES ($1, NOW(), 0, $2, $3, NOW() + INTERVAL '1 hour', $4, $5, $6, $7)
	`, r.taskId, r.workerId, tenantId, r.assignedAt, ackedAt, evictedAt, batchId)
	require.NoError(t, err)
}

// The pull resends what it returns, so anything a worker already has, can't be rebuilt as a single
// start, or belongs to another dispatcher's workers must stay out.
func TestListUnackedAssignmentsReturnsOnlyWhatThePullMaySend(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	t.Cleanup(func() { cleanup() })

	ctx := context.Background()
	tenantId := uuid.New()

	_, err := pool.Exec(ctx, `INSERT INTO "Tenant" ("id", "name", "slug", "createdAt", "updatedAt") VALUES ($1, 'delivery', $2, NOW(), NOW())`, tenantId, "delivery-"+tenantId.String())
	require.NoError(t, err)

	mine := seedDeliveryWorker(t, ctx, pool, tenantId)
	other := seedDeliveryWorker(t, ctx, pool, tenantId)

	now := time.Now().UTC().Truncate(time.Microsecond)
	after, before := now.Add(-time.Minute), now.Add(-2*time.Second)
	inWindow := now.Add(-10 * time.Second)

	for _, r := range []runtimeSeed{
		{taskId: 1, workerId: mine, assignedAt: inWindow},
		{taskId: 2, workerId: mine, assignedAt: inWindow, acked: true},
		{taskId: 3, workerId: mine, assignedAt: inWindow, evicted: true},
		{taskId: 4, workerId: mine, assignedAt: inWindow, batched: true},
		{taskId: 5, workerId: mine, assignedAt: after.Add(-time.Second)},
		{taskId: 6, workerId: mine, assignedAt: now},
		{taskId: 7, workerId: other, assignedAt: inWindow},
		{taskId: 8, workerId: mine, assignedAt: after},
		{taskId: 9, workerId: mine, assignedAt: before},
	} {
		seedRuntime(t, ctx, pool, tenantId, r)
	}

	repo := newDispatcherRepository(createSharedRepositoryForTest(pool))

	rows, err := repo.ListUnackedAssignments(ctx, []uuid.UUID{mine}, after, before, 10)
	require.NoError(t, err)

	got := map[int64]time.Time{}

	for _, r := range rows {
		assert.Equal(t, tenantId, r.TenantID)
		assert.Equal(t, mine, r.WorkerID)
		got[r.TaskID] = r.AssignedAt
	}

	// the window is [after, before): a dispatcher's start is included, the grace boundary is not
	require.ElementsMatch(t, []int64{1, 8}, slices.Collect(maps.Keys(got)))
	assert.True(t, inWindow.Equal(got[1]), "the pull dedupes on the exact assigned_at")
}

func TestListUnackedAssignmentsLimitsEachWorkerOldestFirst(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	t.Cleanup(func() { cleanup() })

	ctx := context.Background()
	tenantId := uuid.New()

	_, err := pool.Exec(ctx, `INSERT INTO "Tenant" ("id", "name", "slug", "createdAt", "updatedAt") VALUES ($1, 'delivery', $2, NOW(), NOW())`, tenantId, "delivery-"+tenantId.String())
	require.NoError(t, err)

	a := seedDeliveryWorker(t, ctx, pool, tenantId)
	b := seedDeliveryWorker(t, ctx, pool, tenantId)
	now := time.Now().UTC()

	// newest first, so a missing ORDER BY would return the newest rows in insertion order
	for i := 2; i >= 0; i-- {
		seedRuntime(t, ctx, pool, tenantId, runtimeSeed{taskId: int64(10 + i), workerId: a, assignedAt: now.Add(-time.Duration(30-i) * time.Second)})
		seedRuntime(t, ctx, pool, tenantId, runtimeSeed{taskId: int64(20 + i), workerId: b, assignedAt: now.Add(-time.Duration(30-i) * time.Second)})
	}

	rows, err := newDispatcherRepository(createSharedRepositoryForTest(pool)).ListUnackedAssignments(ctx, []uuid.UUID{a, b}, now.Add(-time.Minute), now, 2)
	require.NoError(t, err)

	got := map[uuid.UUID][]int64{}

	for _, r := range rows {
		got[r.WorkerID] = append(got[r.WorkerID], r.TaskID)
	}

	assert.ElementsMatch(t, []int64{10, 11}, got[a])
	assert.ElementsMatch(t, []int64{20, 21}, got[b])
}

// The outbox only makes the next DAG step crash-safe if its events commit with the completion, and the
// caller must then not send them again over the task queue.
func TestCompleteTasksWritesInternalEventsToTheOutbox(t *testing.T) {
	for _, writes := range []bool{true, false} {
		t.Run(map[bool]string{true: "writes on", false: "writes off"}[writes], func(t *testing.T) {
			f := newEventOrderFixture(t)

			ob, err := pgoutbox.NewOutbox(f.ctx, f.pool, pgoutbox.WithAutoMigrate(false))
			require.NoError(t, err)

			l := zerolog.Nop()
			f.repo.internalEventOutbox = NewInternalEventOutbox(ob, writes, time.Second, &l)

			task := f.trigger()
			f.assign(task, 0)

			res, err := f.repo.CompleteTasks(f.ctx, internalTenantId, []CompleteTaskOpts{{TaskIdInsertedAtRetryCount: f.key(task, 0), Output: []byte(`{}`)}})
			require.NoError(t, err)
			require.NotEmpty(t, res.InternalEvents)

			// rows written by one transaction share its xmin, which pins the outbox write to the
			// completion's transaction rather than just to a later commit
			var queued int
			require.NoError(t, f.pool.QueryRow(f.ctx, `
				SELECT count(*)
				FROM outbox.messages m
				WHERE m.topic = $1
					AND m.xmin = (SELECT e.xmin FROM v1_task_event e WHERE e.task_id = $2 AND e.task_inserted_at = $3 AND e.event_type = 'COMPLETED')`,
				InternalEventOutboxTopic(f.repo.internalEventOutbox.shardFor(internalTenantId)), task.id, task.insertedAt).Scan(&queued))

			var total int
			require.NoError(t, f.pool.QueryRow(f.ctx, `SELECT count(*) FROM outbox.messages`).Scan(&total))

			assert.Equal(t, writes, res.InternalEventsOutboxed)
			assert.Equal(t, map[bool]int{true: 1, false: 0}[writes], queued, "outbox rows written with the completion")
			assert.Equal(t, queued, total, "outbox rows written outside the completion")
		})
	}
}
