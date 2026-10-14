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

	"github.com/hatchet-dev/hatchet/pkg/repository/cache"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func readPausedUntil(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID, key string) time.Time {
	t.Helper()

	var pausedUntil time.Time
	err := pool.QueryRow(context.Background(), `SELECT paused_until FROM v1_rate_limit_pause WHERE tenant_id = $1 AND key = $2`, tenantID, key).Scan(&pausedUntil)
	require.NoError(t, err)

	return pausedUntil
}

// A later pause call with an earlier deadline must not shorten an active pause, so concurrent tasks hitting the
// same upstream can each pause the key with their own Retry-After without racing.
func TestPauseRateLimitKeepsLongestPause(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	tenantID := createRateLimitTestTenant(t, pool)
	rateLimitRepo := newRateLimitRepository(createRateLimitTestShared(pool))

	const key = "upstream-api:tenant-123"

	now := time.Now().UTC().Truncate(time.Millisecond)

	pause, err := rateLimitRepo.PauseRateLimit(ctx, tenantID, key, now.Add(time.Hour))
	require.NoError(t, err)
	assert.WithinDuration(t, now.Add(time.Hour), pause.PausedUntil.Time, time.Millisecond)

	pause, err = rateLimitRepo.PauseRateLimit(ctx, tenantID, key, now.Add(10*time.Minute))
	require.NoError(t, err)
	assert.WithinDuration(t, now.Add(time.Hour), pause.PausedUntil.Time, time.Millisecond, "an earlier deadline must not shorten the pause")
	assert.WithinDuration(t, now.Add(time.Hour), readPausedUntil(t, pool, tenantID, key), time.Millisecond)

	pause, err = rateLimitRepo.PauseRateLimit(ctx, tenantID, key, now.Add(2*time.Hour))
	require.NoError(t, err)
	assert.WithinDuration(t, now.Add(2*time.Hour), pause.PausedUntil.Time, time.Millisecond)

	_, err = rateLimitRepo.PauseRateLimit(ctx, tenantID, "", now.Add(time.Hour))
	assert.Error(t, err)
}

// The scheduler only learns about pauses through FlushRateLimits, and an expired pause must read as unpaused
// even though its row is still there.
func TestFlushRateLimitsReturnsActivePauses(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	tenantID := createRateLimitTestTenant(t, pool)
	rateLimitRepo := newRateLimitRepository(createRateLimitTestShared(pool))

	const pausedKey, expiredKey, openKey, refilledKey = "rl-paused", "rl-expired", "rl-open", "rl-refilled"

	insertRateLimit(t, pool, tenantID, pausedKey, 10, 10, "1 HOUR", 0)
	insertRateLimit(t, pool, tenantID, expiredKey, 10, 10, "1 HOUR", 0)
	insertRateLimit(t, pool, tenantID, openKey, 10, 10, "1 HOUR", 0)

	// covers the refill branch of the query, which returns rows from a separate CTE
	insertRateLimit(t, pool, tenantID, refilledKey, 10, 0, "1 second", 2*time.Second)

	pausedUntil := time.Now().UTC().Add(time.Hour).Truncate(time.Millisecond)

	_, err := rateLimitRepo.PauseRateLimit(ctx, tenantID, pausedKey, pausedUntil)
	require.NoError(t, err)

	_, err = rateLimitRepo.PauseRateLimit(ctx, tenantID, refilledKey, pausedUntil)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `INSERT INTO v1_rate_limit_pause (tenant_id, key, paused_until) VALUES ($1, $2, NOW() - INTERVAL '1 minute')`, tenantID, expiredKey)
	require.NoError(t, err)

	rows, _, err := rateLimitRepo.FlushRateLimits(ctx, tenantID, map[string]int{}, nil)
	require.NoError(t, err)

	byKey := make(map[string]*sqlcv1.ListRateLimitsForTenantWithMutateRow, len(rows))
	for _, row := range rows {
		byKey[row.Key] = row
	}

	require.Len(t, byKey, 4)

	require.True(t, byKey[pausedKey].PausedUntil.Valid)
	assert.WithinDuration(t, pausedUntil, byKey[pausedKey].PausedUntil.Time, time.Millisecond)

	require.True(t, byKey[refilledKey].PausedUntil.Valid)
	assert.WithinDuration(t, pausedUntil, byKey[refilledKey].PausedUntil.Time, time.Millisecond)
	assert.Equal(t, int32(10), byKey[refilledKey].Value, "a pause must not stop the window from refilling")

	assert.False(t, byKey[expiredKey].PausedUntil.Valid)
	assert.False(t, byKey[openKey].PausedUntil.Valid)
}

