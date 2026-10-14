//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/repository/cache"
	"github.com/hatchet-dev/hatchet/pkg/repository/fairpool"
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

	_, err = pool.Exec(ctx, `
		INSERT INTO v1_rate_limited_queue_items (
			requeue_after, tenant_id, queue, task_id, task_inserted_at, external_id, action_id,
			step_id, workflow_id, workflow_run_id, sticky
		)
		VALUES (NOW() + INTERVAL '1 hour', $1, 'rate-limited', 1, NOW(), gen_random_uuid(), 'a',
			gen_random_uuid(), gen_random_uuid(), gen_random_uuid(), 'NONE')
	`, tenantID)
	require.NoError(t, err)

	logger := zerolog.New(io.Discard)
	shared := &sharedRepository{pool: fairpool.Ungated(pool), l: &logger, queries: queries, queueCache: cache.New(5 * time.Minute)}
	repo := newQueueRepository(shared, tenantID, "idle")

	rows, err := repo.RequeueRateLimitedItems(ctx, tenantID, "idle")
	require.NoError(t, err)
	require.Empty(t, rows)

	listed := func() []string {
		queues, err := queries.ListQueues(ctx, pool, tenantID)
		require.NoError(t, err)

		names := make([]string, 0, len(queues))
		for _, q := range queues {
			names = append(names, q.Name)
		}

		return names
	}

	require.Empty(t, listed(), "an empty requeue must not reactivate the queue")

	_, err = queries.ReactivateInactiveQueuesWithItems(ctx, pool)
	require.NoError(t, err)
	require.Equal(t, []string{"rate-limited"}, listed())
}
