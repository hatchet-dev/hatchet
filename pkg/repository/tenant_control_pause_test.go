//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/integrations/metrics/prometheus"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func TestTenantControlPause(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()
	repo, limits := newTenantControlTestRepo(t, pool)
	paused := createLimitTestTenant(t, pool)
	other := createLimitTestTenant(t, pool)

	row, err := repo.Set(ctx, SetTenantControlOpts{TenantId: paused, Paused: true, Reason: "incident"})
	require.NoError(t, err)
	require.True(t, row.Paused)
	require.False(t, row.IngestPerSecond.Valid)
	require.True(t, row.PausedAt.Valid, "the trigger sets paused_at")
	pausedAt := row.PausedAt.Time

	require.False(t, repo.IsPaused(paused), "a process that has not refreshed pauses no tenant")
	require.NoError(t, repo.refresh(ctx))
	require.True(t, repo.IsPaused(paused))
	require.False(t, repo.IsPaused(other))

	require.Equal(t, map[string]float64{"paused": 1}, tenantControlSeries(t, prometheus.TenantControl, paused))
	require.Equal(t, map[string]float64{"paused": 0}, tenantControlSeries(t, prometheus.TenantControlRefused, paused))

	for _, resource := range []sqlcv1.LimitResource{sqlcv1.LimitResourceTASKRUN, sqlcv1.LimitResourceEVENT} {
		ok, _, err := limits.CanCreate(ctx, resource, paused, 2)
		require.False(t, ok)
		require.ErrorIs(t, err, ErrTenantPaused)
		require.ErrorIs(t, err, ErrResourceExhausted)
		require.True(t, IsTenantControlRefusal(err))

		ok, _, err = limits.CanCreate(ctx, resource, other, 2)
		require.NoError(t, err)
		require.True(t, ok)
	}

	require.Equal(t, map[string]float64{"paused": 4}, tenantControlSeries(t, prometheus.TenantControlRefused, paused), "the counter counts refused items")

	ok, _, err := limits.CanCreate(ctx, sqlcv1.LimitResourceINCOMINGWEBHOOK, paused, 1)
	require.NoError(t, err)
	require.True(t, ok, "the pause refuses only new runs and events")

	// A pause with an ingest limit refuses as a pause and keeps the start of the pause.
	time.Sleep(10 * time.Millisecond)

	row, err = repo.Set(ctx, SetTenantControlOpts{TenantId: paused, Paused: true, IngestPerSecond: perSecond(1000), Reason: "incident, resume slowly"})
	require.NoError(t, err)
	require.True(t, pausedAt.Equal(row.PausedAt.Time), "paused_at stays while the tenant stays paused")
	require.NoError(t, repo.refresh(ctx))
	require.ErrorIs(t, repo.Admit(paused, 1), ErrTenantPaused)
	require.Equal(t, map[string]float64{"paused": 1, "ingest": 1}, tenantControlSeries(t, prometheus.TenantControl, paused))
	require.Equal(t, map[string]float64{"paused": 5, "ingest": 0}, tenantControlSeries(t, prometheus.TenantControlRefused, paused))

	// Ending the pause keeps the ingest limit, and removes the pause's series.
	row, err = repo.Set(ctx, SetTenantControlOpts{TenantId: paused, IngestPerSecond: perSecond(1000), Reason: "resuming"})
	require.NoError(t, err)
	require.False(t, row.Paused)
	require.False(t, row.PausedAt.Valid)
	require.NoError(t, repo.refresh(ctx))
	require.False(t, repo.IsPaused(paused))
	require.NoError(t, repo.Admit(paused, 1))
	require.Equal(t, map[string]float64{"ingest": 1}, tenantControlSeries(t, prometheus.TenantControl, paused))
	require.Equal(t, map[string]float64{"ingest": 0}, tenantControlSeries(t, prometheus.TenantControlRefused, paused))

	// A new pause starts a new paused_at.
	row, err = repo.Set(ctx, SetTenantControlOpts{TenantId: paused, Paused: true, Reason: "again"})
	require.NoError(t, err)
	require.True(t, row.PausedAt.Time.After(pausedAt))

	_, err = repo.Set(ctx, SetTenantControlOpts{TenantId: paused, Reason: "nothing"})
	require.Error(t, err, "a row must set at least one control")

	_, err = repo.Clear(ctx, paused)
	require.NoError(t, err)
	require.NoError(t, repo.refresh(ctx))
	require.NoError(t, repo.Admit(paused, 1))
	require.Empty(t, tenantControlSeries(t, prometheus.TenantControl, paused))
	require.Empty(t, tenantControlSeries(t, prometheus.TenantControlRefused, paused))
}

