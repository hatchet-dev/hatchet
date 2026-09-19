//go:build !e2e && !load && !rampup && !integration

package lease

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/repository"
)

// leaseSchemaPool clones the lease tables into a schema of their own on the database
// DATABASE_URL names, migrated to the current version, and returns a pool whose search path
// starts there. The test is skipped without DATABASE_URL.
func leaseSchemaPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	dsn := os.Getenv("DATABASE_URL")

	if dsn == "" {
		t.Skip("DATABASE_URL is not set")
	}

	ctx := context.Background()

	admin, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)

	schema := "lease_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	ident := pgx.Identifier{schema}.Sanitize()

	_, err = admin.Exec(ctx, "CREATE SCHEMA "+ident)
	require.NoError(t, err)

	t.Cleanup(func() {
		_, err := admin.Exec(ctx, "DROP SCHEMA "+ident+" CASCADE")
		require.NoError(t, err)
		admin.Close()
	})

	for _, name := range []string{"v1_serverless_process", "v1_serverless_lease"} {
		_, err := admin.Exec(ctx, "CREATE TABLE "+ident+"."+name+" (LIKE public."+name+" INCLUDING ALL)")
		require.NoError(t, err)
	}

	cfg, err := pgxpool.ParseConfig(dsn)
	require.NoError(t, err)

	cfg.ConnConfig.RuntimeParams["search_path"] = schema + ",public"

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	require.NoError(t, err)

	t.Cleanup(pool.Close)

	return pool
}

// Round 2 correctness R08, over the production leaser and SQL: a count window of empty units
// ahead of a populated one must not strand it. Four empty unowned units sort before a unit of
// three endpoints and the process, holding one endpoint, is alone; with MaxClaimPerTick 4 the
// old count sampled only the empties, reported a claimable weight of zero, and the process
// never claimed anything.
func TestZeroWeightWindowDoesNotStrandBacklog(t *testing.T) {
	pool := leaseSchemaPool(t)
	ctx := context.Background()

	l := zerolog.Nop()
	repo, cleanup := repository.NewServerlessRepositoryFromPool(pool, &l)

	t.Cleanup(func() { _ = cleanup() })

	owner := uuid.New()
	rec := &fakeReconciler{}
	s := New(repo, rec, Config{ProcessId: owner, MaxClaimPerTick: 4}, &l, Hooks{})

	require.NoError(t, s.Heartbeat(ctx))

	for i := 0; i < 6; i++ {
		tenant := fmt.Sprintf("00000000-0000-0000-0000-%012d", i+1)

		var holder *uuid.UUID

		weight := 0

		if i == 0 {
			holder = &owner
			weight = 1
		}

		if i == 5 {
			weight = 3
		}

		_, err := pool.Exec(ctx, "INSERT INTO v1_serverless_lease (tenant_id, shard, process_id, endpoint_count) VALUES ($1, 0, $2, $3)", tenant, holder, weight)
		require.NoError(t, err)
	}

	sampled, err := repo.Leases().CountClaimable(ctx, 4)
	require.NoError(t, err)
	require.Equal(t, int64(1), sampled.UnitCount, "empty units are not in the sample")
	require.Equal(t, int64(3), sampled.EndpointCount)

	for i := 0; i < 3; i++ {
		require.NoError(t, s.Tick(ctx))
		require.NoError(t, s.Heartbeat(ctx))
	}

	var unowned, weight int

	require.NoError(t, pool.QueryRow(ctx, "SELECT count(*), sum(endpoint_count) FROM v1_serverless_lease WHERE process_id IS NULL").Scan(&unowned, &weight))

	require.Len(t, s.Owned(), 2, "the populated unit behind the empty window is claimed")
	require.Equal(t, 4, unowned, "the empty units stay unowned")
	require.Equal(t, 0, weight)
}
