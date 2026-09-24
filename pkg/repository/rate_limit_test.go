//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
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

func TestUpdateRateLimits_ChargesUsageBeforeRefilling(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	logger := zerolog.Nop()
	repo := newRateLimitRepository(&sharedRepository{
		pool:    pool,
		l:       &logger,
		queries: sqlcv1.New(),
	})

	tests := []struct {
		name          string
		lastRefillAgo time.Duration
		units         int
		wantValue     int32
		wantRefreshed bool
	}{
		{
			// prior-window usage must not consume the new window's budget
			name:          "expired window resets to full limit",
			lastRefillAgo: 2 * time.Minute,
			units:         10,
			wantValue:     10,
			wantRefreshed: true,
		},
		{
			name:          "active window is charged without refill",
			lastRefillAgo: 10 * time.Second,
			units:         4,
			wantValue:     6,
			wantRefreshed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			tenantID := createLimitTestTenant(t, pool)
			key := "test-key"

			var lastRefillBefore time.Time
			err := pool.QueryRow(ctx, `
				INSERT INTO "RateLimit" ("tenantId", "key", "limitValue", "value", "window", "lastRefill")
				VALUES ($1, $2, 10, 10, '1 minute', NOW() - make_interval(secs => $3))
				RETURNING "lastRefill"
			`, tenantID, key, tt.lastRefillAgo.Seconds()).Scan(&lastRefillBefore)
			require.NoError(t, err)

			rows, _, err := repo.UpdateRateLimits(ctx, tenantID, map[string]int{key: tt.units})
			require.NoError(t, err)
			require.Len(t, rows, 1)
			assert.Equal(t, tt.wantValue, rows[0].Value)

			var value int32
			var lastRefillAfter time.Time
			err = pool.QueryRow(ctx, `
				SELECT "value", "lastRefill" FROM "RateLimit" WHERE "tenantId" = $1 AND "key" = $2
			`, tenantID, key).Scan(&value, &lastRefillAfter)
			require.NoError(t, err)

			assert.Equal(t, tt.wantValue, value)
			if tt.wantRefreshed {
				assert.True(t, lastRefillAfter.After(lastRefillBefore), "lastRefill should advance on refill")
			} else {
				assert.True(t, lastRefillAfter.Equal(lastRefillBefore), "lastRefill should be unchanged within the window")
			}
		})
	}
}

func TestGetTaskRateLimits_OptimisticTxDoesNotWriteRateLimits(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	logger := zerolog.Nop()
	shared := &sharedRepository{
		pool:    pool,
		l:       &logger,
		queries: sqlcv1.New(),
	}
	rateLimitRepo := newRateLimitRepository(shared)

	t.Run("new dynamic key is not written and does not block refresh", func(t *testing.T) {
		ctx := context.Background()
		tenantID := createLimitTestTenant(t, pool)
		queueRepo := newQueueRepository(shared, tenantID, "default")
		t.Cleanup(queueRepo.Cleanup)

		qi := insertDynamicRateLimitEvals(t, pool, tenantID, 1001, "new-key", 10)

		tx, err := shared.PrepareOptimisticTx(ctx)
		require.NoError(t, err)
		defer tx.Rollback()

		units, err := queueRepo.GetTaskRateLimits(ctx, tx, []*sqlcv1.V1QueueItem{qi})
		require.NoError(t, err)
		require.Equal(t, int32(1), units[qi.TaskID]["new-key"])

		// mirrors rateLimiter.use refreshing via a separate transaction while the optimistic tx is still open
		refreshCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		defer cancel()

		_, _, err = rateLimitRepo.UpdateRateLimits(refreshCtx, tenantID, map[string]int{})
		require.NoError(t, err, "rate limit refresh blocked on the optimistic transaction")

		tx.Rollback()
		assert.False(t, rateLimitExists(t, pool, tenantID, "new-key"), "rolled back trigger must not leave a rate limit definition")
	})

	t.Run("existing key is not overwritten or locked", func(t *testing.T) {
		ctx := context.Background()
		tenantID := createLimitTestTenant(t, pool)
		queueRepo := newQueueRepository(shared, tenantID, "default")
		t.Cleanup(queueRepo.Cleanup)

		_, err := pool.Exec(ctx, `
			INSERT INTO "RateLimit" ("tenantId", "key", "limitValue", "value", "window", "lastRefill")
			VALUES ($1, 'shared-key', 10, 10, '1 MINUTE', NOW() - INTERVAL '2 minutes')
		`, tenantID)
		require.NoError(t, err)

		// the unassigned task asks for a different limit than the one other tasks share
		qi := insertDynamicRateLimitEvals(t, pool, tenantID, 1002, "shared-key", 5)

		tx, err := shared.PrepareOptimisticTx(ctx)
		require.NoError(t, err)
		defer tx.Rollback()

		_, err = queueRepo.GetTaskRateLimits(ctx, tx, []*sqlcv1.V1QueueItem{qi})
		require.NoError(t, err)

		refreshCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		defer cancel()

		rows, _, err := rateLimitRepo.UpdateRateLimits(refreshCtx, tenantID, map[string]int{"shared-key": 1})
		require.NoError(t, err, "rate limit refresh blocked on the optimistic transaction")
		require.Len(t, rows, 1)
		assert.Equal(t, int32(10), rows[0].LimitValue)
		assert.Equal(t, int32(10), rows[0].Value)
	})

	t.Run("queue loop upserts the definition", func(t *testing.T) {
		ctx := context.Background()
		tenantID := createLimitTestTenant(t, pool)
		queueRepo := newQueueRepository(shared, tenantID, "default")
		t.Cleanup(queueRepo.Cleanup)

		qi := insertDynamicRateLimitEvals(t, pool, tenantID, 1003, "loop-key", 10)

		_, err := queueRepo.GetTaskRateLimits(ctx, nil, []*sqlcv1.V1QueueItem{qi})
		require.NoError(t, err)
		assert.True(t, rateLimitExists(t, pool, tenantID, "loop-key"))
	})
}

