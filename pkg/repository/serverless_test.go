//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

const pgUniqueViolation = "23505"

func newServerlessTestRepository(t *testing.T, pool *pgxpool.Pool) ServerlessRepository {
	t.Helper()

	logger := zerolog.Nop()
	repo, cleanup := NewServerlessRepositoryFromPool(pool, &logger)
	t.Cleanup(func() { _ = cleanup() })

	return repo
}

func serverlessEndpointOpts(name string) CreateServerlessEndpointOpts {
	return CreateServerlessEndpointOpts{
		Name:             name,
		HealthcheckUrl:   "https://example.com/" + name + "/health",
		TriggerUrl:       "https://example.com/" + name + "/trigger",
		SigningSecretEnc: "ciphertext-" + name,
	}
}

// resetServerlessLeases clears the lease and process tables so lease subtests start from a
// known set of units. Endpoint rows are left alone; the subtests that use them run first.
func resetServerlessLeases(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()

	_, err := pool.Exec(ctx, "TRUNCATE v1_serverless_lease, v1_serverless_process")
	require.NoError(t, err)
}

// seedServerlessUnits inserts n unowned units, one tenant each, and returns them.
func seedServerlessUnits(t *testing.T, ctx context.Context, repo ServerlessRepository, n int) []ServerlessUnit {
	t.Helper()

	units := make([]ServerlessUnit, 0, n)

	for i := 0; i < n; i++ {
		unit := ServerlessUnit{TenantId: uuid.New(), Shard: 0}
		require.NoError(t, repo.Leases().InsertIfAbsent(ctx, unit))
		require.NoError(t, repo.Leases().IncrementEndpointCount(ctx, unit, int32(i+1))) // nolint: gosec
		units = append(units, unit)
	}

	return units
}

func leaseRow(t *testing.T, ctx context.Context, pool *pgxpool.Pool, unit ServerlessUnit) (processId *uuid.UUID, endpointCount int32) {
	t.Helper()

	err := pool.QueryRow(ctx,
		"SELECT process_id, endpoint_count FROM v1_serverless_lease WHERE tenant_id = $1 AND shard = $2",
		unit.TenantId, unit.Shard,
	).Scan(&processId, &endpointCount)
	require.NoError(t, err)

	return processId, endpointCount
}

// heartbeat writes a live process row: the claim statement refuses claimers without one.
func heartbeat(t *testing.T, ctx context.Context, repo ServerlessRepository, processId uuid.UUID, ttl time.Duration) {
	t.Helper()

	require.NoError(t, repo.Processes().Upsert(ctx, UpsertServerlessProcessOpts{ProcessId: processId, TTL: ttl}))
}

// expireProcess backdates a process row so the database sees it as dead.
func expireProcess(t *testing.T, ctx context.Context, pool *pgxpool.Pool, processId uuid.UUID, by time.Duration) {
	t.Helper()

	_, err := pool.Exec(ctx, "UPDATE v1_serverless_process SET expires_at = now() - $2::interval WHERE process_id = $1", processId, by)
	require.NoError(t, err)
}

// claimAll drains every claimable unit for processId in batches of limit, from the beginning
// of the key space, and returns the units it received, in order.
func claimAll(t *testing.T, ctx context.Context, repo ServerlessRepository, processId uuid.UUID, limit int32) []ServerlessUnit {
	t.Helper()

	var claimed []ServerlessUnit

	for {
		rows, err := repo.Leases().Claim(ctx, processId, ServerlessUnit{}, limit)
		require.NoError(t, err)

		if len(rows) == 0 {
			return claimed
		}

		for _, row := range rows {
			claimed = append(claimed, ServerlessUnit{TenantId: row.TenantID, Shard: row.Shard})
		}
	}
}

// countClaimable counts with a window far larger than any test population.
func countClaimable(t *testing.T, ctx context.Context, repo ServerlessRepository) *sqlcv1.CountClaimableServerlessLeasesRow {
	t.Helper()

	row, err := repo.Leases().CountClaimable(ctx, 1_000_000)
	require.NoError(t, err)

	return row
}

