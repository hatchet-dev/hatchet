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

	"github.com/hatchet-dev/hatchet/pkg/repository/fairpool"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// A scheduler subscribes to the wake-up topic of the partition id it started with. When every
// partition goes stale at once (the database was unreachable for longer than the staleness
// threshold), a rebalance deletes the partition rows and unassigns their tenants. The next heartbeat
// must bring the partition back under the same id, so the tenants are reassigned to the topic the
// scheduler is still subscribed to.
func TestSchedulerPartitionHeartbeatKeepsIdAfterInactiveRebalance(t *testing.T) {
	t.Parallel()

	pool, cleanup := setupPostgresWithMigration(t)
	t.Cleanup(cleanup)

	logger := zerolog.New(io.Discard)
	repo := newTenantRepository(&sharedRepository{
		pool:    fairpool.Ungated(pool),
		l:       &logger,
		queries: sqlcv1.New(),
	}, time.Minute)

	ctx := context.Background()

	partitionId, err := repo.CreateSchedulerPartition(ctx)
	require.NoError(t, err)

	tenantId := uuid.New()
	slug := "tenant-" + strings.ReplaceAll(tenantId.String(), "-", "")

	_, err = pool.Exec(ctx, `INSERT INTO "Tenant" ("id", "name", "slug", "version") VALUES ($1, $2, $3, 'V1')`, tenantId, "Test Tenant", slug)
	require.NoError(t, err)

	require.NoError(t, repo.RebalanceAllSchedulerPartitions(ctx))
	require.Equal(t, partitionId, tenantSchedulerPartitionId(t, pool, tenantId).String)

	_, err = pool.Exec(ctx, `UPDATE "SchedulerPartition" SET "lastHeartbeat" = NOW() - INTERVAL '1 minute' WHERE "id" = $1`, partitionId)
	require.NoError(t, err)

	require.NoError(t, repo.RebalanceInactiveSchedulerPartitions(ctx))
	require.False(t, tenantSchedulerPartitionId(t, pool, tenantId).Valid, "the inactive rebalance should unassign the tenant")

	require.NoError(t, repo.UpdateSchedulerPartitionHeartbeat(ctx, partitionId))
	require.NoError(t, repo.RebalanceInactiveSchedulerPartitions(ctx))

	assert.Equal(t, partitionId, tenantSchedulerPartitionId(t, pool, tenantId).String, "the tenant should be back on the partition id the scheduler subscribed to")

	tenants, err := repo.ListTenantsBySchedulerPartition(ctx, partitionId)
	require.NoError(t, err)
	require.Len(t, tenants, 1)
	assert.Equal(t, tenantId, tenants[0].ID)

	var partitions int
	require.NoError(t, pool.QueryRow(ctx, `SELECT COUNT(*) FROM "SchedulerPartition"`).Scan(&partitions))
	assert.Equal(t, 1, partitions, "the heartbeat should recreate the partition, not add another one")
}

func tenantSchedulerPartitionId(t *testing.T, pool *pgxpool.Pool, tenantId uuid.UUID) pgtype.Text {
	t.Helper()

	var partitionId pgtype.Text
	require.NoError(t, pool.QueryRow(context.Background(), `SELECT "schedulerPartitionId" FROM "Tenant" WHERE "id" = $1`, tenantId).Scan(&partitionId))

	return partitionId
}