// queuedDeadlines inserts one queued item of the tenant into each table whose rows carry a
// schedule deadline, plus a filled concurrency slot whose task has no runtime (as for a task that
// holds its slot while it waits in v1_rate_limited_queue_items), all due a minute from now, and
// returns a function that reads them back keyed by table.
func queuedDeadlines(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID) func() map[string]time.Time {
	t.Helper()

	ctx := context.Background()

	_, err := pool.Exec(ctx, `
WITH ids AS (
    SELECT $1::uuid AS tenant_id, NOW() AS inserted_at, (NOW() + INTERVAL '1 minute')::timestamp(3) AS due,
        (random() * 1e9)::bigint AS task_id
)
, qi AS (
    INSERT INTO v1_queue_item (tenant_id, queue, task_id, task_inserted_at, external_id, action_id, step_id, workflow_id, workflow_run_id, schedule_timeout_at, sticky)
    SELECT tenant_id, 'q', task_id, inserted_at, gen_random_uuid(), 'a', gen_random_uuid(), gen_random_uuid(), gen_random_uuid(), due, 'NONE' FROM ids
)
, rl AS (
    INSERT INTO v1_rate_limited_queue_items (requeue_after, tenant_id, queue, task_id, task_inserted_at, external_id, action_id, step_id, workflow_id, workflow_run_id, schedule_timeout_at, sticky)
    SELECT inserted_at, tenant_id, 'q', task_id + 1, inserted_at, gen_random_uuid(), 'a', gen_random_uuid(), gen_random_uuid(), gen_random_uuid(), due, 'NONE' FROM ids
)
, bq AS (
    INSERT INTO v1_batched_queue_item (tenant_id, queue, task_id, task_inserted_at, external_id, action_id, step_id, workflow_id, workflow_run_id, schedule_timeout_at, sticky, batch_key)
    SELECT tenant_id, 'q', task_id + 2, inserted_at, gen_random_uuid(), 'a', gen_random_uuid(), gen_random_uuid(), gen_random_uuid(), due, 'NONE', 'k' FROM ids
)
INSERT INTO v1_concurrency_slot (task_id, task_inserted_at, task_retry_count, external_id, tenant_id, workflow_id, workflow_version_id, workflow_run_id, strategy_id, key, is_filled, queue_to_notify, schedule_timeout_at)
SELECT task_id + n, inserted_at, 0, gen_random_uuid(), tenant_id, gen_random_uuid(), gen_random_uuid(), gen_random_uuid(), 1, 'k', n = 4, 'q', due
FROM ids, (VALUES (3), (4)) AS slots (n)
`, tenantID)
	require.NoError(t, err)

	return func() map[string]time.Time {
		rows, err := pool.Query(ctx, `
SELECT 'queue_item', schedule_timeout_at FROM v1_queue_item WHERE tenant_id = $1
UNION ALL SELECT 'rate_limited', schedule_timeout_at FROM v1_rate_limited_queue_items WHERE tenant_id = $1
UNION ALL SELECT 'batched', schedule_timeout_at FROM v1_batched_queue_item WHERE tenant_id = $1
UNION ALL SELECT CASE WHEN is_filled THEN 'filled_slot' ELSE 'queued_slot' END, schedule_timeout_at FROM v1_concurrency_slot WHERE tenant_id = $1
`, tenantID)
		require.NoError(t, err)
		defer rows.Close()

		deadlines := map[string]time.Time{}

		for rows.Next() {
			var table string
			var due time.Time
			require.NoError(t, rows.Scan(&table, &due))
			deadlines[table] = due
		}

		require.NoError(t, rows.Err())
		require.Len(t, deadlines, 5)

		return deadlines
	}
}

// requireShifted checks that every deadline moved forward by the time the tenant was paused, which
// the caller knows to lie between atLeast and atMost.
func requireShifted(t *testing.T, before, after map[string]time.Time, atLeast, atMost time.Duration) {
	t.Helper()

	for table, due := range before {
		moved := after[table].Sub(due)

		require.GreaterOrEqual(t, moved, atLeast, table)
		require.LessOrEqual(t, moved, atMost, table)
	}
}