func insertQueueItemForRateLimitTest(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID, taskID int64, taskInsertedAt time.Time, scheduleTimeoutAt *time.Time) *sqlcv1.V1QueueItem {
	t.Helper()

	var id int64
	err := pool.QueryRow(context.Background(), `
		INSERT INTO v1_queue_item (
			tenant_id, queue, task_id, task_inserted_at, external_id, action_id, step_id,
			workflow_id, workflow_run_id, schedule_timeout_at, sticky
		)
		VALUES ($1, 'default', $2, $3, gen_random_uuid(), 'test:run', gen_random_uuid(),
			gen_random_uuid(), gen_random_uuid(), $4, 'NONE')
		RETURNING id
	`, tenantID, taskID, taskInsertedAt, scheduleTimeoutAt).Scan(&id)
	require.NoError(t, err)

	return &sqlcv1.V1QueueItem{
		ID:             id,
		TenantID:       tenantID,
		TaskID:         taskID,
		TaskInsertedAt: pgtype.Timestamptz{Time: taskInsertedAt, Valid: true},
	}
}

func insertFilledConcurrencySlotForRateLimitTest(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID, taskID int64, taskInsertedAt time.Time, scheduleTimeoutAt time.Time) {
	t.Helper()

	_, err := pool.Exec(context.Background(), `
		INSERT INTO v1_concurrency_slot (
			task_id, task_inserted_at, task_retry_count, external_id, tenant_id, workflow_id,
			workflow_version_id, workflow_run_id, strategy_id, key, is_filled, queue_to_notify, schedule_timeout_at
		)
		VALUES ($1, $2, 0, gen_random_uuid(), $3, gen_random_uuid(),
			gen_random_uuid(), gen_random_uuid(), 1, 'test-key', TRUE, 'default', $4)
	`, taskID, taskInsertedAt, tenantID, scheduleTimeoutAt)
	require.NoError(t, err)
}

func readParkedScheduleTimeout(t *testing.T, pool *pgxpool.Pool, taskID int64) (scheduleTimeoutAt pgtype.Timestamp, requeueAfter time.Time) {
	t.Helper()

	err := pool.QueryRow(context.Background(), `SELECT schedule_timeout_at, requeue_after FROM v1_rate_limited_queue_items WHERE task_id = $1`, taskID).Scan(&scheduleTimeoutAt, &requeueAfter)
	require.NoError(t, err)

	return scheduleTimeoutAt, requeueAfter
}

func readSlotScheduleTimeout(t *testing.T, pool *pgxpool.Pool, taskID int64) time.Time {
	t.Helper()

	var scheduleTimeoutAt time.Time
	err := pool.QueryRow(context.Background(), `SELECT schedule_timeout_at FROM v1_concurrency_slot WHERE task_id = $1`, taskID).Scan(&scheduleTimeoutAt)
	require.NoError(t, err)

	return scheduleTimeoutAt
}

