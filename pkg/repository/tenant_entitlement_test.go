//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func TestHasEntitlement(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()
	logger := zerolog.Nop()
	repo := newTenantEntitlementRepository(&sharedRepository{pool: pool, l: &logger, queries: sqlcv1.New()})
	tenantID := createLimitTestTenant(t, pool)

	enabled, err := repo.HasEntitlement(ctx, tenantID, EntitlementDurableStreams)
	require.NoError(t, err)
	assert.False(t, enabled, "tenants without an entitlement row aren't entitled")

	none, err := repo.GetEntitlements(ctx, tenantID)
	require.NoError(t, err)
	assert.Equal(t, TenantEntitlements{}, none)

	want := TenantEntitlements{DurableStreams: true, DAGOperator: true}
	require.NoError(t, repo.SetEntitlements(ctx, tenantID, want))

	enabled, err = repo.HasEntitlement(ctx, tenantID, EntitlementDurableStreams)
	require.NoError(t, err)
	assert.True(t, enabled)

	enabled, err = repo.HasEntitlement(ctx, tenantID, EntitlementAuditLogs)
	require.NoError(t, err)
	assert.False(t, enabled)

	got, err := repo.GetEntitlements(ctx, tenantID)
	require.NoError(t, err)
	assert.Equal(t, want, got)

	_, err = repo.HasEntitlement(ctx, tenantID, Entitlement("nope"))
	assert.Error(t, err)
}
