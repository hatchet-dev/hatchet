//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func TestUpdateRateLimits_ChargesUsageToReservedWindow(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()
	tenantID := createLimitTestTenant(t, pool)

	l := zerolog.Nop()
	repo := &rateLimitRepository{
		sharedRepository: &sharedRepository{
			pool:    pool,
			l:       &l,
			queries: sqlcv1.New(),
		},
	}

	// a minute-long window that has already expired, so the next flush refills it
	var window0 pgtype.Timestamp
	err := pool.QueryRow(ctx, `
		INSERT INTO "RateLimit" ("tenantId", "key", "limitValue", "value", "window", "lastRefill")
		VALUES ($1, 'key', 10, 10, '1 minute', CURRENT_TIMESTAMP - INTERVAL '2 minutes')
		RETURNING "lastRefill"
	`, tenantID).Scan(&window0)
	require.NoError(t, err)

	flush := func(units int, windowStart time.Time) *sqlcv1.ListRateLimitsForTenantWithMutateRow {
		t.Helper()

		rows, _, err := repo.UpdateRateLimits(ctx, tenantID, map[string]RateLimitUsage{
			"key": {Units: units, WindowStart: windowStart},
		})
		require.NoError(t, err)
		require.Len(t, rows, 1)

		return rows[0]
	}

	refilled := flush(4, window0.Time)
	require.EqualValues(t, 10, refilled.Value, "usage flushed after the refill boundary must not reduce the new window")
	require.True(t, refilled.LastRefill.Time.After(window0.Time))

	require.EqualValues(t, 10, flush(4, window0.Time).Value, "usage from a refilled window must be dropped")
	require.EqualValues(t, 6, flush(4, refilled.LastRefill.Time).Value, "usage from the current window must be charged")
}
