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

func newRateLimitPauseTestRepo(pool *pgxpool.Pool) *rateLimitRepository {
	shared := createRateLimitTestShared(pool)
	shared.v = validator.NewDefaultValidator()

	return newRateLimitRepository(shared)
}

// upsertUsedRateLimit creates a rate limit and spends used of its units, the way the scheduler would.
func upsertUsedRateLimit(t *testing.T, r *rateLimitRepository, tenantID uuid.UUID, key string, limit int, duration string, used int) *sqlcv1.RateLimit {
	t.Helper()

	rl, err := r.UpsertRateLimit(context.Background(), tenantID, key, &UpsertRateLimitOpts{Limit: limit, Duration: &duration})
	require.NoError(t, err)

	if used > 0 {
		_, _, err = r.FlushRateLimits(context.Background(), tenantID, map[string]int{key: used}, nil)
		require.NoError(t, err)
	}

	return rl
}

func waitForPauseToEnd(pause *sqlcv1.V1RateLimitPause) {
	time.Sleep(time.Until(pause.PausedUntil.Time) + 100*time.Millisecond)
}

// shiftedLastRefill is where a window that was frozen for the whole of the pause resumes.
func shiftedLastRefill(rl *sqlcv1.RateLimit, pause *sqlcv1.V1RateLimitPause) time.Time {
	return rl.LastRefill.Time.Add(pause.PausedUntil.Time.Sub(pause.PausedAt.Time))
}

func readPausedUntil(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID, key string) time.Time {
	t.Helper()

	var pausedUntil time.Time
	err := pool.QueryRow(context.Background(), `SELECT paused_until FROM v1_rate_limit_pause WHERE tenant_id = $1 AND key = $2`, tenantID, key).Scan(&pausedUntil)
	require.NoError(t, err)

	return pausedUntil
}

func readRateLimitRow(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID, key string) (value int32, lastRefill time.Time) {
	t.Helper()

	err := pool.QueryRow(context.Background(), `
		SELECT "value", "lastRefill" FROM "RateLimit" WHERE "tenantId" = $1 AND "key" = $2
	`, tenantID, key).Scan(&value, &lastRefill)
	require.NoError(t, err)

	return value, lastRefill
}

// A later pause call with an earlier deadline must not shorten an active pause, so concurrent tasks hitting the
// same upstream can each pause the key with their own Retry-After without racing. Extending a pause keeps its
// original start, so the whole span from the first pause is frozen.
func TestPauseRateLimitKeepsLongestPause(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	tenantID := createRateLimitTestTenant(t, pool)
	rateLimitRepo := newRateLimitRepository(createRateLimitTestShared(pool))

	const key = "upstream-api:tenant-123"

	now := time.Now().UTC().Truncate(time.Millisecond)

	first, err := rateLimitRepo.PauseRateLimit(ctx, tenantID, key, now.Add(time.Hour))
	require.NoError(t, err)
	assert.WithinDuration(t, now.Add(time.Hour), first.PausedUntil.Time, time.Millisecond)

	pause, err := rateLimitRepo.PauseRateLimit(ctx, tenantID, key, now.Add(10*time.Minute))
	require.NoError(t, err)
	assert.WithinDuration(t, now.Add(time.Hour), pause.PausedUntil.Time, time.Millisecond, "an earlier deadline must not shorten the pause")
	assert.WithinDuration(t, now.Add(time.Hour), readPausedUntil(t, pool, tenantID, key), time.Millisecond)

	pause, err = rateLimitRepo.PauseRateLimit(ctx, tenantID, key, now.Add(2*time.Hour))
	require.NoError(t, err)
	assert.WithinDuration(t, now.Add(2*time.Hour), pause.PausedUntil.Time, time.Millisecond)
	assert.True(t, first.PausedAt.Time.Equal(pause.PausedAt.Time), "extending a pause must keep its start")

	_, err = rateLimitRepo.PauseRateLimit(ctx, tenantID, "", now.Add(time.Hour))
	assert.Error(t, err)
}