// Time a run spends parked behind a paused key must not count against its schedule timeout: parking moves the
// timeout to the end of the pause plus whatever budget the run had left, on both the parked item and its
// concurrency slot, while runs parked for an exhausted window keep their original timeout.
func TestMoveRateLimitedQueueItemsExtendsScheduleTimeoutForPause(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	tenantID := createRateLimitTestTenant(t, pool)

	queueRepo := newQueueRepository(createRateLimitTestShared(pool), tenantID, "default")
	t.Cleanup(queueRepo.Cleanup)

	now := time.Now().UTC().Truncate(time.Millisecond)
	taskInsertedAt := now.Add(-time.Minute)

	pausedUntil := now.Add(time.Hour)
	windowRefillAt := now.Add(30 * time.Second)

	const pausedTask, windowTask, noTimeoutTask, expiredTask int64 = 1, 2, 3, 4

	pausedTimeout := now.Add(10 * time.Minute)
	windowTimeout := now.Add(10 * time.Minute)
	expiredTimeout := now.Add(-time.Minute)
	originalSlotTimeout := now.Add(5 * time.Minute)

	pausedQi := insertQueueItemForRateLimitTest(t, pool, tenantID, pausedTask, taskInsertedAt, &pausedTimeout)
	windowQi := insertQueueItemForRateLimitTest(t, pool, tenantID, windowTask, taskInsertedAt, &windowTimeout)
	noTimeoutQi := insertQueueItemForRateLimitTest(t, pool, tenantID, noTimeoutTask, taskInsertedAt, nil)
	expiredQi := insertQueueItemForRateLimitTest(t, pool, tenantID, expiredTask, taskInsertedAt, &expiredTimeout)

	insertFilledConcurrencySlotForRateLimitTest(t, pool, tenantID, pausedTask, taskInsertedAt, originalSlotTimeout)
	insertFilledConcurrencySlotForRateLimitTest(t, pool, tenantID, windowTask, taskInsertedAt, originalSlotTimeout)

	toMove := func(qi *sqlcv1.V1QueueItem, nextRefillAt time.Time, paused bool) *RateLimitResult {
		res := &RateLimitResult{
			V1QueueItem:    qi,
			ExceededKey:    "upstream-api",
			NextRefillAt:   &nextRefillAt,
			TaskId:         qi.TaskID,
			TaskInsertedAt: qi.TaskInsertedAt,
		}

		if paused {
			res.PausedUntil = &nextRefillAt
		}

		return res
	}

	_, _, err := queueRepo.MarkQueueItemsProcessed(ctx, &AssignResults{
		RateLimitedToMove: []*RateLimitResult{
			toMove(pausedQi, pausedUntil, true),
			toMove(windowQi, windowRefillAt, false),
			toMove(noTimeoutQi, pausedUntil, true),
			toMove(expiredQi, pausedUntil, true),
		},
	})
	require.NoError(t, err)

	var remaining int
	err = pool.QueryRow(ctx, `SELECT count(*) FROM v1_queue_item WHERE tenant_id = $1`, tenantID).Scan(&remaining)
	require.NoError(t, err)
	assert.Zero(t, remaining, "every rate limited item should have been parked")

	// tolerance covers the gap between the test's clock and NOW() inside the move statement
	const tolerance = 5 * time.Second

	parkedTimeout, requeueAfter := readParkedScheduleTimeout(t, pool, pausedTask)
	require.True(t, parkedTimeout.Valid)
	assert.WithinDuration(t, pausedUntil.Add(10*time.Minute), parkedTimeout.Time, tolerance)
	assert.WithinDuration(t, pausedUntil, requeueAfter, time.Millisecond)
	assert.WithinDuration(t, parkedTimeout.Time, readSlotScheduleTimeout(t, pool, pausedTask), time.Millisecond, "the filled concurrency slot must not time out before the parked item")

	parkedTimeout, requeueAfter = readParkedScheduleTimeout(t, pool, windowTask)
	require.True(t, parkedTimeout.Valid)
	assert.WithinDuration(t, windowTimeout, parkedTimeout.Time, time.Millisecond, "waiting on a window still counts against the schedule timeout")
	assert.WithinDuration(t, windowRefillAt, requeueAfter, time.Millisecond)
	assert.WithinDuration(t, originalSlotTimeout, readSlotScheduleTimeout(t, pool, windowTask), time.Millisecond)

	parkedTimeout, _ = readParkedScheduleTimeout(t, pool, noTimeoutTask)
	assert.False(t, parkedTimeout.Valid, "a pause must not add a timeout to a run that had none")

	parkedTimeout, _ = readParkedScheduleTimeout(t, pool, expiredTask)
	require.True(t, parkedTimeout.Valid)
	assert.WithinDuration(t, pausedUntil, parkedTimeout.Time, time.Millisecond, "a run with no budget left times out when the pause ends")
}

