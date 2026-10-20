//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/validator"
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

	// its window has elapsed, so it would take the refill branch of the query if it weren't paused
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
	assert.Equal(t, int32(0), byKey[refilledKey].Value, "a paused key must not refill")

	assert.False(t, byKey[expiredKey].PausedUntil.Valid)
	assert.False(t, byKey[openKey].PausedUntil.Valid)
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

func insertRateLimitPause(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID, key string, pausedAgo, untilFromNow time.Duration) {
	t.Helper()

	_, err := pool.Exec(context.Background(), `
		INSERT INTO v1_rate_limit_pause (tenant_id, key, paused_at, paused_until)
		VALUES ($1, $2, NOW() - $3::interval, NOW() + $4::interval)
	`, tenantID, key, pausedAgo.String(), untilFromNow.String())
	require.NoError(t, err)
}

func readRateLimitRow(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID, key string) (value int32, sinceLastRefill time.Duration) {
	t.Helper()

	var micros int64
	err := pool.QueryRow(context.Background(), `
		SELECT "value", (EXTRACT(EPOCH FROM (NOW() - "lastRefill")) * 1000000)::bigint
		FROM "RateLimit" WHERE "tenantId" = $1 AND "key" = $2
	`, tenantID, key).Scan(&value, &micros)
	require.NoError(t, err)

	return value, time.Duration(micros) * time.Microsecond
}

// A paused key keeps the units it had left: with a limit of 20 and 3 used before the pause, only 17 runs
// may start when the pause ends, even if the window would have refilled in the meantime.
func TestFlushRateLimitsFreezesUnitsDuringPause(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	tenantID := createRateLimitTestTenant(t, pool)
	rateLimitRepo := newRateLimitRepository(createRateLimitTestShared(pool))

	const key = "upstream-api"

	insertRateLimit(t, pool, tenantID, key, 20, 17, "1 minute", 2*time.Minute)
	insertRateLimitPause(t, pool, tenantID, key, 90*time.Second, time.Hour)

	rows, nextRefillAt, err := rateLimitRepo.FlushRateLimits(ctx, tenantID, map[string]int{}, nil)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, int32(17), rows[0].Value, "an elapsed window must not refill while the key is paused")
	assert.True(t, rows[0].PausedUntil.Valid)

	// a frozen window's refill time is in the past, which would otherwise make the scheduler flush constantly
	require.NotNil(t, nextRefillAt)
	assert.True(t, nextRefillAt.After(time.Now()), "the next refill must not be scheduled in the past for a paused key")

	shared := createRateLimitTestShared(pool)
	shared.v = validator.NewDefaultValidator()

	listed, err := newRateLimitRepository(shared).ListRateLimits(ctx, tenantID, &ListRateLimitOpts{})
	require.NoError(t, err)
	require.Len(t, listed.Rows, 1)
	assert.Equal(t, int32(17), listed.Rows[0].Value)
}

// When a pause ends, the window resumes where it was: a pause that started 20 seconds into a 1 minute window
// leaves 40 seconds of that window, with the same units, once it ends.
func TestFlushRateLimitsResumesWindowWherePauseStarted(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	tenantID := createRateLimitTestTenant(t, pool)
	rateLimitRepo := newRateLimitRepository(createRateLimitTestShared(pool))

	const key, elapsedKey = "upstream-api", "elapsed-before-pause"

	insertRateLimit(t, pool, tenantID, key, 20, 17, "1 minute", 5*time.Minute+20*time.Second)
	insertRateLimitPause(t, pool, tenantID, key, 5*time.Minute, -time.Second)

	// this window had already run out before the pause started, so it refills as soon as the pause ends
	insertRateLimit(t, pool, tenantID, elapsedKey, 20, 0, "1 minute", 10*time.Minute)
	insertRateLimitPause(t, pool, tenantID, elapsedKey, 5*time.Minute, -time.Second)

	rows, _, err := rateLimitRepo.FlushRateLimits(ctx, tenantID, map[string]int{}, nil)
	require.NoError(t, err)

	byKey := make(map[string]*sqlcv1.ListRateLimitsForTenantWithMutateRow, len(rows))
	for _, row := range rows {
		byKey[row.Key] = row
	}

	require.Contains(t, byKey, key)
	assert.Equal(t, int32(17), byKey[key].Value)
	assert.False(t, byKey[key].PausedUntil.Valid)

	value, sinceLastRefill := readRateLimitRow(t, pool, tenantID, key)
	assert.Equal(t, int32(17), value)
	assert.InDelta(t, (21 * time.Second).Seconds(), sinceLastRefill.Seconds(), 1, "the window should have the time it had left when the pause started")

	require.Contains(t, byKey, elapsedKey)
	assert.Equal(t, int32(20), byKey[elapsedKey].Value)

	var remaining int
	err = pool.QueryRow(ctx, `SELECT count(*) FROM v1_rate_limit_pause WHERE tenant_id = $1`, tenantID).Scan(&remaining)
	require.NoError(t, err)
	assert.Zero(t, remaining, "expired pauses are removed once their window shift is applied")
}

// Extending a pause must keep its original start, so the whole span from the first pause is frozen.
func TestPauseRateLimitKeepsPauseStart(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	tenantID := createRateLimitTestTenant(t, pool)
	rateLimitRepo := newRateLimitRepository(createRateLimitTestShared(pool))

	const key = "upstream-api"

	first, err := rateLimitRepo.PauseRateLimit(ctx, tenantID, key, time.Now().Add(time.Minute))
	require.NoError(t, err)

	second, err := rateLimitRepo.PauseRateLimit(ctx, tenantID, key, time.Now().Add(time.Hour))
	require.NoError(t, err)

	assert.True(t, first.PausedAt.Time.Equal(second.PausedAt.Time))
	assert.True(t, second.PausedUntil.Time.After(first.PausedUntil.Time))
}

// Re-pausing a key whose last pause ended before the scheduler settled it must settle that pause first, so the
// old pause's frozen span still shifts the window and the new pause starts its own span.
func TestPauseRateLimitSettlesUnsettledExpiredPause(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	tenantID := createRateLimitTestTenant(t, pool)
	rateLimitRepo := newRateLimitRepository(createRateLimitTestShared(pool))

	const key = "upstream-api"

	insertRateLimit(t, pool, tenantID, key, 20, 17, "1 minute", 5*time.Minute+20*time.Second)
	insertRateLimitPause(t, pool, tenantID, key, 5*time.Minute, -time.Second)

	until := time.Now().Add(time.Hour)

	pause, err := rateLimitRepo.PauseRateLimit(ctx, tenantID, key, until)
	require.NoError(t, err)
	assert.WithinDuration(t, time.Now(), pause.PausedAt.Time, 5*time.Second, "the new pause starts now")
	assert.WithinDuration(t, until, pause.PausedUntil.Time, time.Millisecond, "an ended pause's deadline must not carry over")

	_, sinceLastRefill := readRateLimitRow(t, pool, tenantID, key)
	assert.InDelta(t, (21 * time.Second).Seconds(), sinceLastRefill.Seconds(), 1, "the ended pause's frozen span still shifts the window")
}