// The scheduler only learns about pauses through FlushRateLimits, and an ended pause must read as unpaused.
func TestFlushRateLimitsReturnsActivePauses(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	tenantID := createRateLimitTestTenant(t, pool)
	rateLimitRepo := newRateLimitPauseTestRepo(pool)

	const pausedKey, endedKey, openKey, refilledKey = "rl-paused", "rl-ended", "rl-open", "rl-refilled"

	upsertUsedRateLimit(t, rateLimitRepo, tenantID, pausedKey, 10, "HOUR", 0)
	upsertUsedRateLimit(t, rateLimitRepo, tenantID, endedKey, 10, "HOUR", 0)
	upsertUsedRateLimit(t, rateLimitRepo, tenantID, openKey, 10, "HOUR", 0)
	upsertUsedRateLimit(t, rateLimitRepo, tenantID, refilledKey, 10, "SECOND", 10)

	pausedUntil := time.Now().UTC().Add(time.Hour).Truncate(time.Millisecond)

	_, err := rateLimitRepo.PauseRateLimit(ctx, tenantID, pausedKey, pausedUntil)
	require.NoError(t, err)

	_, err = rateLimitRepo.PauseRateLimit(ctx, tenantID, refilledKey, pausedUntil)
	require.NoError(t, err)

	ended, err := rateLimitRepo.PauseRateLimit(ctx, tenantID, endedKey, time.Now().Add(500*time.Millisecond))
	require.NoError(t, err)

	// also lets the refilled key's window elapse, so it would take the refill branch of the query if it weren't paused
	waitForPauseToEnd(ended)
	time.Sleep(time.Second)

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

	assert.False(t, byKey[endedKey].PausedUntil.Valid)
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

// A paused key keeps the units it had left: with a limit of 20 and 3 used before the pause, only 17 runs
// may start when the pause ends, even if the window would have refilled in the meantime.
func TestFlushRateLimitsFreezesUnitsDuringPause(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	tenantID := createRateLimitTestTenant(t, pool)
	rateLimitRepo := newRateLimitPauseTestRepo(pool)

	const key = "upstream-api"

	upsertUsedRateLimit(t, rateLimitRepo, tenantID, key, 20, "SECOND", 3)

	_, err := rateLimitRepo.PauseRateLimit(ctx, tenantID, key, time.Now().Add(time.Hour))
	require.NoError(t, err)

	// the one second window elapses while the key is paused
	time.Sleep(1100 * time.Millisecond)

	rows, nextRefillAt, err := rateLimitRepo.FlushRateLimits(ctx, tenantID, map[string]int{}, nil)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, int32(17), rows[0].Value, "an elapsed window must not refill while the key is paused")
	assert.True(t, rows[0].PausedUntil.Valid)

	// a frozen window's refill time is in the past, which would otherwise make the scheduler flush constantly
	require.NotNil(t, nextRefillAt)
	assert.True(t, nextRefillAt.After(time.Now()), "the next refill must not be scheduled in the past for a paused key")

	listed, err := rateLimitRepo.ListRateLimits(ctx, tenantID, &ListRateLimitOpts{})
	require.NoError(t, err)
	require.Len(t, listed.Rows, 1)
	assert.Equal(t, int32(17), listed.Rows[0].Value)
}

// When a pause ends, the window resumes where it was, with the same units, once the scheduler settles it.
func TestFlushRateLimitsResumesWindowWherePauseStarted(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	tenantID := createRateLimitTestTenant(t, pool)
	rateLimitRepo := newRateLimitPauseTestRepo(pool)

	const key, elapsedKey = "upstream-api", "elapsed-before-pause"

	rl := upsertUsedRateLimit(t, rateLimitRepo, tenantID, key, 20, "MINUTE", 3)
	upsertUsedRateLimit(t, rateLimitRepo, tenantID, elapsedKey, 20, "SECOND", 20)

	// the elapsed key's one second window runs out before its pause starts, so it refills as soon as the pause ends
	time.Sleep(1100 * time.Millisecond)

	pause, err := rateLimitRepo.PauseRateLimit(ctx, tenantID, key, time.Now().Add(500*time.Millisecond))
	require.NoError(t, err)

	_, err = rateLimitRepo.PauseRateLimit(ctx, tenantID, elapsedKey, pause.PausedUntil.Time)
	require.NoError(t, err)

	waitForPauseToEnd(pause)

	rows, _, err := rateLimitRepo.FlushRateLimits(ctx, tenantID, map[string]int{}, nil)
	require.NoError(t, err)

	byKey := make(map[string]*sqlcv1.ListRateLimitsForTenantWithMutateRow, len(rows))
	for _, row := range rows {
		byKey[row.Key] = row
	}

	require.Contains(t, byKey, key)
	assert.Equal(t, int32(17), byKey[key].Value)
	assert.False(t, byKey[key].PausedUntil.Valid)

	value, lastRefill := readRateLimitRow(t, pool, tenantID, key)
	assert.Equal(t, int32(17), value)
	assert.WithinDuration(t, shiftedLastRefill(rl, pause), lastRefill, 2*time.Millisecond, "the window should have the time it had left when the pause started")

	require.Contains(t, byKey, elapsedKey)
	assert.Equal(t, int32(20), byKey[elapsedKey].Value)

	var remaining int
	err = pool.QueryRow(ctx, `SELECT count(*) FROM v1_rate_limit_pause WHERE tenant_id = $1`, tenantID).Scan(&remaining)
	require.NoError(t, err)
	assert.Zero(t, remaining, "expired pauses are removed once their window shift is applied")
}

