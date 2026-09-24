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
	"github.com/hatchet-dev/hatchet/pkg/validator"
)

func TestUpdateRateLimits_ChargesUsageBeforeRefilling(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	repo := newRateLimitRepository(newRateLimitTestShared(pool))

	tests := []struct {
		name          string
		duration      string
		waitForExpiry bool
		units         int
		wantValue     int32
		wantRefreshed bool
	}{
		{
			// prior-window usage must not consume the new window's budget
			name:          "expired window resets to full limit",
			duration:      "SECOND",
			waitForExpiry: true,
			units:         10,
			wantValue:     10,
			wantRefreshed: true,
		},
		{
			name:          "active window is charged without refill",
			duration:      "MINUTE",
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

			created, err := repo.UpsertRateLimit(ctx, tenantID, key, &UpsertRateLimitOpts{Limit: 10, Duration: &tt.duration})
			require.NoError(t, err)

			if tt.waitForExpiry {
				time.Sleep(1100 * time.Millisecond)
			}

			rows, _, err := repo.UpdateRateLimits(ctx, tenantID, map[string]int{key: tt.units}, nil)
			require.NoError(t, err)
			require.Len(t, rows, 1)
			assert.Equal(t, tt.wantValue, rows[0].Value)

			if tt.wantRefreshed {
				assert.True(t, rows[0].LastRefill.Time.After(created.LastRefill.Time), "lastRefill should advance on refill")
			} else {
				assert.True(t, rows[0].LastRefill.Time.Equal(created.LastRefill.Time), "lastRefill should be unchanged within the window")
			}
		})
	}
}

func TestGetTaskRateLimits_DoesNotWriteRateLimits(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	shared := newRateLimitTestShared(pool)
	rateLimitRepo := newRateLimitRepository(shared)

	for _, optimistic := range []bool{false, true} {
		name := "queue loop"
		if optimistic {
			name = "optimistic"
		}

		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			tenantID := createLimitTestTenant(t, pool)
			queueRepo := newQueueRepository(shared, tenantID, "default")
			t.Cleanup(queueRepo.Cleanup)

			qi := createDynamicRateLimitEvals(t, shared, tenantID, 1001, "dynamic-key", 10)

			var tx *OptimisticTx
			if optimistic {
				var err error
				tx, err = shared.PrepareOptimisticTx(ctx)
				require.NoError(t, err)
				defer tx.Rollback()
			}

			units, definitions, err := queueRepo.GetTaskRateLimits(ctx, tx, []*sqlcv1.V1QueueItem{qi})
			require.NoError(t, err)
			assert.Equal(t, int32(1), units[qi.TaskID]["dynamic-key"])
			assert.Equal(t, map[string]RateLimitDefinition{"dynamic-key": {LimitValue: 10, Window: "1 MINUTE"}}, definitions)

			// mirrors rateLimiter.use refreshing via a separate transaction while an optimistic tx may still be open
			refreshCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
			defer cancel()

			_, _, err = rateLimitRepo.UpdateRateLimits(refreshCtx, tenantID, map[string]int{}, nil)
			require.NoError(t, err, "rate limit refresh blocked on the caller's transaction")

			assert.Empty(t, listRateLimitsByKey(t, rateLimitRepo, tenantID, "dynamic-key"), "definitions are only written by UpdateRateLimits")
		})
	}
}

