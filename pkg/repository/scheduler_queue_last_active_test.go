//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
	"io"
	"sort"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/repository/cache"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func TestIdleQueuesGoInactive(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	tenantID := uuid.New()
	queries := sqlcv1.New()

	_, err := pool.Exec(ctx, `
		INSERT INTO v1_queue (tenant_id, name, last_active)
		VALUES ($1, 'idle', NOW() - INTERVAL '2 days'), ($1, 'rate-limited', NOW() - INTERVAL '2 days')
	`, tenantID)
	require.NoError(t, err)

	insertRateLimitedQueueItem(t, pool, tenantID, "rate-limited", 1, "-1 minute")

	logger := zerolog.New(io.Discard)
	shared := &sharedRepository{pool: pool, l: &logger, queries: queries, queueCache: cache.New(5 * time.Minute)}
	repo := newQueueRepository(shared, tenantID, "idle")

	rows, err := repo.RequeueRateLimitedItems(ctx, tenantID, "idle")
	require.NoError(t, err)
	require.Empty(t, rows)

	require.Empty(t, listQueueNames(t, pool, tenantID), "an empty requeue must not reactivate the queue")

	_, err = queries.ReactivateInactiveQueuesWithItems(ctx, pool)
	require.NoError(t, err)
	require.Equal(t, []string{"rate-limited"}, listQueueNames(t, pool, tenantID))
}

func TestReactivateInactiveQueuesWithItems(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	queries := sqlcv1.New()
	tenants := []uuid.UUID{uuid.New(), uuid.New()}

	var taskID int64

	for _, tenantID := range tenants {
		_, err := pool.Exec(ctx, `
			INSERT INTO v1_queue (tenant_id, name, last_active)
			VALUES
				($1, 'ready-active', NOW()),
				($1, 'ready-inactive', NOW() - INTERVAL '2 days'),
				($1, 'parked-due', NOW() - INTERVAL '2 days'),
				($1, 'parked-mixed', NOW() - INTERVAL '2 days'),
				($1, 'parked-future', NOW() - INTERVAL '2 days'),
				($1, 'empty', NOW() - INTERVAL '2 days')
		`, tenantID)
		require.NoError(t, err)

		for _, queue := range []string{"ready-active", "ready-inactive", "ready-inactive"} {
			taskID++
			_, err = pool.Exec(ctx, `
				INSERT INTO v1_queue_item (
					tenant_id, queue, task_id, task_inserted_at, external_id, action_id, step_id,
					workflow_id, workflow_run_id, sticky
				)
				VALUES ($1, $2, $3, NOW(), gen_random_uuid(), $2, gen_random_uuid(), gen_random_uuid(),
					gen_random_uuid(), 'NONE')
			`, tenantID, queue, taskID)
			require.NoError(t, err)
		}

		for _, item := range []struct{ queue, requeueIn string }{
			{"parked-due", "-1 minute"},
			{"parked-mixed", "3 days"},
			{"parked-mixed", "-1 minute"},
			{"parked-future", "3 days"},
			{"parked-future", "1 hour"},
		} {
			taskID++
			insertRateLimitedQueueItem(t, pool, tenantID, item.queue, taskID, item.requeueIn)
		}
	}

	result, err := queries.ReactivateInactiveQueuesWithItems(ctx, pool)
	require.NoError(t, err)
	require.EqualValues(t, 3*len(tenants), result.RowsAffected())

	for _, tenantID := range tenants {
		require.Equal(t, []string{"parked-due", "parked-mixed", "ready-active", "ready-inactive"}, listQueueNames(t, pool, tenantID),
			"queues whose parked items are not yet due, or that have no items, stay inactive")
	}

	result, err = queries.ReactivateInactiveQueuesWithItems(ctx, pool)
	require.NoError(t, err)
	require.Zero(t, result.RowsAffected(), "active queues are not touched again")

	_, err = pool.Exec(ctx, `
		UPDATE v1_rate_limited_queue_items
		SET requeue_after = NOW() - INTERVAL '1 second'
		WHERE queue = 'parked-future' AND requeue_after < NOW() + INTERVAL '1 day'
	`)
	require.NoError(t, err)

	result, err = queries.ReactivateInactiveQueuesWithItems(ctx, pool)
	require.NoError(t, err)
	require.EqualValues(t, len(tenants), result.RowsAffected())

	for _, tenantID := range tenants {
		require.Contains(t, listQueueNames(t, pool, tenantID), "parked-future",
			"a queue is reactivated once one of its parked items is due")
	}
}

func insertRateLimitedQueueItem(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID, queue string, taskID int64, requeueIn string) {
	t.Helper()

	_, err := pool.Exec(context.Background(), `
		INSERT INTO v1_rate_limited_queue_items (
			requeue_after, tenant_id, queue, task_id, task_inserted_at, external_id, action_id,
			step_id, workflow_id, workflow_run_id, sticky
		)
		VALUES (NOW() + $4::interval, $1, $2, $3, NOW(), gen_random_uuid(), $2,
			gen_random_uuid(), gen_random_uuid(), gen_random_uuid(), 'NONE')
	`, tenantID, queue, taskID, requeueIn)
	require.NoError(t, err)
}

func listQueueNames(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID) []string {
	t.Helper()

	queues, err := sqlcv1.New().ListQueues(context.Background(), pool, tenantID)
	require.NoError(t, err)

	names := make([]string, 0, len(queues))
	for _, q := range queues {
		names = append(names, q.Name)
	}

	sort.Strings(names)

	return names
}