// A key that stays paused across several requeues gets re-parked each time; the run's remaining budget must
// carry over instead of growing by the length of every pause.
func TestMoveRateLimitedQueueItemsKeepsBudgetAcrossRepeatedPauses(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	tenantID := createRateLimitTestTenant(t, pool)

	shared := createRateLimitTestShared(pool)
	shared.queueCache = cache.New(5 * time.Minute)

	queueRepo := newQueueRepository(shared, tenantID, "default")
	t.Cleanup(queueRepo.Cleanup)

	now := time.Now().UTC().Truncate(time.Millisecond)
	taskInsertedAt := now.Add(-time.Minute)
	scheduleTimeoutAt := now.Add(10 * time.Minute)

	const taskID int64 = 1

	qi := insertQueueItemForRateLimitTest(t, pool, tenantID, taskID, taskInsertedAt, &scheduleTimeoutAt)

	park := func(qi *sqlcv1.V1QueueItem, pausedUntil time.Time) {
		_, _, err := queueRepo.MarkQueueItemsProcessed(ctx, &AssignResults{
			RateLimitedToMove: []*RateLimitResult{{
				V1QueueItem:    qi,
				ExceededKey:    "upstream-api",
				NextRefillAt:   &pausedUntil,
				PausedUntil:    &pausedUntil,
				TaskId:         qi.TaskID,
				TaskInsertedAt: qi.TaskInsertedAt,
			}},
		})
		require.NoError(t, err)
	}

	park(qi, now.Add(time.Hour))

	// fast-forward to the end of the first pause: the item is due and its full 10 minute budget is still left
	_, err := pool.Exec(ctx, `
		UPDATE v1_rate_limited_queue_items
		SET requeue_after = NOW() - INTERVAL '1 second', schedule_timeout_at = (NOW() + INTERVAL '10 minutes')::TIMESTAMP(3)
		WHERE task_id = $1
	`, taskID)
	require.NoError(t, err)

	requeued, err := queueRepo.RequeueRateLimitedItems(ctx, tenantID, "default")
	require.NoError(t, err)
	require.Len(t, requeued, 1)

	// the scheduler finds the key paused again and re-parks the item
	secondPause := time.Now().UTC().Add(2 * time.Hour)

	park(&sqlcv1.V1QueueItem{
		ID:             requeued[0].ID,
		TenantID:       tenantID,
		TaskID:         taskID,
		TaskInsertedAt: qi.TaskInsertedAt,
	}, secondPause)

	parkedTimeout, _ := readParkedScheduleTimeout(t, pool, taskID)
	require.True(t, parkedTimeout.Valid)
	assert.WithinDuration(t, secondPause.Add(10*time.Minute), parkedTimeout.Time, 5*time.Second)
}

// A dynamic key can be paused before any run has rendered it, so the pause must apply on the same flush that
// creates the key's row from its first definition.
func TestFlushRateLimitsAppliesPauseToNewDynamicKey(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	tenantID := createRateLimitTestTenant(t, pool)
	rateLimitRepo := newRateLimitRepository(createRateLimitTestShared(pool))

	const key = "upstream-api:tenant-123"

	pausedUntil := time.Now().UTC().Add(time.Hour).Truncate(time.Millisecond)

	_, err := rateLimitRepo.PauseRateLimit(ctx, tenantID, key, pausedUntil)
	require.NoError(t, err)

	rows, _, err := rateLimitRepo.FlushRateLimits(ctx, tenantID, map[string]int{}, nil)
	require.NoError(t, err)
	assert.Empty(t, rows, "a pause alone must not create a rate limit row")

	rows, _, err = rateLimitRepo.FlushRateLimits(ctx, tenantID, map[string]int{}, map[string]RateLimitDefinition{key: {LimitValue: 10, Window: "1 MINUTE"}})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, key, rows[0].Key)
	require.True(t, rows[0].PausedUntil.Valid)
	assert.WithinDuration(t, pausedUntil, rows[0].PausedUntil.Time, time.Millisecond)
}
