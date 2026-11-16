//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	promclient "github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/integrations/metrics/prometheus"
	"github.com/hatchet-dev/hatchet/pkg/repository/fairpool"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// newTenantControlTestRepo builds a control repository on the shared repository of a tenant limit
// repository, so both read the same snapshot. Overrides are pushed into the returned gated pool,
// which has 10 connections and no default cap.
func newTenantControlTestRepo(t *testing.T, pool *pgxpool.Pool) (*tenantControlRepository, *tenantLimitRepository, *fairpool.Pool) {
	t.Helper()

	cfg := pool.Config()
	cfg.MaxConns = 10

	gated, err := fairpool.NewWithConfig(context.Background(), cfg, fairpool.Options{MaxWait: 100 * time.Millisecond})
	require.NoError(t, err)
	t.Cleanup(gated.Close)

	limits := createTenantLimitRepositoryForTest(t, pool, defaultLimitTestConfig())

	return newTenantControlRepository(limits.sharedRepository, gated), limits, gated
}

// tenantControlSeries returns the hatchet_tenant_control series of one tenant, keyed by control.
func tenantControlSeries(t *testing.T, tenantID uuid.UUID) map[string]float64 {
	t.Helper()

	ch := make(chan promclient.Metric)
	go func() {
		prometheus.TenantControl.Collect(ch)
		close(ch)
	}()

	series := map[string]float64{}

	for metric := range ch {
		var m dto.Metric
		require.NoError(t, metric.Write(&m))

		labels := map[string]string{}
		for _, label := range m.GetLabel() {
			labels[label.GetName()] = label.GetValue()
		}

		if labels["tenant_id"] == tenantID.String() {
			series[labels["control"]] = m.GetGauge().GetValue()
		}
	}

	return series
}

func TestTenantControlSetClearList(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()
	repo, _, _ := newTenantControlTestRepo(t, pool)
	tenantID := createLimitTestTenant(t, pool)

	row, err := repo.Set(ctx, SetTenantControlOpts{TenantId: tenantID, Paused: true, Reason: "noisy neighbour"})
	require.NoError(t, err)
	require.True(t, row.Paused)
	require.False(t, row.DbConnPercent.Valid)
	require.Equal(t, "noisy neighbour", row.Reason)

	// set replaces the whole row: a control left out is unset
	row, err = repo.Set(ctx, SetTenantControlOpts{TenantId: tenantID, DBConnPercent: int32Ptr(10), DBTimePercent: int32Ptr(20), IngestPerSecond: int32Ptr(500), Reason: "draining"})
	require.NoError(t, err)
	require.False(t, row.Paused)
	require.Equal(t, int32(10), row.DbConnPercent.Int32)
	require.Equal(t, int32(20), row.DbTimePercent.Int32)
	require.Equal(t, int32(500), row.IngestPerSecond.Int32)

	rows, err := repo.List(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, tenantID, rows[0].TenantID)
	require.Equal(t, "draining", rows[0].Reason)

	cleared, err := repo.Clear(ctx, tenantID)
	require.NoError(t, err)
	require.True(t, cleared)

	cleared, err = repo.Clear(ctx, tenantID)
	require.NoError(t, err)
	require.False(t, cleared)

	rows, err = repo.List(ctx)
	require.NoError(t, err)
	require.Empty(t, rows)

	_, err = repo.Set(ctx, SetTenantControlOpts{TenantId: uuid.New(), Paused: true, Reason: "unknown tenant"})
	require.Error(t, err, "a control for a tenant that does not exist must be refused")

	_, err = repo.Set(ctx, SetTenantControlOpts{TenantId: tenantID, DBConnPercent: int32Ptr(0), Reason: "out of range"})
	require.Error(t, err, "db_conn_percent outside 1 to 100 must be refused")

	_, err = repo.Set(ctx, SetTenantControlOpts{TenantId: tenantID, DBTimePercent: int32Ptr(101), Reason: "out of range"})
	require.Error(t, err, "db_time_percent outside 1 to 100 must be refused")

	_, err = repo.Set(ctx, SetTenantControlOpts{TenantId: tenantID, IngestPerSecond: int32Ptr(0), Reason: "out of range"})
	require.Error(t, err, "ingest_per_second below 1 must be refused")
}

