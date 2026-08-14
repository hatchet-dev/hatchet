//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/config/limits"
	"github.com/hatchet-dev/hatchet/pkg/repository/cache"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// upsertAffectedFields are the columns UpsertTenantResourceLimits updates on conflict
// (limit/alarm/updatedAt). Window/customValueMeter are also updated on upsert, but
// insert-only self-heal tests only need these fields to detect a DO UPDATE clobber.
type upsertAffectedFields struct {
	LimitValue int32
	AlarmValue pgtype.Int4
	UpdatedAt  time.Time
}

func createTenantLimitRepositoryForTest(t *testing.T, pool *pgxpool.Pool, config limits.LimitConfigFile) *tenantLimitRepository {
	t.Helper()

	logger := zerolog.Nop()
	c := cache.New(time.Minute)
	t.Cleanup(c.Stop)

	return &tenantLimitRepository{
		sharedRepository: &sharedRepository{
			pool:    pool,
			ddlPool: pool,
			l:       &logger,
			queries: sqlcv1.New(),
		},
		config:        config,
		enforceLimits: true,
		c:             c,
		unflushed:     make(meterSet),
	}
}

func defaultLimitTestConfig() limits.LimitConfigFile {
	return limits.LimitConfigFile{
		DefaultTaskRunLimit:              100,
		DefaultTaskRunAlarmLimit:         80,
		DefaultTaskRunWindow:             24 * time.Hour,
		DefaultEventLimit:                50,
		DefaultEventAlarmLimit:           40,
		DefaultEventWindow:               24 * time.Hour,
		DefaultWorkerLimit:               3,
		DefaultWorkerAlarmLimit:          2,
		DefaultWorkerSlotLimit:           100,
		DefaultWorkerSlotAlarmLimit:      80,
		DefaultIncomingWebhookLimit:      5,
		DefaultIncomingWebhookAlarmLimit: 4,
	}
}

func createLimitTestTenant(t *testing.T, pool *pgxpool.Pool) uuid.UUID {
	t.Helper()

	tenantID := uuid.New()
	_, err := pool.Exec(context.Background(), `
		INSERT INTO "Tenant" ("id", "name", "slug", "createdAt", "updatedAt")
		VALUES ($1, $2, $3, NOW(), NOW())
	`, tenantID, "limit-test-"+tenantID.String(), "limit-test-"+tenantID.String())
	require.NoError(t, err)

	return tenantID
}

func loadUpsertAffectedFields(ctx context.Context, pool *pgxpool.Pool, tenantID uuid.UUID, resource sqlcv1.LimitResource) (upsertAffectedFields, error) {
	var fields upsertAffectedFields
	var updatedAt pgtype.Timestamp

	err := pool.QueryRow(ctx, `
		SELECT "limitValue", "alarmValue", "updatedAt"
		FROM "TenantResourceLimit"
		WHERE "tenantId" = $1 AND "resource" = $2::"LimitResource"
	`, tenantID, string(resource)).Scan(&fields.LimitValue, &fields.AlarmValue, &updatedAt)
	if err != nil {
		return upsertAffectedFields{}, err
	}

	if !updatedAt.Valid {
		return upsertAffectedFields{}, errors.New("updatedAt is NULL")
	}
	fields.UpdatedAt = updatedAt.Time.UTC()

	return fields, nil
}

func requireUpsertAffectedFields(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID, resource sqlcv1.LimitResource) upsertAffectedFields {
	t.Helper()

	fields, err := loadUpsertAffectedFields(context.Background(), pool, tenantID, resource)
	require.NoError(t, err)
	return fields
}