func TestTenantControlResumeMovesScheduleDeadlines(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()
	repo, _ := newTenantControlTestRepo(t, pool)

	pause := func(t *testing.T, tenantID uuid.UUID) {
		_, err := repo.Set(ctx, SetTenantControlOpts{TenantId: tenantID, Paused: true, Reason: "incident"})
		require.NoError(t, err)
	}

	resumes := map[string]func(t *testing.T, tenantID uuid.UUID){
		"set without a pause": func(t *testing.T, tenantID uuid.UUID) {
			_, err := repo.Set(ctx, SetTenantControlOpts{TenantId: tenantID, IngestPerSecond: perSecond(100), Reason: "resume slowly"})
			require.NoError(t, err)
		},
		"clear": func(t *testing.T, tenantID uuid.UUID) {
			cleared, err := repo.Clear(ctx, tenantID)
			require.NoError(t, err)
			require.True(t, cleared)
		},
		"plain sql update": func(t *testing.T, tenantID uuid.UUID) {
			_, err := pool.Exec(ctx, `UPDATE v1_tenant_control SET paused = FALSE, ingest_per_second = 100 WHERE tenant_id = $1`, tenantID)
			require.NoError(t, err)
		},
		"plain sql delete": func(t *testing.T, tenantID uuid.UUID) {
			_, err := pool.Exec(ctx, `DELETE FROM v1_tenant_control WHERE tenant_id = $1`, tenantID)
			require.NoError(t, err)
		},
	}

	for name, resume := range resumes {
		t.Run(name, func(t *testing.T) {
			tenantID := createLimitTestTenant(t, pool)
			other := createLimitTestTenant(t, pool)

			deadlines := queuedDeadlines(t, pool, tenantID)
			otherDeadlines := queuedDeadlines(t, pool, other)
			before, otherBefore := deadlines(), otherDeadlines()

			const pausedFor = 500 * time.Millisecond

			start := time.Now()
			pause(t, tenantID)
			time.Sleep(pausedFor)

			// staying paused moves nothing
			pause(t, tenantID)
			require.Equal(t, before, deadlines())

			resume(t, tenantID)
			requireShifted(t, before, deadlines(), pausedFor, time.Since(start))
			require.Equal(t, otherBefore, otherDeadlines(), "another tenant's deadlines do not move")

			// a second resume has no pause to make up for
			after := deadlines()
			_, err := pool.Exec(ctx, `DELETE FROM v1_tenant_control WHERE tenant_id = $1`, tenantID)
			require.NoError(t, err)
			require.Equal(t, after, deadlines())
		})
	}

	t.Run("deleting the tenant", func(t *testing.T) {
		tenantID := createLimitTestTenant(t, pool)
		deadlines := queuedDeadlines(t, pool, tenantID)
		before := deadlines()

		pause(t, tenantID)
		time.Sleep(100 * time.Millisecond)

		_, err := pool.Exec(ctx, `DELETE FROM "Tenant" WHERE id = $1`, tenantID)
		require.NoError(t, err)

		rows, err := repo.List(ctx)
		require.NoError(t, err)
		require.Empty(t, rows, "the control row goes with the tenant")
		require.Equal(t, before, deadlines(), "the cascade does not touch the rows of a deleted tenant")
	})
}

// The scheduler's tenant list reads the pause from v1_tenant_control itself, so it leaves out a
// paused tenant whatever the process's snapshot holds.
func TestListSchedulableTenantsLeavesOutPausedTenants(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()
	controls, limits := newTenantControlTestRepo(t, pool)
	tenants := newTenantRepository(limits.sharedRepository, time.Minute)

	partitionID, err := tenants.CreateSchedulerPartition(ctx)
	require.NoError(t, err)

	paused, limited, plain := createLimitTestTenant(t, pool), createLimitTestTenant(t, pool), createLimitTestTenant(t, pool)

	_, err = pool.Exec(ctx, `UPDATE "Tenant" SET "schedulerPartitionId" = $1, "version" = 'V1' WHERE id = ANY($2::uuid[])`,
		partitionID, []uuid.UUID{paused, limited, plain})
	require.NoError(t, err)

	ids := func(list []*sqlcv1.Tenant, err error) []uuid.UUID {
		t.Helper()
		require.NoError(t, err)

		out := make([]uuid.UUID, 0, len(list))
		for _, tenant := range list {
			out = append(out, tenant.ID)
		}

		return out
	}

	_, err = controls.Set(ctx, SetTenantControlOpts{TenantId: paused, Paused: true, Reason: "incident"})
	require.NoError(t, err)
	_, err = controls.Set(ctx, SetTenantControlOpts{TenantId: limited, IngestPerSecond: perSecond(10), Reason: "noisy"})
	require.NoError(t, err)

	require.ElementsMatch(t, []uuid.UUID{limited, plain}, ids(tenants.ListSchedulableTenantsBySchedulerPartition(ctx, partitionID)),
		"a paused tenant is left out; one with only an ingest limit is not")
	require.ElementsMatch(t, []uuid.UUID{paused, limited, plain}, ids(tenants.ListTenantsBySchedulerPartition(ctx, partitionID)),
		"the unfiltered list keeps the paused tenant")

	_, err = controls.Set(ctx, SetTenantControlOpts{TenantId: paused, IngestPerSecond: perSecond(10), Reason: "resuming"})
	require.NoError(t, err)

	require.ElementsMatch(t, []uuid.UUID{paused, limited, plain}, ids(tenants.ListSchedulableTenantsBySchedulerPartition(ctx, partitionID)),
		"an unpaused tenant is listed again")
}