func TestTenantControlRefresh(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()
	repo, _, gated := newTenantControlTestRepo(t, pool)
	tenantID := createLimitTestTenant(t, pool)

	_, err := repo.Set(ctx, SetTenantControlOpts{TenantId: tenantID, Paused: true, DBConnPercent: int32Ptr(10), DBTimePercent: int32Ptr(5), Reason: "test"})
	require.NoError(t, err)
	require.False(t, repo.IsPaused(tenantID), "a process that has not refreshed treats every tenant as uncontrolled")

	require.NoError(t, repo.refresh(ctx))
	require.True(t, repo.IsPaused(tenantID))
	require.Equal(t, map[string]float64{"paused": 1, "slow": 1, "budget": 1}, tenantControlSeries(t, tenantID))

	// 10 percent of 10 connections: the second checkout waits
	tx, err := gated.ForTenant(tenantID).Begin(ctx)
	require.NoError(t, err)

	_, err = gated.ForTenant(tenantID).Begin(ctx)
	var limitErr *fairpool.LimitError
	require.ErrorAs(t, err, &limitErr)
	require.Equal(t, int64(1), limitErr.Limit)

	// a failed refresh keeps the last snapshot
	_, err = repo.Clear(ctx, tenantID)
	require.NoError(t, err)

	canceled, cancel := context.WithCancel(ctx)
	cancel()
	require.Error(t, repo.refresh(canceled))
	require.True(t, repo.IsPaused(tenantID))

	require.NoError(t, repo.refresh(ctx))
	require.False(t, repo.IsPaused(tenantID))
	require.Empty(t, tenantControlSeries(t, tenantID))

	other, err := gated.ForTenant(tenantID).Begin(ctx)
	require.NoError(t, err, "a cleared tenant is no longer capped")
	require.NoError(t, other.Rollback(ctx))
	require.NoError(t, tx.Rollback(ctx))
}

func TestCanCreateRefusesPausedTenant(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()
	repo, limits, _ := newTenantControlTestRepo(t, pool)
	limits.enforceLimits = false

	paused := createLimitTestTenant(t, pool)
	other := createLimitTestTenant(t, pool)

	_, err := repo.Set(ctx, SetTenantControlOpts{TenantId: paused, Paused: true, Reason: "test"})
	require.NoError(t, err)
	require.NoError(t, repo.refresh(ctx))

	for _, resource := range []sqlcv1.LimitResource{sqlcv1.LimitResourceTASKRUN, sqlcv1.LimitResourceEVENT} {
		ok, _, err := limits.CanCreate(ctx, resource, paused, 1)
		require.False(t, ok)
		require.ErrorIs(t, err, ErrTenantPaused)
		require.True(t, errors.Is(err, ErrResourceExhausted), "a pause must map like a quota for callers that check ErrResourceExhausted")

		ok, _, err = limits.CanCreate(ctx, resource, other, 1)
		require.NoError(t, err)
		require.True(t, ok)
	}

	ok, _, err := limits.CanCreate(ctx, sqlcv1.LimitResourceINCOMINGWEBHOOK, paused, 1)
	require.NoError(t, err)
	require.True(t, ok, "a pause only refuses new runs and events")

	_, err = repo.Clear(ctx, paused)
	require.NoError(t, err)
	require.NoError(t, repo.refresh(ctx))

	ok, _, err = limits.CanCreate(ctx, sqlcv1.LimitResourceTASKRUN, paused, 1)
	require.NoError(t, err)
	require.True(t, ok)
}

