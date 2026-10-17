package repository

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/repository/cache"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/validator"
)

func newTenantAlertingTestRepo(pool *pgxpool.Pool) *tenantAlertingRepository {
	l := zerolog.Nop()

	return &tenantAlertingRepository{
		sharedRepository: &sharedRepository{
			pool:    pool,
			ddlPool: pool,
			v:       validator.NewDefaultValidator(),
			l:       &l,
			queries: sqlcv1.New(),
		},
		cache: cache.New(0),
	}
}

// seedTenantWithLimitAlert creates a tenant, a TASK_RUN limit and one alert
// row for it, returning the tenant id.
func seedTenantWithLimitAlert(t *testing.T, pool *pgxpool.Pool) uuid.UUID {
	t.Helper()
	ctx := context.Background()

	tenantID := uuid.New()
	_, err := pool.Exec(ctx, `INSERT INTO "Tenant" ("id", "name", "slug") VALUES ($1, $2, $3)`, tenantID, "Limit Alert Tenant", "limit-alert-"+tenantID.String()[:8])
	require.NoError(t, err)

	limitID := uuid.New()
	_, err = pool.Exec(ctx, `
		INSERT INTO "TenantResourceLimit" ("id", "resource", "tenantId", "limitValue", "alarmValue", "value", "window", "lastRefill")
		VALUES ($1, 'TASK_RUN', $2, 2000, 1600, 1700, '24h0m0s', NOW())
	`, limitID, tenantID)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `
		INSERT INTO "TenantResourceLimitAlert" ("id", "resourceLimitId", "tenantId", "resource", "alertType", "value", "limit")
		VALUES ($1, $2, $3, 'TASK_RUN', 'Alarm', 1700, 2000)
	`, uuid.New(), limitID, tenantID)
	require.NoError(t, err)

	return tenantID
}

func countLimitAlerts(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID) int {
	t.Helper()

	var n int
	require.NoError(t, pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM "TenantResourceLimitAlert" WHERE "tenantId" = $1`, tenantID).Scan(&n))

	return n
}

func boolPtr(b bool) *bool { return &b }

func TestUpsertTenantAlertingSettings_EnablingLimitAlertsClearsAlertLog(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()
	repo := newTenantAlertingTestRepo(pool)
	tenantID := seedTenantWithLimitAlert(t, pool)

	// Start with alerts disabled. The row created above represents a crossing
	// recorded while alerts were off.
	_, err := repo.UpsertTenantAlertingSettings(ctx, tenantID, &UpsertTenantAlertingSettingsOpts{
		EnableTenantResourceLimitAlerts: boolPtr(false),
	})
	require.NoError(t, err)
	require.Equal(t, 1, countLimitAlerts(t, pool, tenantID))

	// Re-saving while still disabled, or saving unrelated settings, keeps the log.
	_, err = repo.UpsertTenantAlertingSettings(ctx, tenantID, &UpsertTenantAlertingSettingsOpts{
		EnableTenantResourceLimitAlerts: boolPtr(false),
	})
	require.NoError(t, err)
	_, err = repo.UpsertTenantAlertingSettings(ctx, tenantID, &UpsertTenantAlertingSettingsOpts{
		EnableExpiringTokenAlerts: boolPtr(false),
	})
	require.NoError(t, err)
	require.Equal(t, 1, countLimitAlerts(t, pool, tenantID))

	// Turning alerts on clears the log so the next poll re-alerts.
	settings, err := repo.UpsertTenantAlertingSettings(ctx, tenantID, &UpsertTenantAlertingSettingsOpts{
		EnableTenantResourceLimitAlerts: boolPtr(true),
	})
	require.NoError(t, err)
	require.True(t, settings.EnableTenantResourceLimitAlerts)
	require.Equal(t, 0, countLimitAlerts(t, pool, tenantID))
}

func TestUpsertTenantAlertingSettings_EnabledToEnabledKeepsAlertLog(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()
	repo := newTenantAlertingTestRepo(pool)
	tenantID := seedTenantWithLimitAlert(t, pool)

	// No settings row yet: the default is enabled, so enabling is not a
	// transition and the dedupe log must survive.
	_, err := repo.UpsertTenantAlertingSettings(ctx, tenantID, &UpsertTenantAlertingSettingsOpts{
		EnableTenantResourceLimitAlerts: boolPtr(true),
	})
	require.NoError(t, err)
	require.Equal(t, 1, countLimitAlerts(t, pool, tenantID))

	// Enabled to enabled with an existing row is not a transition either.
	_, err = repo.UpsertTenantAlertingSettings(ctx, tenantID, &UpsertTenantAlertingSettingsOpts{
		EnableTenantResourceLimitAlerts: boolPtr(true),
	})
	require.NoError(t, err)
	require.Equal(t, 1, countLimitAlerts(t, pool, tenantID))
}
