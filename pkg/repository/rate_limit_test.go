//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
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

func TestGetTaskRateLimits_OptimisticTxDoesNotBlockRateLimitRefresh(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	logger := zerolog.Nop()
	shared := &sharedRepository{
		pool:    pool,
		l:       &logger,
		queries: sqlcv1.New(),
	}
	rateLimitRepo := newRateLimitRepository(shared)

	tests := []struct {
		name          string
		existing      bool
		lastRefillAgo time.Duration
		wantValue     int32
	}{
		{
			name:      "new dynamic key",
			wantValue: 9,
		},
		{
			// the upsert's ON CONFLICT row lock must not block the refill either
			name:          "existing dynamic key with expired window",
			existing:      true,
			lastRefillAgo: 2 * time.Minute,
			wantValue:     10,
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			tenantID := createLimitTestTenant(t, pool)
			key := "dynamic-key"

			queueRepo := newQueueRepository(shared, tenantID, "default")
			t.Cleanup(queueRepo.Cleanup)

			if tt.existing {
				_, err := pool.Exec(ctx, `
					INSERT INTO "RateLimit" ("tenantId", "key", "limitValue", "value", "window", "lastRefill")
					VALUES ($1, $2, 10, 10, '1 MINUTE', NOW() - make_interval(secs => $3))
				`, tenantID, key, tt.lastRefillAgo.Seconds())
				require.NoError(t, err)
			}

			qi := &sqlcv1.V1QueueItem{
				TenantID:       tenantID,
				TaskID:         int64(1000 + i),
				TaskInsertedAt: pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
				StepID:         uuid.New(),
			}

			for _, eval := range []struct {
				kind     sqlcv1.StepExpressionKind
				valueStr *string
				valueInt *int32
			}{
				{kind: sqlcv1.StepExpressionKindDYNAMICRATELIMITKEY, valueStr: strPtr(key)},
				{kind: sqlcv1.StepExpressionKindDYNAMICRATELIMITVALUE, valueInt: int32Ptr(10)},
				{kind: sqlcv1.StepExpressionKindDYNAMICRATELIMITWINDOW, valueStr: strPtr("MINUTE")},
				{kind: sqlcv1.StepExpressionKindDYNAMICRATELIMITUNITS, valueInt: int32Ptr(1)},
			} {
				_, err := pool.Exec(ctx, `
					INSERT INTO v1_task_expression_eval (key, task_id, task_inserted_at, value_str, value_int, kind)
					VALUES ($1, $2, $3, $4, $5, $6)
				`, "rl", qi.TaskID, qi.TaskInsertedAt, eval.valueStr, eval.valueInt, string(eval.kind))
				require.NoError(t, err)
			}

			tx, err := shared.PrepareOptimisticTx(ctx)
			require.NoError(t, err)
			defer tx.Rollback()

			units, err := queueRepo.GetTaskRateLimits(ctx, tx, []*sqlcv1.V1QueueItem{qi})
			require.NoError(t, err)
			require.Equal(t, int32(1), units[qi.TaskID][key])

			// mirrors rateLimiter.use refreshing via a separate transaction while the optimistic tx is still open
			refreshCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
			defer cancel()

			rows, _, err := rateLimitRepo.UpdateRateLimits(refreshCtx, tenantID, map[string]int{key: 1})
			require.NoError(t, err, "rate limit refresh blocked on the optimistic transaction")

			var got *sqlcv1.ListRateLimitsForTenantWithMutateRow
			for _, row := range rows {
				if row.Key == key {
					got = row
				}
			}

			require.NotNil(t, got, "dynamic key should be visible to the refresh before the optimistic tx commits")
			assert.Equal(t, tt.wantValue, got.Value)

			require.NoError(t, tx.Commit(ctx))
		})
	}
}