func TestCanCreateMetersIngestRate(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()
	repo, limits, _ := newTenantControlTestRepo(t, pool)
	limits.enforceLimits = false

	limited := createLimitTestTenant(t, pool)
	other := createLimitTestTenant(t, pool)

	_, err := repo.Set(ctx, SetTenantControlOpts{TenantId: limited, IngestPerSecond: int32Ptr(50), Reason: "test"})
	require.NoError(t, err)
	require.NoError(t, repo.refresh(ctx))
	require.Equal(t, map[string]float64{"ingest": 1}, tenantControlSeries(t, limited))

	// Both tenants ask as fast as they can for three seconds. The limited tenant gets the bucket's
	// one second of tokens up front and 50 a second after that; the other tenant gets everything.
	var refusal *TenantRateLimitedError
	accepted, refused, otherCalls := 0, 0, 0

	start := time.Now()

	for time.Since(start) < 3*time.Second {
		ok, _, err := limits.CanCreate(ctx, sqlcv1.LimitResourceTASKRUN, limited, 1)
		if err == nil {
			require.True(t, ok)
			accepted++
		} else {
			require.False(t, ok)
			require.ErrorAs(t, err, &refusal)
			require.ErrorIs(t, err, ErrResourceExhausted)
			require.Positive(t, refusal.RetryAfter)
			require.LessOrEqual(t, refusal.RetryAfter, 20*time.Millisecond, "one item at 50 a second is at most 20 ms away")
			refused++
		}

		ok, _, err = limits.CanCreate(ctx, sqlcv1.LimitResourceEVENT, other, 1)
		require.NoError(t, err)
		require.True(t, ok)
		otherCalls++

		time.Sleep(time.Millisecond)
	}

	want := 50 + 50*time.Since(start).Seconds()
	t.Logf("limited tenant: %d accepted, %d refused, want about %.0f; other tenant: %d of %d accepted", accepted, refused, want, otherCalls, otherCalls)
	require.InDelta(t, want, accepted, 0.05*want)
	require.Greater(t, refused, accepted, "the callers asked much faster than the limit")

	// A bulk push of 500 events into a full bucket is admitted whole and leaves 450 tokens (9 s)
	// of debt, which a refresh with the same limit keeps.
	time.Sleep(time.Second)

	ok, _, err := limits.CanCreate(ctx, sqlcv1.LimitResourceEVENT, limited, 500)
	require.NoError(t, err)
	require.True(t, ok)

	require.NoError(t, repo.refresh(ctx))
	require.ErrorAs(t, repo.Admit(limited, 1), &refusal)
	require.Greater(t, refusal.RetryAfter, 8900*time.Millisecond)

	// A new limit starts a full bucket, without the old limit's debt.
	_, err = repo.Set(ctx, SetTenantControlOpts{TenantId: limited, IngestPerSecond: int32Ptr(10), Reason: "test"})
	require.NoError(t, err)
	require.NoError(t, repo.refresh(ctx))
	require.NoError(t, repo.Admit(limited, 10))
	require.ErrorAs(t, repo.Admit(limited, 1), &refusal)
	require.InDelta(t, 100*time.Millisecond, refusal.RetryAfter, float64(10*time.Millisecond))

	// A pause refuses whatever the bucket holds.
	_, err = repo.Set(ctx, SetTenantControlOpts{TenantId: limited, Paused: true, IngestPerSecond: int32Ptr(10), Reason: "test"})
	require.NoError(t, err)
	require.NoError(t, repo.refresh(ctx))
	require.ErrorIs(t, repo.Admit(limited, 1), ErrTenantPaused)

	// Clearing drops the bucket: the tenant is unlimited, and the same limit set again starts full.
	_, err = repo.Clear(ctx, limited)
	require.NoError(t, err)
	require.NoError(t, repo.refresh(ctx))
	require.Empty(t, tenantControlSeries(t, limited))
	require.NoError(t, repo.Admit(limited, 1000))

	_, err = repo.Set(ctx, SetTenantControlOpts{TenantId: limited, IngestPerSecond: int32Ptr(10), Reason: "test"})
	require.NoError(t, err)
	require.NoError(t, repo.refresh(ctx))
	require.NoError(t, repo.Admit(limited, 10))
	require.Error(t, repo.Admit(limited, 1))

	ok, _, err = limits.CanCreate(ctx, sqlcv1.LimitResourceINCOMINGWEBHOOK, limited, 1)
	require.NoError(t, err)
	require.True(t, ok, "the limit only meters new runs and events")
}

// queuedDeadlineTables are the tables whose rows carry a schedule deadline while they wait.
var queuedDeadlineTables = []string{"v1_queue_item", "v1_concurrency_slot", "v1_batched_queue_item", "v1_rate_limited_queue_items"}

