//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	promclient "github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/integrations/metrics/prometheus"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// newTenantControlTestRepo builds a control repository on the shared repository of a tenant limit
// repository that does not enforce limits, so CanCreate consults only the controls.
func newTenantControlTestRepo(t *testing.T, pool *pgxpool.Pool) (*tenantControlRepository, *tenantLimitRepository) {
	t.Helper()

	limits := createTenantLimitRepositoryForTest(t, pool, defaultLimitTestConfig())
	limits.enforceLimits = false
	limits.tenantControls = newTenantControlRepository(limits.sharedRepository)

	return limits.tenantControls, limits
}

func perSecond(n int32) *int32 {
	return &n
}

// tenantControlSeries returns one tenant's series of a tenant control metric, keyed by control.
func tenantControlSeries(t *testing.T, collector promclient.Collector, tenantID uuid.UUID) map[string]float64 {
	t.Helper()

	ch := make(chan promclient.Metric)
	go func() {
		collector.Collect(ch)
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

		if labels["tenant_id"] != tenantID.String() {
			continue
		}

		if m.GetGauge() != nil {
			series[labels["control"]] = m.GetGauge().GetValue()
		} else {
			series[labels["control"]] = m.GetCounter().GetValue()
		}
	}

	return series
}

func TestTenantControlSetClearList(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()
	repo, _ := newTenantControlTestRepo(t, pool)
	tenantID := createLimitTestTenant(t, pool)

	row, err := repo.Set(ctx, SetTenantControlOpts{TenantId: tenantID, IngestPerSecond: perSecond(500), Reason: "noisy tenant"})
	require.NoError(t, err)
	require.Equal(t, pgtype.Int4{Int32: 500, Valid: true}, row.IngestPerSecond)
	require.Equal(t, "noisy tenant", row.Reason)

	// set replaces the row
	row, err = repo.Set(ctx, SetTenantControlOpts{TenantId: tenantID, IngestPerSecond: perSecond(50), Reason: "still noisy"})
	require.NoError(t, err)
	require.Equal(t, pgtype.Int4{Int32: 50, Valid: true}, row.IngestPerSecond)

	rows, err := repo.List(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, tenantID, rows[0].TenantID)
	require.Equal(t, "still noisy", rows[0].Reason)

	cleared, err := repo.Clear(ctx, tenantID)
	require.NoError(t, err)
	require.True(t, cleared)

	cleared, err = repo.Clear(ctx, tenantID)
	require.NoError(t, err)
	require.False(t, cleared)

	rows, err = repo.List(ctx)
	require.NoError(t, err)
	require.Empty(t, rows)

	_, err = repo.Set(ctx, SetTenantControlOpts{TenantId: uuid.New(), IngestPerSecond: perSecond(10), Reason: "unknown tenant"})
	require.Error(t, err, "a control for a tenant that does not exist must be refused")

	_, err = repo.Set(ctx, SetTenantControlOpts{TenantId: tenantID, IngestPerSecond: perSecond(0), Reason: "out of range"})
	require.Error(t, err, "ingest_per_second below 1 must be refused")

	// deleting the tenant deletes its control
	_, err = repo.Set(ctx, SetTenantControlOpts{TenantId: tenantID, IngestPerSecond: perSecond(10), Reason: "test"})
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `DELETE FROM "Tenant" WHERE id = $1`, tenantID)
	require.NoError(t, err)

	rows, err = repo.List(ctx)
	require.NoError(t, err)
	require.Empty(t, rows)
}

func TestTenantControlRefresh(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()
	repo, _ := newTenantControlTestRepo(t, pool)
	tenantID := createLimitTestTenant(t, pool)

	_, err := repo.Set(ctx, SetTenantControlOpts{TenantId: tenantID, IngestPerSecond: perSecond(1), Reason: "test"})
	require.NoError(t, err)
	require.NoError(t, repo.Admit(tenantID, 100), "a process that has not refreshed treats every tenant as uncontrolled")

	require.NoError(t, repo.refresh(ctx))
	require.Equal(t, map[string]float64{"ingest": 1}, tenantControlSeries(t, prometheus.TenantControl, tenantID))
	require.Equal(t, map[string]float64{"ingest": 0}, tenantControlSeries(t, prometheus.TenantControlRefused, tenantID))

	require.NoError(t, repo.Admit(tenantID, 1))

	var refusal *TenantRateLimitedError
	require.ErrorAs(t, repo.Admit(tenantID, 3), &refusal)
	require.Equal(t, map[string]float64{"ingest": 3}, tenantControlSeries(t, prometheus.TenantControlRefused, tenantID), "the counter counts refused items")

	// a failed refresh keeps the last snapshot
	_, err = repo.Clear(ctx, tenantID)
	require.NoError(t, err)

	canceled, cancel := context.WithCancel(ctx)
	cancel()
	require.Error(t, repo.refresh(canceled))
	require.ErrorAs(t, repo.Admit(tenantID, 1), &refusal)

	require.NoError(t, repo.refresh(ctx))
	require.NoError(t, repo.Admit(tenantID, 100))
	require.Empty(t, tenantControlSeries(t, prometheus.TenantControl, tenantID))
	require.Empty(t, tenantControlSeries(t, prometheus.TenantControlRefused, tenantID))
}