// Re-pausing a key whose last pause ended before the scheduler settled it must settle that pause first, so the
// old pause's frozen span still shifts the window and the new pause starts its own span.
func TestPauseRateLimitSettlesUnsettledExpiredPause(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	tenantID := createRateLimitTestTenant(t, pool)
	rateLimitRepo := newRateLimitPauseTestRepo(pool)

	const key = "upstream-api"

	rl := upsertUsedRateLimit(t, rateLimitRepo, tenantID, key, 20, "MINUTE", 3)

	ended, err := rateLimitRepo.PauseRateLimit(ctx, tenantID, key, time.Now().Add(500*time.Millisecond))
	require.NoError(t, err)

	waitForPauseToEnd(ended)

	until := time.Now().Add(time.Hour)

	pause, err := rateLimitRepo.PauseRateLimit(ctx, tenantID, key, until)
	require.NoError(t, err)
	assert.True(t, pause.PausedAt.Time.After(ended.PausedUntil.Time), "the new pause starts now")
	assert.WithinDuration(t, until, pause.PausedUntil.Time, time.Millisecond, "an ended pause's deadline must not carry over")

	_, lastRefill := readRateLimitRow(t, pool, tenantID, key)
	assert.WithinDuration(t, shiftedLastRefill(rl, ended), lastRefill, 2*time.Millisecond, "the ended pause's frozen span still shifts the window")
}

// A pause requested with a deadline that has already passed, like a Retry-After that arrives late, froze nothing,
// so neither listing nor settling it may move the window's last refill earlier and let runs start too soon.
func TestPastDeadlinePauseDoesNotShiftWindow(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	tenantID := createRateLimitTestTenant(t, pool)
	rateLimitRepo := newRateLimitPauseTestRepo(pool)

	const key = "upstream-api"

	upsertUsedRateLimit(t, rateLimitRepo, tenantID, key, 20, "MINUTE", 3)
	_, lastRefill := readRateLimitRow(t, pool, tenantID, key)

	time.Sleep(time.Second)

	// after the last refill but before the pause request, the span that would be subtracted
	pause, err := rateLimitRepo.PauseRateLimit(ctx, tenantID, key, lastRefill.Add(500*time.Millisecond))
	require.NoError(t, err)
	require.True(t, pause.PausedUntil.Time.Before(pause.PausedAt.Time))

	listed, err := rateLimitRepo.ListRateLimits(ctx, tenantID, &ListRateLimitOpts{})
	require.NoError(t, err)
	require.Len(t, listed.Rows, 1)
	assert.EqualValues(t, 17, listed.Rows[0].Value)
	assert.WithinDuration(t, lastRefill, listed.Rows[0].LastRefill.Time, time.Millisecond, "listing must not move the window earlier")
	assert.False(t, listed.Rows[0].PausedUntil.Valid)

	_, _, err = rateLimitRepo.FlushRateLimits(ctx, tenantID, map[string]int{}, nil)
	require.NoError(t, err)

	value, settledLastRefill := readRateLimitRow(t, pool, tenantID, key)
	assert.EqualValues(t, 17, value)
	assert.WithinDuration(t, lastRefill, settledLastRefill, time.Millisecond, "settling must not move the window earlier")
}

// Until the scheduler settles an ended pause, listing must already apply its window shift, or a key with 17 of 20
// units left would show a full bucket that goes back to 17 once settled.
func TestListRateLimitsShiftsWindowOfUnsettledExpiredPause(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	tenantID := createRateLimitTestTenant(t, pool)
	rateLimitRepo := newRateLimitPauseTestRepo(pool)

	const endedKey, pausedKey = "ended", "paused"

	endedRl := upsertUsedRateLimit(t, rateLimitRepo, tenantID, endedKey, 20, "SECOND", 3)
	upsertUsedRateLimit(t, rateLimitRepo, tenantID, pausedKey, 20, "SECOND", 3)

	ended, err := rateLimitRepo.PauseRateLimit(ctx, tenantID, endedKey, time.Now().Add(1500*time.Millisecond))
	require.NoError(t, err)

	_, err = rateLimitRepo.PauseRateLimit(ctx, tenantID, pausedKey, time.Now().Add(time.Hour))
	require.NoError(t, err)

	// unshifted, both one second windows have elapsed by now
	waitForPauseToEnd(ended)

	listed, err := rateLimitRepo.ListRateLimits(ctx, tenantID, &ListRateLimitOpts{})
	require.NoError(t, err)
	require.Len(t, listed.Rows, 2)

	rows := make(map[string]*sqlcv1.ListRateLimitsForTenantNoMutateRow, len(listed.Rows))

	for _, row := range listed.Rows {
		rows[row.Key] = row
	}

	endedRow := rows[endedKey]
	assert.EqualValues(t, 17, endedRow.Value, "the frozen window still has time left")
	assert.WithinDuration(t, shiftedLastRefill(endedRl, ended), endedRow.LastRefill.Time, 2*time.Millisecond)
	assert.False(t, endedRow.PausedUntil.Valid)

	pausedRow := rows[pausedKey]
	assert.EqualValues(t, 17, pausedRow.Value, "a paused key doesn't refill")
	assert.True(t, pausedRow.PausedUntil.Valid)
}
