//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestFlushWorkerHeartbeats(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()

	repo := newBatchTestRepository(pool)
	workers := newWorkerRepository(repo.sharedRepository)

	otherTenantId := uuid.New()
	_, err := pool.Exec(ctx, `INSERT INTO "Tenant" ("id", "name", "slug") VALUES ($1, $2, $3)`, otherTenantId, "Other Tenant", "flush-hb-other-tenant")
	require.NoError(t, err)

	base := time.Now().UTC().Truncate(time.Second)

	nullWorker := uuid.New()
	staleWorker := uuid.New()
	freshWorker := uuid.New()
	otherTenantWorker := uuid.New()

	_, err = pool.Exec(ctx, `
		INSERT INTO "Worker" ("id", "tenantId", "name", "lastHeartbeatAt", "createdAt", "updatedAt")
		VALUES
			($1, $5, 'null-worker', NULL, NOW(), NOW()),
			($2, $5, 'stale-worker', $7, NOW(), NOW()),
			($3, $5, 'fresh-worker', $8, NOW(), NOW()),
			($4, $6, 'other-tenant-worker', $7, NOW(), NOW())
	`,
		nullWorker,
		staleWorker,
		freshWorker,
		otherTenantWorker,
		internalTenantId,
		otherTenantId,
		base.Add(-45*time.Second),
		base.Add(-2*time.Second),
	)
	require.NoError(t, err)

	nullReceivedAt := base.Add(-10 * time.Second)
	staleReceivedAt := base.Add(-6 * time.Second)
	olderThanFresh := base.Add(-20 * time.Second)
	otherTenantReceivedAt := base.Add(-4 * time.Second)

	require.NoError(t, workers.FlushWorkerHeartbeats(ctx, nil))

	require.NoError(t, workers.FlushWorkerHeartbeats(ctx, []WorkerHeartbeat{
		{TenantId: internalTenantId, WorkerId: nullWorker, LastHeartbeatAt: nullReceivedAt},
		{TenantId: internalTenantId, WorkerId: staleWorker, LastHeartbeatAt: staleReceivedAt},
		{TenantId: internalTenantId, WorkerId: freshWorker, LastHeartbeatAt: olderThanFresh},
		// Mismatched tenant must not touch another tenant's worker.
		{TenantId: internalTenantId, WorkerId: otherTenantWorker, LastHeartbeatAt: otherTenantReceivedAt},
		// Unknown worker id must not fail the batch.
		{TenantId: internalTenantId, WorkerId: uuid.New(), LastHeartbeatAt: base},
	}))

	readHeartbeat := func(workerId uuid.UUID) time.Time {
		t.Helper()

		var ts time.Time
		err := pool.QueryRow(ctx, `SELECT "lastHeartbeatAt" FROM "Worker" WHERE "id" = $1`, workerId).Scan(&ts)
		require.NoError(t, err)

		return ts.UTC()
	}

	require.Equal(t, nullReceivedAt, readHeartbeat(nullWorker), "NULL lastHeartbeatAt should be set to the received time")
	require.Equal(t, staleReceivedAt, readHeartbeat(staleWorker), "older lastHeartbeatAt should advance to the worker's own received time")
	require.Equal(t, base.Add(-2*time.Second), readHeartbeat(freshWorker), "flush must not move a newer recorded heartbeat backwards")
	require.Equal(t, base.Add(-45*time.Second), readHeartbeat(otherTenantWorker), "flush must be scoped by tenant")

	// With the owning tenant, the other tenant's worker advances as expected.
	require.NoError(t, workers.FlushWorkerHeartbeats(ctx, []WorkerHeartbeat{
		{TenantId: otherTenantId, WorkerId: otherTenantWorker, LastHeartbeatAt: otherTenantReceivedAt},
	}))
	require.Equal(t, otherTenantReceivedAt, readHeartbeat(otherTenantWorker))
}
