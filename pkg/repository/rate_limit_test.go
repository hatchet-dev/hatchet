//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func createRateLimitTestShared(pool *pgxpool.Pool) *sharedRepository {
	logger := zerolog.New(io.Discard)

	return &sharedRepository{
		pool:    pool,
		ddlPool: pool,
		l:       &logger,
		queries: sqlcv1.New(),
	}
}

func createRateLimitTestTenant(t *testing.T, pool *pgxpool.Pool) uuid.UUID {
	t.Helper()

	tenantID := uuid.New()
	slug := "tenant-" + strings.ReplaceAll(tenantID.String(), "-", "")

	_, err := pool.Exec(context.Background(), `INSERT INTO "Tenant" ("id", "name", "slug") VALUES ($1, $2, $3)`, tenantID, "Test Tenant", slug)
	require.NoError(t, err)

	return tenantID
}

func insertRateLimit(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID, key string, limit, value int, window string, lastRefillAgo time.Duration) {
	t.Helper()

	_, err := pool.Exec(context.Background(), `
		INSERT INTO "RateLimit" ("tenantId", "key", "limitValue", "value", "window", "lastRefill")
		VALUES ($1, $2, $3, $4, $5, (NOW() - $6::interval)::timestamp)
	`, tenantID, key, limit, value, window, lastRefillAgo.String())
	require.NoError(t, err)
}

func insertDynamicRateLimitEvals(t *testing.T, pool *pgxpool.Pool, taskID int64, taskInsertedAt time.Time, globalKeyToKey map[string]string) {
	t.Helper()

	for globalKey, rlKey := range globalKeyToKey {
		_, err := pool.Exec(context.Background(), `
			INSERT INTO v1_task_expression_eval (key, task_id, task_inserted_at, value_str, value_int, kind) VALUES
				($1, $2, $3, $4, NULL, 'DYNAMIC_RATE_LIMIT_KEY'),
				($1, $2, $3, NULL, 10, 'DYNAMIC_RATE_LIMIT_VALUE'),
				($1, $2, $3, 'HOUR', NULL, 'DYNAMIC_RATE_LIMIT_WINDOW'),
				($1, $2, $3, NULL, 1, 'DYNAMIC_RATE_LIMIT_UNITS')
		`, globalKey, taskID, taskInsertedAt, rlKey)
		require.NoError(t, err)
	}
}

func newRateLimitTestQueueItems(tenantID uuid.UUID, taskID int64, taskInsertedAt time.Time) []*sqlcv1.V1QueueItem {
	return []*sqlcv1.V1QueueItem{{
		TenantID:       tenantID,
		TaskID:         taskID,
		TaskInsertedAt: pgtype.Timestamptz{Time: taskInsertedAt, Valid: true},
		StepID:         uuid.New(),
	}}
}

// GetTaskRateLimits must not take "RateLimit" row locks: FlushRateLimits is the only multi-row writer, and a
// second writer outside its advisory lock can lock rows in a different order and deadlock (40P01).
func TestGetTaskRateLimitsDoesNotLockRateLimitRows(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	tenantID := createRateLimitTestTenant(t, pool)

	const keyA, keyC = "rl-a", "rl-c"

	insertRateLimit(t, pool, tenantID, keyA, 10, 10, "1 HOUR", 2*time.Hour)
	insertRateLimit(t, pool, tenantID, keyC, 10, 10, "1 HOUR", 0)

	taskID := int64(1)
	taskInsertedAt := time.Now().UTC()

	insertDynamicRateLimitEvals(t, pool, taskID, taskInsertedAt, map[string]string{"ga": keyA, "gc": keyC})

	shared := createRateLimitTestShared(pool)
	queueRepo := newQueueRepository(shared, tenantID, "default")
	t.Cleanup(queueRepo.Cleanup)

	queueItems := newRateLimitTestQueueItems(tenantID, taskID, taskInsertedAt)

	// a concurrent FlushRateLimits holds these row locks while it charges and refills
	blocker, err := pool.Begin(ctx)
	require.NoError(t, err)
	defer blocker.Rollback(ctx) // nolint: errcheck

	_, err = blocker.Exec(ctx, `SELECT 1 FROM "RateLimit" WHERE "tenantId" = $1 FOR UPDATE`, tenantID)
	require.NoError(t, err)

	otx, err := shared.PrepareOptimisticTx(ctx)
	require.NoError(t, err)
	defer otx.Rollback()

	// the lock timeout turns a regression into a fast failure instead of a hang
	_, err = otx.tx.Exec(ctx, `SET LOCAL lock_timeout = '2s'`)
	require.NoError(t, err)

	units, definitions, err := queueRepo.GetTaskRateLimits(ctx, otx, queueItems)
	require.NoError(t, err)
	require.NoError(t, otx.Commit(ctx))

	assert.Equal(t, map[string]int32{keyA: 1, keyC: 1}, units[taskID])
	assert.Equal(t, map[string]RateLimitDefinition{
		keyA: {LimitValue: 10, Window: "1 HOUR"},
		keyC: {LimitValue: 10, Window: "1 HOUR"},
	}, definitions)
}