func TestServerlessRepository(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()
	repo := newServerlessTestRepository(t, pool)

	t.Run("endpoint round trip", func(t *testing.T) {
		tenantId := uuid.New()

		created, err := repo.Endpoints().Create(ctx, tenantId, serverlessEndpointOpts("endpoint-a"))
		require.NoError(t, err)

		assert.Equal(t, tenantId, created.TenantID)
		assert.Equal(t, "endpoint-a", created.Name)
		assert.NotEqual(t, uuid.Nil, created.Namespace)
		assert.Equal(t, sqlcv1.V1ServerlessEndpointKindCLOUDFLAREWORKERS, created.Kind)
		assert.Equal(t, defaultServerlessRequestTimeoutSeconds, created.RequestTimeoutSeconds)
		assert.Equal(t, defaultServerlessPollIntervalSeconds, created.PollIntervalSeconds)
		assert.Equal(t, defaultServerlessInlineWaitBudgetMs, created.InlineWaitBudgetMs)
		assert.JSONEq(t, `{}`, string(created.Labels))
		assert.True(t, created.Enabled)
		// shard_count defaults to 1, so every endpoint lands on shard 0
		assert.Equal(t, int32(0), created.Shard)
		assert.False(t, created.Healthy.Valid)
		assert.Empty(t, created.RegisteredActions)

		got, err := repo.Endpoints().Get(ctx, tenantId, created.ID)
		require.NoError(t, err)
		assert.Equal(t, created.ID, got.ID)
		assert.Equal(t, created.Namespace, got.Namespace)

		// the endpoint is scoped to its tenant
		_, err = repo.Endpoints().Get(ctx, uuid.New(), created.ID)
		assert.ErrorIs(t, err, pgx.ErrNoRows)

		listed, count, err := repo.Endpoints().List(ctx, tenantId, ListServerlessEndpointsOpts{Limit: 10})
		require.NoError(t, err)
		assert.Equal(t, int64(1), count)
		require.Len(t, listed, 1)
		assert.Equal(t, created.ID, listed[0].ID)

		// same name in the same tenant is rejected by the unique constraint
		_, err = repo.Endpoints().Create(ctx, tenantId, serverlessEndpointOpts("endpoint-a"))
		var pgErr *pgconn.PgError
		require.ErrorAs(t, err, &pgErr)
		assert.Equal(t, pgUniqueViolation, pgErr.Code)

		// same name in another tenant is fine
		_, err = repo.Endpoints().Create(ctx, uuid.New(), serverlessEndpointOpts("endpoint-a"))
		require.NoError(t, err)

		newName := "endpoint-b"
		newTimeout := int32(5)
		disabled := false
		kind := sqlcv1.V1ServerlessEndpointKindGENERICHTTP

		updated, err := repo.Endpoints().Update(ctx, tenantId, created.ID, UpdateServerlessEndpointOpts{
			Name:                  &newName,
			Kind:                  &kind,
			RequestTimeoutSeconds: &newTimeout,
			Enabled:               &disabled,
			Labels:                []byte(`{"env": "test"}`),
		})
		require.NoError(t, err)
		assert.Equal(t, "endpoint-b", updated.Name)
		assert.Equal(t, kind, updated.Kind)
		assert.Equal(t, int32(5), updated.RequestTimeoutSeconds)
		assert.False(t, updated.Enabled)
		assert.JSONEq(t, `{"env": "test"}`, string(updated.Labels))
		// untouched fields keep their values, and the immutable ones cannot change
		assert.Equal(t, created.HealthcheckUrl, updated.HealthcheckUrl)
		assert.Equal(t, created.SigningSecretEnc, updated.SigningSecretEnc)
		assert.Equal(t, created.Namespace, updated.Namespace)
		assert.Equal(t, created.Shard, updated.Shard)
		assert.True(t, updated.UpdatedAt.Time.After(created.UpdatedAt.Time))

		// the incremental refresh sees the config change, keyed after the created row's version
		changed, err := repo.Endpoints().ListUpdatedSince(ctx, tenantId, created.UpdatedAt.Time, created.ID)
		require.NoError(t, err)
		require.Len(t, changed, 1)
		assert.Equal(t, created.ID, changed[0].ID)

		// the row at the watermark itself is not returned again
		changed, err = repo.Endpoints().ListUpdatedSince(ctx, tenantId, updated.UpdatedAt.Time, updated.ID)
		require.NoError(t, err)
		assert.Empty(t, changed, "the keyset excludes the last applied row")

		// a health flip is recorded without bumping updated_at, and returns its own timestamp
		statusError := "healthcheck returned 500"
		changedAt, err := repo.Endpoints().UpdateStatus(ctx, created.ID, false, &statusError)
		require.NoError(t, err)

		afterStatus, err := repo.Endpoints().Get(ctx, tenantId, created.ID)
		require.NoError(t, err)
		assert.True(t, afterStatus.Healthy.Valid)
		assert.False(t, afterStatus.Healthy.Bool)
		assert.Equal(t, statusError, afterStatus.StatusError.String)
		assert.True(t, afterStatus.StatusChangedAt.Valid)
		assert.Equal(t, afterStatus.StatusChangedAt.Time, changedAt, "the status write returns the row's status_changed_at")
		assert.Equal(t, updated.UpdatedAt.Time, afterStatus.UpdatedAt.Time, "health flip must not bump updated_at")

		// the health flip surfaces in the incremental refresh through the row version
		changed, err = repo.Endpoints().ListUpdatedSince(ctx, tenantId, updated.UpdatedAt.Time, updated.ID)
		require.NoError(t, err)
		require.Len(t, changed, 1, "a health flip surfaces in the incremental refresh")
		assert.False(t, changed[0].Healthy.Bool)

		// a workflow change is recorded and does bump updated_at
		actions := []string{created.Namespace.String() + "_svc:run"}
		require.NoError(t, repo.Endpoints().UpdateRegisteredActions(ctx, created.ID, actions))

		afterActions, err := repo.Endpoints().Get(ctx, tenantId, created.ID)
		require.NoError(t, err)
		assert.Equal(t, actions, afterActions.RegisteredActions)
		assert.True(t, afterActions.UpdatedAt.Time.After(updated.UpdatedAt.Time), "registered_actions change must bump updated_at")

		changed, err = repo.Endpoints().ListUpdatedSince(ctx, tenantId, changedAt, created.ID)
		require.NoError(t, err)
		require.Len(t, changed, 1)
		assert.Equal(t, actions, changed[0].RegisteredActions)

		deleted, err := repo.Endpoints().Delete(ctx, tenantId, created.ID)
		require.NoError(t, err)
		assert.Equal(t, created.ID, deleted.ID)

		_, err = repo.Endpoints().Get(ctx, tenantId, created.ID)
		assert.ErrorIs(t, err, pgx.ErrNoRows)

		_, err = repo.Endpoints().Delete(ctx, tenantId, created.ID)
		assert.ErrorIs(t, err, pgx.ErrNoRows)
	})

	t.Run("create validates options", func(t *testing.T) {
		tenantId := uuid.New()

		badUrl := serverlessEndpointOpts("bad-url")
		badUrl.TriggerUrl = "not a url"
		_, err := repo.Endpoints().Create(ctx, tenantId, badUrl)
		assert.Error(t, err)

		noSecret := serverlessEndpointOpts("no-secret")
		noSecret.SigningSecretEnc = ""
		_, err = repo.Endpoints().Create(ctx, tenantId, noSecret)
		assert.Error(t, err)

		tooLongTimeout := serverlessEndpointOpts("too-long-timeout")
		timeout := int32(1_000_000)
		tooLongTimeout.RequestTimeoutSeconds = &timeout
		_, err = repo.Endpoints().Create(ctx, tenantId, tooLongTimeout)
		assert.Error(t, err)

		badLabels := serverlessEndpointOpts("bad-labels")
		badLabels.Labels = []byte(`[1, 2]`)
		_, err = repo.Endpoints().Create(ctx, tenantId, badLabels)
		assert.Error(t, err)

		_, count, err := repo.Endpoints().List(ctx, tenantId, ListServerlessEndpointsOpts{Limit: 10})
		require.NoError(t, err)
		assert.Equal(t, int64(0), count, "no endpoint should have been created")
	})

	t.Run("namespace is unique", func(t *testing.T) {
		tenantId := uuid.New()

		first, err := repo.Endpoints().Create(ctx, tenantId, serverlessEndpointOpts("endpoint-a"))
		require.NoError(t, err)

		second, err := repo.Endpoints().Create(ctx, tenantId, serverlessEndpointOpts("endpoint-b"))
		require.NoError(t, err)

		assert.NotEqual(t, first.Namespace, second.Namespace)

		// the constraint holds even against a raw insert that reuses a namespace
		_, err = pool.Exec(ctx, `
			INSERT INTO v1_serverless_endpoint (tenant_id, name, namespace, healthcheck_url, trigger_url, signing_secret_enc)
			VALUES ($1, 'endpoint-c', $2, 'https://example.com/h', 'https://example.com/t', 'enc')`,
			tenantId, first.Namespace,
		)
		var pgErr *pgconn.PgError
		require.ErrorAs(t, err, &pgErr)
		assert.Equal(t, pgUniqueViolation, pgErr.Code)
		assert.Equal(t, "v1_serverless_endpoint_namespace_key", pgErr.ConstraintName)
	})

	t.Run("lease unit and endpoint_count follow endpoint create and delete", func(t *testing.T) {
		tenantId := uuid.New()
		unit := ServerlessUnit{TenantId: tenantId, Shard: 0}

		// the tenant row is created alongside the first endpoint
		_, err := repo.Tenants().Get(ctx, tenantId)
		assert.ErrorIs(t, err, pgx.ErrNoRows)

		first, err := repo.Endpoints().Create(ctx, tenantId, serverlessEndpointOpts("endpoint-a"))
		require.NoError(t, err)

		tenant, err := repo.Tenants().Get(ctx, tenantId)
		require.NoError(t, err)
		assert.Equal(t, int32(1), tenant.ShardCount)

		processId, endpointCount := leaseRow(t, ctx, pool, unit)
		assert.Nil(t, processId, "a new unit is unowned")
		assert.Equal(t, int32(1), endpointCount)

		second, err := repo.Endpoints().Create(ctx, tenantId, serverlessEndpointOpts("endpoint-b"))
		require.NoError(t, err)

		_, endpointCount = leaseRow(t, ctx, pool, unit)
		assert.Equal(t, int32(2), endpointCount)

		_, err = repo.Endpoints().Delete(ctx, tenantId, first.ID)
		require.NoError(t, err)

		_, endpointCount = leaseRow(t, ctx, pool, unit)
		assert.Equal(t, int32(1), endpointCount)

		_, err = repo.Endpoints().Delete(ctx, tenantId, second.ID)
		require.NoError(t, err)

		processId, endpointCount = leaseRow(t, ctx, pool, unit)
		assert.Nil(t, processId)
		assert.Equal(t, int32(0), endpointCount, "the lease row is kept with a zero count")
	})

	t.Run("tenant shard_count decides the shard", func(t *testing.T) {
		tenantId := uuid.New()
		const shardCount = int32(8)

		tenant, err := repo.Tenants().UpdateShardCount(ctx, tenantId, shardCount)
		require.NoError(t, err)
		assert.Equal(t, shardCount, tenant.ShardCount)

		_, err = repo.Tenants().UpdateShardCount(ctx, tenantId, 0)
		assert.Error(t, err)

		// every shard has a lease row up front
		var leaseRows int
		require.NoError(t, pool.QueryRow(ctx, "SELECT count(*) FROM v1_serverless_lease WHERE tenant_id = $1", tenantId).Scan(&leaseRows))
		assert.Equal(t, int(shardCount), leaseRows)

		const numEndpoints = 32
		created := make(map[uuid.UUID]*sqlcv1.V1ServerlessEndpoint, numEndpoints)
		shardsSeen := make(map[int32]int)

		for i := 0; i < numEndpoints; i++ {
			endpoint, err := repo.Endpoints().Create(ctx, tenantId, serverlessEndpointOpts(fmt.Sprintf("endpoint-%d", i)))
			require.NoError(t, err)

			require.GreaterOrEqual(t, endpoint.Shard, int32(0))
			require.Less(t, endpoint.Shard, shardCount)

			// the shard is the documented hash of the id
			var expectedShard int32
			require.NoError(t, pool.QueryRow(ctx, "SELECT (abs(hashtext($1::text)::bigint) % $2)::int", endpoint.ID, shardCount).Scan(&expectedShard))
			assert.Equal(t, expectedShard, endpoint.Shard)

			created[endpoint.ID] = endpoint
			shardsSeen[endpoint.Shard]++
		}

		assert.Greater(t, len(shardsSeen), 1, "32 endpoints over 8 shards should spread across more than one shard")

		// endpoint_count per unit adds up to the endpoints created
		for shard, want := range shardsSeen {
			_, endpointCount := leaseRow(t, ctx, pool, ServerlessUnit{TenantId: tenantId, Shard: shard})
			assert.Equal(t, int32(want), endpointCount, "shard %d", shard) // nolint: gosec
		}

		// ListForUnits pages every endpoint of the tenant's units exactly once
		units := make([]ServerlessUnit, 0, shardCount)
		for shard := int32(0); shard < shardCount; shard++ {
			units = append(units, ServerlessUnit{TenantId: tenantId, Shard: shard})
		}

		seen := make(map[uuid.UUID]bool)
		afterId := uuid.Nil

		for {
			page, err := repo.Endpoints().ListForUnits(ctx, units, afterId, 5)
			require.NoError(t, err)

			if len(page) == 0 {
				break
			}

			for _, endpoint := range page {
				require.False(t, seen[endpoint.ID], "endpoint %s returned twice", endpoint.ID)
				require.Contains(t, created, endpoint.ID)
				seen[endpoint.ID] = true
			}

			afterId = page[len(page)-1].ID
		}

		assert.Len(t, seen, numEndpoints)

		// a single unit returns only its own endpoints
		someShard := units[0]
		for shard := range shardsSeen {
			someShard = ServerlessUnit{TenantId: tenantId, Shard: shard}
			break
		}

		onlyShard, err := repo.Endpoints().ListForUnits(ctx, []ServerlessUnit{someShard}, uuid.Nil, 100)
		require.NoError(t, err)
		assert.Len(t, onlyShard, shardsSeen[someShard.Shard])
		for _, endpoint := range onlyShard {
			assert.Equal(t, someShard.Shard, endpoint.Shard)
		}

		forTenant, err := repo.Endpoints().ListForTenant(ctx, tenantId)
		require.NoError(t, err)
		assert.Len(t, forTenant, numEndpoints)
	})

	t.Run("process heartbeat, liveness and sweep", func(t *testing.T) {
		resetServerlessLeases(t, ctx, pool)

		shortLived := uuid.New()
		longLived := uuid.New()
		hostname := "host-a"

		require.NoError(t, repo.Processes().Upsert(ctx, UpsertServerlessProcessOpts{
			ProcessId: shortLived,
			TTL:       200 * time.Millisecond,
			UnitCount: 3,
			Hostname:  &hostname,
		}))
		require.NoError(t, repo.Processes().Upsert(ctx, UpsertServerlessProcessOpts{
			ProcessId:     longLived,
			TTL:           time.Hour,
			UnitCount:     1,
			EndpointCount: 7,
		}))

		require.Error(t, repo.Processes().Upsert(ctx, UpsertServerlessProcessOpts{ProcessId: uuid.New()}), "a TTL is required")

		live, dead, err := repo.Processes().ListLive(ctx)
		require.NoError(t, err)
		require.Len(t, live, 2)
		assert.Empty(t, dead)

		time.Sleep(300 * time.Millisecond)

		live, dead, err = repo.Processes().ListLive(ctx)
		require.NoError(t, err)
		require.Len(t, live, 1)
		assert.Equal(t, longLived, live[0].ProcessID)
		assert.Equal(t, int32(7), live[0].EndpointCount)
		require.Len(t, dead, 1)
		assert.Equal(t, shortLived, dead[0].ProcessID)

		// a heartbeat revives an expired row; the counts follow the latest heartbeat
		require.NoError(t, repo.Processes().Upsert(ctx, UpsertServerlessProcessOpts{
			ProcessId: shortLived,
			TTL:       time.Hour,
			UnitCount: 4,
		}))

		live, dead, err = repo.Processes().ListLive(ctx)
		require.NoError(t, err)
		assert.Len(t, live, 2)
		assert.Empty(t, dead)

		// expire it again and sweep it
		require.NoError(t, repo.Processes().Upsert(ctx, UpsertServerlessProcessOpts{ProcessId: shortLived, TTL: time.Millisecond}))
		time.Sleep(20 * time.Millisecond)

		swept, _, err := repo.Processes().DeleteExpired(ctx, time.Now().Add(-time.Hour))
		require.NoError(t, err)
		assert.Equal(t, int64(0), swept, "a row that expired after the cutoff is kept for takeover")

		swept, _, err = repo.Processes().DeleteExpired(ctx, time.Now())
		require.NoError(t, err)
		assert.Equal(t, int64(1), swept)

		require.NoError(t, repo.Processes().Delete(ctx, longLived))

		live, dead, err = repo.Processes().ListLive(ctx)
		require.NoError(t, err)
		assert.Empty(t, live)
		assert.Empty(t, dead)
	})

	t.Run("routing lookups by namespace, version listing and fetch by id", func(t *testing.T) {
		tenantId := uuid.New()

		a, err := repo.Endpoints().Create(ctx, tenantId, serverlessEndpointOpts("route-a"))
		require.NoError(t, err)
		b, err := repo.Endpoints().Create(ctx, tenantId, serverlessEndpointOpts("route-b"))
		require.NoError(t, err)

		got, err := repo.Endpoints().GetByNamespace(ctx, tenantId, a.Namespace)
		require.NoError(t, err)
		assert.Equal(t, a.ID, got.ID)

		_, err = repo.Endpoints().GetByNamespace(ctx, uuid.New(), a.Namespace)
		assert.ErrorIs(t, err, pgx.ErrNoRows, "the lookup is scoped to the tenant")

		// A status write moves b's version past a's; the listing pages in version order.
		_, err = repo.Endpoints().UpdateStatus(ctx, b.ID, false, nil)
		require.NoError(t, err)

		first, err := repo.Endpoints().ListVersions(ctx, tenantId, ServerlessEndpointVersion{}, 1)
		require.NoError(t, err)
		require.Len(t, first, 1)
		assert.Equal(t, a.ID, first[0].ID)
		assert.True(t, first[0].Version.Equal(a.UpdatedAt.Time))

		second, err := repo.Endpoints().ListVersions(ctx, tenantId, first[0], 1)
		require.NoError(t, err)
		require.Len(t, second, 1)
		assert.Equal(t, b.ID, second[0].ID)
		assert.True(t, second[0].Version.After(first[0].Version))

		third, err := repo.Endpoints().ListVersions(ctx, tenantId, second[0], 1)
		require.NoError(t, err)
		assert.Empty(t, third)

		rows, err := repo.Endpoints().ListByIds(ctx, []uuid.UUID{b.ID, a.ID})
		require.NoError(t, err)
		require.Len(t, rows, 2)
		assert.True(t, rows[0].ID.String() < rows[1].ID.String(), "rows come back by id")

		byId := map[uuid.UUID]*sqlcv1.V1ServerlessEndpoint{rows[0].ID: rows[0], rows[1].ID: rows[1]}
		require.Contains(t, byId, a.ID)
		require.Contains(t, byId, b.ID)
		assert.True(t, byId[b.ID].StatusChangedAt.Valid, "the fetched row carries the status write")
	})

	t.Run("empty units are neither counted nor claimed", func(t *testing.T) {
		resetServerlessLeases(t, ctx, pool)

		// Four empty units sort before the populated one; the count sample of four must
		// skip them, and a claim must never take them.
		for i := 0; i < 4; i++ {
			unit := ServerlessUnit{TenantId: uuid.MustParse(fmt.Sprintf("00000000-0000-0000-0000-00000000000%d", i+1)), Shard: 0}
			require.NoError(t, repo.Leases().InsertIfAbsent(ctx, unit))
		}

		populated := ServerlessUnit{TenantId: uuid.MustParse("00000000-0000-0000-0000-000000000009"), Shard: 0}
		require.NoError(t, repo.Leases().InsertIfAbsent(ctx, populated))
		require.NoError(t, repo.Leases().IncrementEndpointCount(ctx, populated, 3))

		sampled, err := repo.Leases().CountClaimable(ctx, 4)
		require.NoError(t, err)
		assert.Equal(t, int64(1), sampled.UnitCount)
		assert.Equal(t, int64(3), sampled.EndpointCount)

		processId := uuid.New()
		heartbeat(t, ctx, repo, processId, time.Minute)

		claimed := claimAll(t, ctx, repo, processId, 10)
		assert.Equal(t, []ServerlessUnit{populated}, claimed)

		// An empty unit held by a dead process is not counted or claimed either.
		dead := uuid.New()
		heartbeat(t, ctx, repo, dead, time.Minute)
		emptyDead := ServerlessUnit{TenantId: uuid.MustParse("00000000-0000-0000-0000-000000000005"), Shard: 0}
		_, err = pool.Exec(ctx, "UPDATE v1_serverless_lease SET process_id = $1 WHERE tenant_id = $2", dead, emptyDead.TenantId)
		require.NoError(t, err)
		expireProcess(t, ctx, pool, dead, time.Minute)

		sampled, err = repo.Leases().CountClaimable(ctx, 4)
		require.NoError(t, err)
		assert.Equal(t, int64(0), sampled.UnitCount)
		assert.Empty(t, claimAll(t, ctx, repo, processId, 10))
	})

	t.Run("abandoned count is bounded as a whole", func(t *testing.T) {
		resetServerlessLeases(t, ctx, pool)

		// Three dead owners of four units each: a window of five covers five units, not
		// five per owner.
		for i := 0; i < 3; i++ {
			dead := uuid.New()
			heartbeat(t, ctx, repo, dead, time.Minute)

			for j := 0; j < 4; j++ {
				unit := ServerlessUnit{TenantId: uuid.New(), Shard: 0}
				require.NoError(t, repo.Leases().InsertIfAbsent(ctx, unit))
				require.NoError(t, repo.Leases().IncrementEndpointCount(ctx, unit, 1))
				_, err := pool.Exec(ctx, "UPDATE v1_serverless_lease SET process_id = $1 WHERE tenant_id = $2", dead, unit.TenantId)
				require.NoError(t, err)
			}

			expireProcess(t, ctx, pool, dead, time.Minute)
		}

		windowed, err := repo.Leases().CountClaimable(ctx, 5)
		require.NoError(t, err)
		assert.Equal(t, int64(5), windowed.UnitCount)
		assert.Equal(t, int64(5), windowed.EndpointCount)

		exact := countClaimable(t, ctx, repo)
		assert.Equal(t, int64(12), exact.UnitCount)
	})

	t.Run("concurrent claims never double-claim", func(t *testing.T) {
		resetServerlessLeases(t, ctx, pool)

		const numUnits = 20
		units := seedServerlessUnits(t, ctx, repo, numUnits)

		unowned := countClaimable(t, ctx, repo)
		assert.Equal(t, int64(numUnits), unowned.UnitCount)
		// endpoint counts were seeded as 1..20
		assert.Equal(t, int64(numUnits*(numUnits+1)/2), unowned.EndpointCount)

		windowed, err := repo.Leases().CountClaimable(ctx, 5)
		require.NoError(t, err)
		assert.Equal(t, int64(5), windowed.UnitCount, "the count is capped at the window")

		processA := uuid.New()
		processB := uuid.New()
		heartbeat(t, ctx, repo, processA, time.Minute)
		heartbeat(t, ctx, repo, processB, time.Minute)

		var wg sync.WaitGroup
		var mu sync.Mutex
		claimedBy := make(map[ServerlessUnit]uuid.UUID)
		var total int

		for _, processId := range []uuid.UUID{processA, processB} {
			wg.Add(1)

			go func(processId uuid.UUID) {
				defer wg.Done()

				claimed := claimAll(t, ctx, repo, processId, 3)

				mu.Lock()
				defer mu.Unlock()

				for _, unit := range claimed {
					if owner, ok := claimedBy[unit]; ok {
						assert.Fail(t, "unit claimed twice", "unit %v claimed by %s and %s", unit, owner, processId)
					}

					claimedBy[unit] = processId
					total++
				}
			}(processId)
		}

		wg.Wait()

		assert.Equal(t, numUnits, total, "every unit claimed exactly once")
		assert.Len(t, claimedBy, numUnits)

		for _, unit := range units {
			owner, ok := claimedBy[unit]
			require.True(t, ok, "unit %v never claimed", unit)

			processId, _ := leaseRow(t, ctx, pool, unit)
			require.NotNil(t, processId)
			assert.Equal(t, owner, *processId)
		}

		ownedA, err := repo.Leases().ListOwned(ctx, processA)
		require.NoError(t, err)
		ownedB, err := repo.Leases().ListOwned(ctx, processB)
		require.NoError(t, err)
		assert.Equal(t, numUnits, len(ownedA)+len(ownedB))

		unowned = countClaimable(t, ctx, repo)
		assert.Equal(t, int64(0), unowned.UnitCount)
		assert.Equal(t, int64(0), unowned.EndpointCount)

		// nothing is left for a third process
		third := uuid.New()
		heartbeat(t, ctx, repo, third, time.Minute)
		none, err := repo.Leases().Claim(ctx, third, ServerlessUnit{}, 10)
		require.NoError(t, err)
		assert.Empty(t, none)

		// a non-positive limit is a no-op
		none, err = repo.Leases().Claim(ctx, third, ServerlessUnit{}, 0)
		require.NoError(t, err)
		assert.Empty(t, none)
	})

	t.Run("a claimer without a live row cannot claim", func(t *testing.T) {
		resetServerlessLeases(t, ctx, pool)
		seedServerlessUnits(t, ctx, repo, 3)

		processId := uuid.New()

		rows, err := repo.Leases().Claim(ctx, processId, ServerlessUnit{}, 10)
		require.NoError(t, err)
		assert.Empty(t, rows, "no heartbeat row yet")

		heartbeat(t, ctx, repo, processId, time.Minute)
		expireProcess(t, ctx, pool, processId, time.Second)

		rows, err = repo.Leases().Claim(ctx, processId, ServerlessUnit{}, 10)
		require.NoError(t, err)
		assert.Empty(t, rows, "an expired row is not live")

		heartbeat(t, ctx, repo, processId, time.Minute)

		rows, err = repo.Leases().Claim(ctx, processId, ServerlessUnit{}, 10)
		require.NoError(t, err)
		assert.Len(t, rows, 3)
	})

	t.Run("claim walks unowned units from the start key", func(t *testing.T) {
		resetServerlessLeases(t, ctx, pool)

		units := seedServerlessUnits(t, ctx, repo, 6)
		sort.Slice(units, func(i, j int) bool { return units[i].TenantId.String() < units[j].TenantId.String() })

		processId := uuid.New()
		heartbeat(t, ctx, repo, processId, time.Minute)

		// RETURNING does not preserve the walk's order, so results are compared as sets.
		claimedUnits := func(rows []*sqlcv1.ClaimServerlessLeasesRow) []ServerlessUnit {
			out := make([]ServerlessUnit, 0, len(rows))

			for _, row := range rows {
				out = append(out, ServerlessUnit{TenantId: row.TenantID, Shard: row.Shard})
			}

			return out
		}

		// Starting at the third key returns exactly the units after it.
		rows, err := repo.Leases().Claim(ctx, processId, units[2], 10)
		require.NoError(t, err)
		assert.ElementsMatch(t, units[3:], claimedUnits(rows))

		// The wrap from the zero key takes the rest, and the limit bounds one statement.
		rows, err = repo.Leases().Claim(ctx, processId, ServerlessUnit{}, 2)
		require.NoError(t, err)
		assert.ElementsMatch(t, units[:2], claimedUnits(rows))

		rows, err = repo.Leases().Claim(ctx, processId, ServerlessUnit{}, 10)
		require.NoError(t, err)
		assert.ElementsMatch(t, units[2:3], claimedUnits(rows))
	})

	// A process that expired but heartbeats again before anyone took its units over stays the
	// owner: the claim statement reads liveness in its own snapshot, so a dead list read
	// earlier cannot transfer a live owner's unit.
	t.Run("an owner that revives before the takeover keeps its units", func(t *testing.T) {
		resetServerlessLeases(t, ctx, pool)
		seedServerlessUnits(t, ctx, repo, 1)

		a, b := uuid.New(), uuid.New()
		heartbeat(t, ctx, repo, a, time.Second)
		heartbeat(t, ctx, repo, b, time.Minute)

		require.Len(t, claimAll(t, ctx, repo, a, 1), 1)

		expireProcess(t, ctx, pool, a, time.Second)

		_, dead, err := repo.Processes().ListLive(ctx)
		require.NoError(t, err)
		require.Len(t, dead, 1, "b sees a as dead")

		// a heartbeats before b claims
		heartbeat(t, ctx, repo, a, time.Minute)

		rows, err := repo.Leases().Claim(ctx, b, ServerlessUnit{}, 1)
		require.NoError(t, err)
		assert.Empty(t, rows, "a live owner's unit is not transferred on the strength of an earlier dead list")

		claimable := countClaimable(t, ctx, repo)
		assert.Equal(t, int64(0), claimable.UnitCount, "count and claim agree on liveness")

		// once a is really dead, b takes over
		expireProcess(t, ctx, pool, a, time.Second)

		claimable = countClaimable(t, ctx, repo)
		assert.Equal(t, int64(1), claimable.UnitCount)

		rows, err = repo.Leases().Claim(ctx, b, ServerlessUnit{}, 1)
		require.NoError(t, err)
		assert.Len(t, rows, 1)
	})

	// Sweeping a dead process row must not strand the units it still owns: the row deletion
	// releases them in the same statement, and count and claim keep agreeing.
	t.Run("a swept owner row releases its units", func(t *testing.T) {
		resetServerlessLeases(t, ctx, pool)
		seedServerlessUnits(t, ctx, repo, 3)

		dead, live := uuid.New(), uuid.New()
		heartbeat(t, ctx, repo, dead, time.Second)
		heartbeat(t, ctx, repo, live, time.Minute)

		require.Len(t, claimAll(t, ctx, repo, dead, 10), 3)

		expireProcess(t, ctx, pool, dead, 2*time.Hour)

		swept, released, err := repo.Processes().DeleteExpired(ctx, time.Now().Add(-time.Hour))
		require.NoError(t, err)
		assert.Equal(t, int64(1), swept)
		assert.Equal(t, int64(3), released, "the sweep released what the swept row still owned")

		claimable := countClaimable(t, ctx, repo)
		assert.Equal(t, int64(3), claimable.UnitCount)

		assert.Len(t, claimAll(t, ctx, repo, live, 10), 3, "the survivor takes the swept process's units")
	})

	t.Run("a graceful delete releases units a failed release left behind", func(t *testing.T) {
		resetServerlessLeases(t, ctx, pool)
		units := seedServerlessUnits(t, ctx, repo, 2)

		processId := uuid.New()
		heartbeat(t, ctx, repo, processId, time.Minute)
		require.Len(t, claimAll(t, ctx, repo, processId, 10), 2)

		require.NoError(t, repo.Processes().Delete(ctx, processId))

		for _, unit := range units {
			owner, _ := leaseRow(t, ctx, pool, unit)
			assert.Nil(t, owner)
		}

		live, dead, err := repo.Processes().ListLive(ctx)
		require.NoError(t, err)
		assert.Empty(t, live)
		assert.Empty(t, dead)
	})

	t.Run("dead process takeover", func(t *testing.T) {
		resetServerlessLeases(t, ctx, pool)

		const numUnits = 10
		seedServerlessUnits(t, ctx, repo, numUnits)

		deadProcess := uuid.New()
		liveProcess := uuid.New()
		successor := uuid.New()

		for _, processId := range []uuid.UUID{deadProcess, liveProcess, successor} {
			heartbeat(t, ctx, repo, processId, time.Minute)
		}

		deadClaimed, err := repo.Leases().Claim(ctx, deadProcess, ServerlessUnit{}, 6)
		require.NoError(t, err)
		assert.Len(t, deadClaimed, 6, "a claim takes exactly the units it asks for")

		liveClaimed, err := repo.Leases().Claim(ctx, liveProcess, ServerlessUnit{}, 4)
		require.NoError(t, err)
		assert.Len(t, liveClaimed, 4)

		// a live owner's units are not claimable
		none, err := repo.Leases().Claim(ctx, successor, ServerlessUnit{}, numUnits)
		require.NoError(t, err)
		assert.Empty(t, none)

		// once the dead process's row expires, exactly its units are handed over
		expireProcess(t, ctx, pool, deadProcess, time.Second)

		taken := claimAll(t, ctx, repo, successor, 4)
		assert.Len(t, taken, 6)

		ownedDead, err := repo.Leases().ListOwned(ctx, deadProcess)
		require.NoError(t, err)
		assert.Empty(t, ownedDead)

		ownedLive, err := repo.Leases().ListOwned(ctx, liveProcess)
		require.NoError(t, err)
		assert.Len(t, ownedLive, 4)

		ownedSuccessor, err := repo.Leases().ListOwned(ctx, successor)
		require.NoError(t, err)
		assert.Len(t, ownedSuccessor, 6)

		for _, lease := range ownedSuccessor {
			assert.True(t, lease.ClaimedAt.Valid)
		}
	})

	t.Run("shed and release all", func(t *testing.T) {
		resetServerlessLeases(t, ctx, pool)

		const numUnits = 20
		units := seedServerlessUnits(t, ctx, repo, numUnits)

		owner := uuid.New()
		other := uuid.New()
		heartbeat(t, ctx, repo, owner, time.Minute)
		heartbeat(t, ctx, repo, other, time.Minute)

		assert.Len(t, claimAll(t, ctx, repo, owner, 7), numUnits)

		// only the owner can shed
		shed, err := repo.Leases().Shed(ctx, other, units[:5])
		require.NoError(t, err)
		assert.Empty(t, shed)

		shed, err = repo.Leases().Shed(ctx, owner, units[:5])
		require.NoError(t, err)
		assert.Len(t, shed, 5)

		for _, row := range shed {
			assert.Greater(t, row.EndpointCount, int32(0), "shed rows carry their weight back")
		}

		owned, err := repo.Leases().ListOwned(ctx, owner)
		require.NoError(t, err)
		assert.Len(t, owned, numUnits-5)

		unowned := countClaimable(t, ctx, repo)
		assert.Equal(t, int64(5), unowned.UnitCount)

		for _, unit := range units[:5] {
			processId, _ := leaseRow(t, ctx, pool, unit)
			assert.Nil(t, processId)
		}

		// shedding an already shed unit is a no-op
		shed, err = repo.Leases().Shed(ctx, owner, units[:5])
		require.NoError(t, err)
		assert.Empty(t, shed)

		// empty input is a no-op
		shed, err = repo.Leases().Shed(ctx, owner, nil)
		require.NoError(t, err)
		assert.Empty(t, shed)

		// a shed unit is claimable again, by anyone
		reclaimed := claimAll(t, ctx, repo, other, 10)
		assert.Len(t, reclaimed, 5)

		released, err := repo.Leases().ReleaseAll(ctx, owner)
		require.NoError(t, err)
		assert.Equal(t, int64(numUnits-5), released)

		owned, err = repo.Leases().ListOwned(ctx, owner)
		require.NoError(t, err)
		assert.Empty(t, owned)

		ownedOther, err := repo.Leases().ListOwned(ctx, other)
		require.NoError(t, err)
		assert.Len(t, ownedOther, 5, "release all touches only the caller's units")

		unowned = countClaimable(t, ctx, repo)
		assert.Equal(t, int64(numUnits-5), unowned.UnitCount)

		released, err = repo.Leases().ReleaseAll(ctx, owner)
		require.NoError(t, err)
		assert.Equal(t, int64(0), released)
	})

	// Holding units must not cost per-unit writes: the process heartbeat is the only
	// steady-state write. The xmin system column changes on any UPDATE of a row, even one
	// that writes identical values, so it proves the lease rows were never touched.
	t.Run("steady-state heartbeats do not rewrite lease rows", func(t *testing.T) {
		resetServerlessLeases(t, ctx, pool)

		const numUnits = 10
		seedServerlessUnits(t, ctx, repo, numUnits)

		processId := uuid.New()
		heartbeat(t, ctx, repo, processId, time.Minute)
		claimed := claimAll(t, ctx, repo, processId, numUnits)
		require.Len(t, claimed, numUnits)

		heartbeat := func() {
			require.NoError(t, repo.Processes().Upsert(ctx, UpsertServerlessProcessOpts{
				ProcessId:     processId,
				TTL:           15 * time.Second,
				UnitCount:     numUnits,
				EndpointCount: 55,
			}))
		}

		heartbeat()

		snapshotLeaseRowVersions := func() map[ServerlessUnit]string {
			rows, err := pool.Query(ctx, "SELECT tenant_id, shard, xmin::text FROM v1_serverless_lease")
			require.NoError(t, err)
			defer rows.Close()

			versions := make(map[ServerlessUnit]string, numUnits)

			for rows.Next() {
				var unit ServerlessUnit
				var xmin string
				require.NoError(t, rows.Scan(&unit.TenantId, &unit.Shard, &xmin))
				versions[unit] = xmin
			}

			require.NoError(t, rows.Err())
			require.Len(t, versions, numUnits)

			return versions
		}

		processExpiry := func() time.Time {
			var expiresAt time.Time
			require.NoError(t, pool.QueryRow(ctx, "SELECT expires_at FROM v1_serverless_process WHERE process_id = $1", processId).Scan(&expiresAt))
			return expiresAt
		}

		before := snapshotLeaseRowVersions()
		expiryBefore := processExpiry()

		for i := 0; i < 5; i++ {
			time.Sleep(10 * time.Millisecond)
			heartbeat()
		}

		assert.Equal(t, before, snapshotLeaseRowVersions(), "lease rows were written during steady-state hold")
		assert.True(t, processExpiry().After(expiryBefore), "the process heartbeat kept advancing expires_at")

		// ownership is intact
		owned, err := repo.Leases().ListOwned(ctx, processId)
		require.NoError(t, err)
		assert.Len(t, owned, numUnits)
	})

	t.Run("tenant upsert is idempotent", func(t *testing.T) {
		tenantId := uuid.New()

		first, err := repo.Tenants().Upsert(ctx, tenantId)
		require.NoError(t, err)
		assert.Equal(t, int32(1), first.ShardCount)

		_, err = repo.Tenants().UpdateShardCount(ctx, tenantId, 3)
		require.NoError(t, err)

		again, err := repo.Tenants().Upsert(ctx, tenantId)
		require.NoError(t, err)
		assert.Equal(t, int32(3), again.ShardCount, "upsert returns the existing row untouched")
	})
}