func TestCanCreate_MissingIncomingWebhook_PreservesPaidLimits(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()
	tenantID := createLimitTestTenant(t, pool)

	const paidTaskRunLimit int32 = 10000
	const paidTaskRunAlarm int32 = 8000
	const paidEventLimit int32 = 5000
	const paidEventAlarm int32 = 4000

	// Fixed past updatedAt so an upsert DO UPDATE would visibly move the timestamp.
	paidUpdatedAt := time.Now().UTC().Add(-2 * time.Hour).Truncate(time.Millisecond)

	_, err := pool.Exec(ctx, `
		INSERT INTO "TenantResourceLimit" (
			"id", "createdAt", "updatedAt", "tenantId", "resource", "value",
			"limitValue", "alarmValue", "window", "customValueMeter", "lastRefill"
		) VALUES
			(gen_random_uuid(), $4, $4, $1, 'TASK_RUN', 0, $2, $5, '24h0m0s', false, $4),
			(gen_random_uuid(), $4, $4, $1, 'EVENT', 0, $3, $6, '24h0m0s', false, $4)
	`, tenantID, paidTaskRunLimit, paidEventLimit, paidUpdatedAt, paidTaskRunAlarm, paidEventAlarm)
	require.NoError(t, err)

	cfg := defaultLimitTestConfig()
	// Defaults are far below the paid rows; upsert would clobber limit/alarm and bump updatedAt.
	require.Less(t, cfg.DefaultTaskRunLimit, paidTaskRunLimit)
	require.Less(t, cfg.DefaultTaskRunAlarmLimit, paidTaskRunAlarm)
	require.Less(t, cfg.DefaultEventLimit, paidEventLimit)
	require.Less(t, cfg.DefaultEventAlarmLimit, paidEventAlarm)

	beforeTaskRun := requireUpsertAffectedFields(t, pool, tenantID, sqlcv1.LimitResourceTASKRUN)
	beforeEvent := requireUpsertAffectedFields(t, pool, tenantID, sqlcv1.LimitResourceEVENT)

	repo := createTenantLimitRepositoryForTest(t, pool, cfg)

	ok, _, err := repo.canCreate(ctx, nil, sqlcv1.LimitResourceINCOMINGWEBHOOK, tenantID, 1)
	require.NoError(t, err)
	assert.True(t, ok)

	afterTaskRun := requireUpsertAffectedFields(t, pool, tenantID, sqlcv1.LimitResourceTASKRUN)
	afterEvent := requireUpsertAffectedFields(t, pool, tenantID, sqlcv1.LimitResourceEVENT)

	assert.Equal(t, beforeTaskRun, afterTaskRun, "insert-only self-heal must not change upsert-affected TASK_RUN fields")
	assert.Equal(t, beforeEvent, afterEvent, "insert-only self-heal must not change upsert-affected EVENT fields")
	assert.Equal(t, paidTaskRunLimit, afterTaskRun.LimitValue)
	assert.Equal(t, paidTaskRunAlarm, afterTaskRun.AlarmValue.Int32)
	assert.True(t, afterTaskRun.AlarmValue.Valid)
	assert.Equal(t, paidEventLimit, afterEvent.LimitValue)
	assert.Equal(t, paidEventAlarm, afterEvent.AlarmValue.Int32)
	assert.True(t, afterEvent.AlarmValue.Valid)

	webhook, err := loadUpsertAffectedFields(ctx, pool, tenantID, sqlcv1.LimitResourceINCOMINGWEBHOOK)
	require.NoError(t, err)
	assert.Equal(t, cfg.DefaultIncomingWebhookLimit, webhook.LimitValue)
}

func TestCanCreate_MissingIncomingWebhook_ReReadsAndEnforcesDefault(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()
	tenantID := createLimitTestTenant(t, pool)
	cfg := defaultLimitTestConfig()
	repo := createTenantLimitRepositoryForTest(t, pool, cfg)

	_, err := loadUpsertAffectedFields(ctx, pool, tenantID, sqlcv1.LimitResourceINCOMINGWEBHOOK)
	require.ErrorIs(t, err, pgx.ErrNoRows, "webhook limit must be absent before self-heal")

	// Old bug returned true immediately after inserting defaults. With value=0 and
	// limit=5, requesting 6 must deny: value+n-1 = 5 >= 5.
	ok, percent, err := repo.canCreate(ctx, nil, sqlcv1.LimitResourceINCOMINGWEBHOOK, tenantID, 6)
	require.NoError(t, err)
	assert.False(t, ok, "inserted default must be re-read and enforced, not auto-allowed")
	assert.Equal(t, 100, percent)

	webhook := requireUpsertAffectedFields(t, pool, tenantID, sqlcv1.LimitResourceINCOMINGWEBHOOK)
	assert.Equal(t, cfg.DefaultIncomingWebhookLimit, webhook.LimitValue)
	assert.True(t, webhook.AlarmValue.Valid)
	assert.Equal(t, cfg.DefaultIncomingWebhookAlarmLimit, webhook.AlarmValue.Int32)
}