// insertQueuedItems writes one waiting row per table in queuedDeadlineTables for the tenant, each
// with a schedule deadline that is already past.
func insertQueuedItems(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID) {
	t.Helper()

	ctx := context.Background()
	deadline := "NOW() - INTERVAL '30 minutes'"
	itemColumns := "tenant_id, queue, task_id, task_inserted_at, external_id, action_id, step_id, workflow_id, workflow_run_id, schedule_timeout_at, sticky"
	itemValues := "$1, 'default', 1, NOW(), gen_random_uuid(), 'action', gen_random_uuid(), gen_random_uuid(), gen_random_uuid(), " + deadline + ", 'NONE'"

	for _, sql := range []string{
		"INSERT INTO v1_queue_item (" + itemColumns + ") VALUES (" + itemValues + ")",
		"INSERT INTO v1_batched_queue_item (" + itemColumns + ", batch_key) VALUES (" + itemValues + ", 'batch')",
		"INSERT INTO v1_rate_limited_queue_items (" + itemColumns + ", requeue_after) VALUES (" + itemValues + ", NOW())",
		`INSERT INTO v1_concurrency_slot (task_id, task_inserted_at, task_retry_count, external_id, tenant_id, workflow_id,
			workflow_version_id, workflow_run_id, strategy_id, key, queue_to_notify, schedule_timeout_at)
		VALUES (1, NOW(), 0, gen_random_uuid(), $1, gen_random_uuid(), gen_random_uuid(), gen_random_uuid(), 1, 'key', 'default', ` + deadline + ")",
	} {
		_, err := pool.Exec(ctx, sql, tenantID)
		require.NoError(t, err, sql)
	}
}

// queuedDeadlines returns the tenant's schedule deadline in each table of queuedDeadlineTables.
func queuedDeadlines(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID) map[string]time.Time {
	t.Helper()

	deadlines := map[string]time.Time{}

	for _, table := range queuedDeadlineTables {
		var deadline time.Time
		err := pool.QueryRow(context.Background(), "SELECT schedule_timeout_at FROM "+table+" WHERE tenant_id = $1", tenantID).Scan(&deadline)
		require.NoError(t, err, table)
		deadlines[table] = deadline
	}

	return deadlines
}

// backdatePause makes the tenant's current pause look as if it began ago.
func backdatePause(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID, ago time.Duration) {
	t.Helper()

	tag, err := pool.Exec(context.Background(),
		"UPDATE v1_tenant_control SET paused_at = paused_at - make_interval(secs => $2) WHERE tenant_id = $1 AND paused_at IS NOT NULL",
		tenantID, ago.Seconds())
	require.NoError(t, err)
	require.Equal(t, int64(1), tag.RowsAffected())
}

// requireMoved checks that every deadline moved forward by at least by and at most a minute more.
func requireMoved(t *testing.T, before, after map[string]time.Time, by time.Duration) {
	t.Helper()

	for _, table := range queuedDeadlineTables {
		moved := after[table].Sub(before[table])
		require.GreaterOrEqual(t, moved, by, table)
		require.Less(t, moved, by+time.Minute, table)
	}
}

func TestTenantControlResumeMovesScheduleDeadlines(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()
	repo, _, _ := newTenantControlTestRepo(t, pool)
	tenantID := createLimitTestTenant(t, pool)
	other := createLimitTestTenant(t, pool)

	row, err := repo.Set(ctx, SetTenantControlOpts{TenantId: tenantID, Paused: true, Reason: "noisy neighbour"})
	require.NoError(t, err)
	require.True(t, row.PausedAt.Valid)

	backdatePause(t, pool, tenantID, time.Hour)

	// changing the reason or the percent of a paused row keeps paused_at and moves no deadline
	insertQueuedItems(t, pool, tenantID)
	insertQueuedItems(t, pool, other)
	start := queuedDeadlines(t, pool, tenantID)
	otherStart := queuedDeadlines(t, pool, other)

	backdated, err := repo.Set(ctx, SetTenantControlOpts{TenantId: tenantID, Paused: true, DBConnPercent: int32Ptr(10), Reason: "still noisy"})
	require.NoError(t, err)
	require.GreaterOrEqual(t, time.Since(backdated.PausedAt.Time), time.Hour, "a paused row must keep the start of its pause")
	require.Equal(t, start, queuedDeadlines(t, pool, tenantID))

	// Set to unpaused ends the pause: deadlines move by the hour it lasted
	row, err = repo.Set(ctx, SetTenantControlOpts{TenantId: tenantID, DBConnPercent: int32Ptr(10), Reason: "draining"})
	require.NoError(t, err)
	require.False(t, row.PausedAt.Valid)

	resumed := queuedDeadlines(t, pool, tenantID)
	requireMoved(t, start, resumed, time.Hour)
	require.Equal(t, otherStart, queuedDeadlines(t, pool, other), "another tenant's deadlines must not move")

	// Clear of an unpaused row moves nothing
	_, err = repo.Clear(ctx, tenantID)
	require.NoError(t, err)
	require.Equal(t, resumed, queuedDeadlines(t, pool, tenantID))

	// Clear of a paused row ends the pause like Set
	_, err = repo.Set(ctx, SetTenantControlOpts{TenantId: tenantID, Paused: true, Reason: "again"})
	require.NoError(t, err)
	backdatePause(t, pool, tenantID, 2*time.Hour)

	cleared, err := repo.Clear(ctx, tenantID)
	require.NoError(t, err)
	require.True(t, cleared)
	requireMoved(t, resumed, queuedDeadlines(t, pool, tenantID), 2*time.Hour)
	require.Equal(t, otherStart, queuedDeadlines(t, pool, other))
}