// FlushRateLimits creates rows for new dynamic definitions, and an unchanged definition must not rewrite
// the row, since the scheduler passes definitions on every flush that sees them.
func TestFlushRateLimitsUpsertsDefinitionsWithoutNoopWrites(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	tenantID := createRateLimitTestTenant(t, pool)

	const key = "rl-noop"

	rateLimitRepo := newRateLimitRepository(createRateLimitTestShared(pool))

	// xmin changes whenever the row is rewritten, even if every column value is identical
	readRow := func() (xmin string, limitValue int32, window string) {
		err := pool.QueryRow(ctx, `SELECT xmin::text, "limitValue", "window" FROM "RateLimit" WHERE "tenantId" = $1 AND "key" = $2`, tenantID, key).Scan(&xmin, &limitValue, &window)
		require.NoError(t, err)
		return xmin, limitValue, window
	}

	def := map[string]RateLimitDefinition{key: {LimitValue: 10, Window: "1 HOUR"}}

	rows, _, err := rateLimitRepo.FlushRateLimits(ctx, tenantID, map[string]int{}, def)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, key, rows[0].Key)

	firstXmin, _, _ := readRow()

	for range 3 {
		_, _, err = rateLimitRepo.FlushRateLimits(ctx, tenantID, map[string]int{}, def)
		require.NoError(t, err)
	}

	xmin, _, _ := readRow()
	assert.Equal(t, firstXmin, xmin, "unchanged dynamic rate limit definition should not rewrite the RateLimit row")

	_, _, err = rateLimitRepo.FlushRateLimits(ctx, tenantID, map[string]int{}, map[string]RateLimitDefinition{key: {LimitValue: 5, Window: "1 MINUTE"}})
	require.NoError(t, err)

	xmin, limitValue, window := readRow()
	assert.NotEqual(t, firstXmin, xmin)
	assert.Equal(t, int32(5), limitValue)
	assert.Equal(t, "1 MINUTE", window)
}

// The scheduler only flushes once a window has elapsed, so the units it flushes were spent in the previous
// window and must not be charged against the refilled budget.
func TestFlushRateLimitsChargesUsageToWindowItWasSpentIn(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	tenantID := createRateLimitTestTenant(t, pool)

	const key, openKey = "rl-refill", "rl-open"
	const limit = 10

	// the previous window ended one second ago and its value was never decremented in the database
	insertRateLimit(t, pool, tenantID, key, limit, limit, "1 second", 2*time.Second)

	// this window is still open, so its usage comes out of the current budget
	insertRateLimit(t, pool, tenantID, openKey, limit, limit, "1 HOUR", 0)

	rateLimitRepo := newRateLimitRepository(createRateLimitTestShared(pool))

	// the scheduler used the entire previous window's budget in memory before this flush
	rows, _, err := rateLimitRepo.FlushRateLimits(ctx, tenantID, map[string]int{key: limit, openKey: 3}, nil)
	require.NoError(t, err)

	returned := make(map[string]int32, len(rows))
	for _, row := range rows {
		returned[row.Key] = row.Value
	}

	assert.Equal(t, int32(limit), returned[key], "new window should start with its full budget, not be charged for the previous window's usage")
	assert.Equal(t, int32(limit-3), returned[openKey])

	readValue := func(k string) int32 {
		var v int32
		err := pool.QueryRow(ctx, `SELECT "value" FROM "RateLimit" WHERE "tenantId" = $1 AND "key" = $2`, tenantID, k).Scan(&v)
		require.NoError(t, err)
		return v
	}

	assert.Equal(t, int32(limit), readValue(key))
	assert.Equal(t, int32(limit-3), readValue(openKey))
}
