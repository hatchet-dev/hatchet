//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddStreamBytesSumsConcurrentFlushes(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()
	repo := createOLAPRepositoryManagingAllPartitions(t, pool)
	require.NoError(t, repo.UpdateTablePartitions(ctx))

	tenantA, tenantB, idle := uuid.New(), uuid.New(), uuid.New()

	// every engine and API process flushes its own totals, in no particular tenant order
	flushes := [][]TenantStreamBytes{
		{{TenantID: tenantA, PublishedBytes: 10, DeliveredBytes: 20}, {TenantID: tenantB, PublishedBytes: 1}},
		{{TenantID: tenantB, DeliveredBytes: 2}, {TenantID: tenantA, PublishedBytes: 5}, {TenantID: idle}},
	}

	var wg sync.WaitGroup

	for range 10 {
		for _, flush := range flushes {
			wg.Go(func() {
				assert.NoError(t, repo.AddStreamBytes(ctx, flush))
			})
		}
	}

	wg.Wait()

	totals := func(tenantId uuid.UUID) (published, delivered int64) {
		require.NoError(t, pool.QueryRow(ctx, `
SELECT COALESCE(SUM(published_bytes), 0), COALESCE(SUM(delivered_bytes), 0)
FROM v1_agg_stream_bytes_minute_olap
WHERE tenant_id = $1`, tenantId).Scan(&published, &delivered))

		return published, delivered
	}

	published, delivered := totals(tenantA)
	assert.Equal(t, int64(150), published)
	assert.Equal(t, int64(200), delivered)

	published, delivered = totals(tenantB)
	assert.Equal(t, int64(10), published)
	assert.Equal(t, int64(20), delivered)

	var idleRows int
	require.NoError(t, pool.QueryRow(ctx, `SELECT COUNT(*) FROM v1_agg_stream_bytes_minute_olap WHERE tenant_id = $1`, idle).Scan(&idleRows))
	assert.Zero(t, idleRows, "a tenant with no bytes gets no row")
}