func TestGetTaskRateLimits_OptimisticTxNeedsNoSecondConnection(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()
	tenantID := createLimitTestTenant(t, pool)
	qi := insertDynamicRateLimitEvals(t, pool, tenantID, 2001, "pool-key", 10)

	// a single-connection pool: any second checkout while the optimistic tx is open would hang
	cfg := pool.Config().Copy()
	cfg.MaxConns = 1
	cfg.MinConns = 0
	onePool, err := pgxpool.NewWithConfig(ctx, cfg)
	require.NoError(t, err)
	defer onePool.Close()

	logger := zerolog.Nop()
	shared := &sharedRepository{
		pool:    onePool,
		l:       &logger,
		queries: sqlcv1.New(),
	}
	queueRepo := newQueueRepository(shared, tenantID, "default")
	defer queueRepo.Cleanup()

	tx, err := shared.PrepareOptimisticTx(ctx)
	require.NoError(t, err)
	defer tx.Rollback()

	callCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	_, err = queueRepo.GetTaskRateLimits(callCtx, tx, []*sqlcv1.V1QueueItem{qi})
	require.NoError(t, err, "optimistic GetTaskRateLimits must not check out a second connection")

	require.NoError(t, tx.Commit(ctx))
}

func insertDynamicRateLimitEvals(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID, taskID int64, key string, limit int32) *sqlcv1.V1QueueItem {
	t.Helper()

	qi := &sqlcv1.V1QueueItem{
		TenantID:       tenantID,
		TaskID:         taskID,
		TaskInsertedAt: pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
		StepID:         uuid.New(),
	}

	for _, eval := range []struct {
		kind     sqlcv1.StepExpressionKind
		valueStr *string
		valueInt *int32
	}{
		{kind: sqlcv1.StepExpressionKindDYNAMICRATELIMITKEY, valueStr: strPtr(key)},
		{kind: sqlcv1.StepExpressionKindDYNAMICRATELIMITVALUE, valueInt: int32Ptr(limit)},
		{kind: sqlcv1.StepExpressionKindDYNAMICRATELIMITWINDOW, valueStr: strPtr("MINUTE")},
		{kind: sqlcv1.StepExpressionKindDYNAMICRATELIMITUNITS, valueInt: int32Ptr(1)},
	} {
		_, err := pool.Exec(context.Background(), `
			INSERT INTO v1_task_expression_eval (key, task_id, task_inserted_at, value_str, value_int, kind)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, "rl", qi.TaskID, qi.TaskInsertedAt, eval.valueStr, eval.valueInt, string(eval.kind))
		require.NoError(t, err)
	}

	return qi
}

func rateLimitExists(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID, key string) bool {
	t.Helper()

	var exists bool
	err := pool.QueryRow(context.Background(), `
		SELECT EXISTS (SELECT 1 FROM "RateLimit" WHERE "tenantId" = $1 AND "key" = $2)
	`, tenantID, key).Scan(&exists)
	require.NoError(t, err)

	return exists
}