// TestTenantControlResumeMovesLargeBacklog ends a pause of a tenant with thousands of queued rows
// through a pool whose statement timeout is far shorter than the shift takes, as the engine's pools
// are for a large enough backlog: the resume uses its own statement timeout.
func TestTenantControlResumeMovesLargeBacklog(t *testing.T) {
	const rows = 10000

	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()

	cfg := pool.Config()
	cfg.ConnConfig.RuntimeParams["statement_timeout"] = "100"
	short, err := pgxpool.NewWithConfig(ctx, cfg)
	require.NoError(t, err)
	t.Cleanup(short.Close)

	repo, _, _ := newTenantControlTestRepo(t, short)
	tenantID := createLimitTestTenant(t, pool)

	deadline := "NOW() - INTERVAL '30 minutes'"
	itemColumns := "tenant_id, queue, task_id, task_inserted_at, external_id, action_id, step_id, workflow_id, workflow_run_id, schedule_timeout_at, sticky"
	itemValues := "$1, 'default', g, NOW(), gen_random_uuid(), 'action', gen_random_uuid(), gen_random_uuid(), gen_random_uuid(), " + deadline + ", 'NONE'"
	series := " FROM generate_series(1, $2) AS g"

	for _, sql := range []string{
		"INSERT INTO v1_queue_item (" + itemColumns + ") SELECT " + itemValues + series,
		"INSERT INTO v1_batched_queue_item (" + itemColumns + ", batch_key) SELECT " + itemValues + ", 'batch'" + series,
		"INSERT INTO v1_rate_limited_queue_items (" + itemColumns + ", requeue_after) SELECT " + itemValues + ", NOW()" + series,
		`INSERT INTO v1_concurrency_slot (task_id, task_inserted_at, task_retry_count, external_id, tenant_id, workflow_id,
			workflow_version_id, workflow_run_id, strategy_id, key, queue_to_notify, schedule_timeout_at)
		SELECT g, NOW(), 0, gen_random_uuid(), $1, gen_random_uuid(), gen_random_uuid(), gen_random_uuid(), 1, 'key', 'default', ` + deadline + series,
	} {
		_, err := pool.Exec(ctx, sql, tenantID, rows)
		require.NoError(t, err, sql)
	}

	_, err = repo.Set(ctx, SetTenantControlOpts{TenantId: tenantID, Paused: true, Reason: "noisy neighbour"})
	require.NoError(t, err)
	backdatePause(t, pool, tenantID, time.Hour)

	// the shift alone does not fit in the pool's statement timeout
	tx, err := short.Begin(ctx)
	require.NoError(t, err)
	err = repo.queries.ShiftScheduleTimeouts(ctx, tx, sqlcv1.ShiftScheduleTimeoutsParams{
		Tenantid: tenantID,
		Pausedat: pgtype.Timestamptz{Time: time.Now().Add(-time.Hour), Valid: true},
	})
	require.ErrorContains(t, err, "statement timeout")
	require.NoError(t, tx.Rollback(ctx))

	start := time.Now()
	_, err = repo.Set(ctx, SetTenantControlOpts{TenantId: tenantID, Reason: "resumed"})
	elapsed := time.Since(start)
	require.NoError(t, err)

	t.Logf("resume of %d rows in each of %d tables took %s", rows, len(queuedDeadlineTables), elapsed)

	// every deadline was 30 minutes past and moved by the hour of the pause
	for _, table := range queuedDeadlineTables {
		var moved int
		err := pool.QueryRow(ctx,
			"SELECT count(*) FROM "+table+" WHERE tenant_id = $1 AND schedule_timeout_at > NOW() + INTERVAL '29 minutes'",
			tenantID).Scan(&moved)
		require.NoError(t, err, table)
		require.Equal(t, rows, moved, table)
	}
}