func TestUpdateRateLimits_UpsertsDefinitions(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	rateLimitRepo := newRateLimitRepository(newRateLimitTestShared(pool))
	minute := "MINUTE"

	t.Run("new key is created before usage is charged", func(t *testing.T) {
		ctx := context.Background()
		tenantID := createLimitTestTenant(t, pool)

		rows, _, err := rateLimitRepo.UpdateRateLimits(ctx, tenantID,
			map[string]int{"new-key": 1},
			map[string]RateLimitDefinition{"new-key": {LimitValue: 10, Window: "1 MINUTE"}},
		)
		require.NoError(t, err)
		require.Len(t, rows, 1)
		assert.Equal(t, int32(10), rows[0].LimitValue)
		assert.Equal(t, int32(9), rows[0].Value)
	})

	t.Run("changed definition lowers the limit", func(t *testing.T) {
		ctx := context.Background()
		tenantID := createLimitTestTenant(t, pool)

		_, err := rateLimitRepo.UpsertRateLimit(ctx, tenantID, "shared-key", &UpsertRateLimitOpts{Limit: 10, Duration: &minute})
		require.NoError(t, err)

		rows, _, err := rateLimitRepo.UpdateRateLimits(ctx, tenantID,
			map[string]int{},
			map[string]RateLimitDefinition{"shared-key": {LimitValue: 5, Window: "1 MINUTE"}},
		)
		require.NoError(t, err)
		require.Len(t, rows, 1)
		assert.Equal(t, int32(5), rows[0].LimitValue)
		assert.Equal(t, int32(5), rows[0].Value)
	})
}

func TestGetTaskRateLimits_OptimisticTxNeedsNoSecondConnection(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()
	tenantID := createLimitTestTenant(t, pool)
	qi := createDynamicRateLimitEvals(t, newRateLimitTestShared(pool), tenantID, 2001, "pool-key", 10)

	// a single-connection pool: any second checkout while the optimistic tx is open would hang
	cfg := pool.Config().Copy()
	cfg.MaxConns = 1
	cfg.MinConns = 0
	onePool, err := pgxpool.NewWithConfig(ctx, cfg)
	require.NoError(t, err)
	defer onePool.Close()

	shared := newRateLimitTestShared(onePool)
	queueRepo := newQueueRepository(shared, tenantID, "default")
	defer queueRepo.Cleanup()

	tx, err := shared.PrepareOptimisticTx(ctx)
	require.NoError(t, err)
	defer tx.Rollback()

	callCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	_, _, err = queueRepo.GetTaskRateLimits(callCtx, tx, []*sqlcv1.V1QueueItem{qi})
	require.NoError(t, err, "optimistic GetTaskRateLimits must not check out a second connection")

	require.NoError(t, tx.Commit(ctx))
}

func newRateLimitTestShared(pool *pgxpool.Pool) *sharedRepository {
	logger := zerolog.Nop()

	return &sharedRepository{
		pool:    pool,
		l:       &logger,
		v:       validator.NewDefaultValidator(),
		queries: sqlcv1.New(),
	}
}

func createDynamicRateLimitEvals(t *testing.T, shared *sharedRepository, tenantID uuid.UUID, taskID int64, key string, limit int) *sqlcv1.V1QueueItem {
	t.Helper()

	task := &V1TaskWithPayload{V1Task: &sqlcv1.V1Task{
		ID:         taskID,
		InsertedAt: pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
		ExternalID: uuid.New(),
	}}

	window := "MINUTE"
	units := 1

	err := shared.createExpressionEvals(context.Background(), shared.pool, []*V1TaskWithPayload{task}, map[uuid.UUID][]createTaskExpressionEvalOpt{
		task.ExternalID: {
			{Key: "rl", Kind: sqlcv1.StepExpressionKindDYNAMICRATELIMITKEY, ValueStr: &key},
			{Key: "rl", Kind: sqlcv1.StepExpressionKindDYNAMICRATELIMITVALUE, ValueInt: &limit},
			{Key: "rl", Kind: sqlcv1.StepExpressionKindDYNAMICRATELIMITWINDOW, ValueStr: &window},
			{Key: "rl", Kind: sqlcv1.StepExpressionKindDYNAMICRATELIMITUNITS, ValueInt: &units},
		},
	})
	require.NoError(t, err)

	return &sqlcv1.V1QueueItem{
		TenantID:       tenantID,
		TaskID:         task.ID,
		TaskInsertedAt: task.InsertedAt,
		StepID:         uuid.New(),
	}
}

func listRateLimitsByKey(t *testing.T, repo *rateLimitRepository, tenantID uuid.UUID, key string) []*sqlcv1.ListRateLimitsForTenantNoMutateRow {
	t.Helper()

	res, err := repo.ListRateLimits(context.Background(), tenantID, &ListRateLimitOpts{Search: &key})
	require.NoError(t, err)

	return res.Rows
}