func TestTenantControlRefreshLoopAppliesAChange(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	repo, _ := newTenantControlTestRepo(t, pool)
	tenantID := createLimitTestTenant(t, pool)

	go repo.RefreshLoop(ctx)

	// let the loop's first read pass, so the change below waits for a tick
	time.Sleep(100 * time.Millisecond)

	_, err := repo.Set(ctx, SetTenantControlOpts{TenantId: tenantID, IngestPerSecond: perSecond(1), Reason: "test"})
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		return repo.Admit(tenantID, 2) != nil
	}, 5*tenantControlRefreshInterval, 10*time.Millisecond)
}

// Admission reads only the snapshot: with the database gone, a controlled tenant is still limited
// and every other tenant is still admitted.
func TestCanCreateDoesNotQueryTheDatabaseForControls(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()
	repo, limits := newTenantControlTestRepo(t, pool)
	limited := createLimitTestTenant(t, pool)
	other := createLimitTestTenant(t, pool)

	_, err := repo.Set(ctx, SetTenantControlOpts{TenantId: limited, IngestPerSecond: perSecond(1), Reason: "test"})
	require.NoError(t, err)
	require.NoError(t, repo.refresh(ctx))

	pool.Close()

	ok, _, err := limits.CanCreate(ctx, sqlcv1.LimitResourceTASKRUN, limited, 1)
	require.NoError(t, err)
	require.True(t, ok)

	var refusal *TenantRateLimitedError
	ok, _, err = limits.CanCreate(ctx, sqlcv1.LimitResourceTASKRUN, limited, 1)
	require.False(t, ok)
	require.ErrorAs(t, err, &refusal)

	ok, _, err = limits.CanCreate(ctx, sqlcv1.LimitResourceEVENT, other, 1000)
	require.NoError(t, err)
	require.True(t, ok)
}

func TestCanCreateMetersIngestRate(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()
	repo, limits := newTenantControlTestRepo(t, pool)

	limited := createLimitTestTenant(t, pool)
	other := createLimitTestTenant(t, pool)

	_, err := repo.Set(ctx, SetTenantControlOpts{TenantId: limited, IngestPerSecond: perSecond(50), Reason: "test"})
	require.NoError(t, err)
	require.NoError(t, repo.refresh(ctx))

	// Both tenants ask as fast as they can for three seconds, the limited one alternating runs and
	// events. It gets the bucket's one second of tokens up front and 50 a second after that; the
	// other tenant gets everything.
	var refusal *TenantRateLimitedError
	accepted, refused := 0, 0

	start := time.Now()

	for i := 0; time.Since(start) < 3*time.Second; i++ {
		resource := sqlcv1.LimitResourceTASKRUN
		if i%2 == 1 {
			resource = sqlcv1.LimitResourceEVENT
		}

		ok, _, err := limits.CanCreate(ctx, resource, limited, 1)
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

		time.Sleep(time.Millisecond)
	}

	want := 50 + 50*time.Since(start).Seconds()
	require.InDelta(t, want, accepted, 0.05*want)
	require.Greater(t, refused, accepted, "the callers asked much faster than the limit")
	require.Equal(t, map[string]float64{"ingest": float64(refused)}, tenantControlSeries(t, prometheus.TenantControlRefused, limited))

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
	_, err = repo.Set(ctx, SetTenantControlOpts{TenantId: limited, IngestPerSecond: perSecond(10), Reason: "test"})
	require.NoError(t, err)
	require.NoError(t, repo.refresh(ctx))
	require.NoError(t, repo.Admit(limited, 10))
	require.ErrorAs(t, repo.Admit(limited, 1), &refusal)
	require.InDelta(t, 100*time.Millisecond, refusal.RetryAfter, float64(10*time.Millisecond))

	// Clearing drops the bucket: the tenant is unlimited, and the same limit set again starts full.
	_, err = repo.Clear(ctx, limited)
	require.NoError(t, err)
	require.NoError(t, repo.refresh(ctx))
	require.NoError(t, repo.Admit(limited, 1000))

	_, err = repo.Set(ctx, SetTenantControlOpts{TenantId: limited, IngestPerSecond: perSecond(10), Reason: "test"})
	require.NoError(t, err)
	require.NoError(t, repo.refresh(ctx))
	require.NoError(t, repo.Admit(limited, 10))
	require.Error(t, repo.Admit(limited, 1))

	ok, _, err = limits.CanCreate(ctx, sqlcv1.LimitResourceINCOMINGWEBHOOK, limited, 1)
	require.NoError(t, err)
	require.True(t, ok, "the limit only meters new runs and events")
}
