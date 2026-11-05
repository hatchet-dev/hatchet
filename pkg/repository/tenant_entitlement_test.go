//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
	"sync"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/repository/fairpool"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func TestHasEntitlement(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()
	logger := zerolog.Nop()
	repo := newTenantEntitlementRepository(&sharedRepository{pool: fairpool.Ungated(pool), l: &logger, queries: sqlcv1.New()})
	tenantID := createLimitTestTenant(t, pool)

	enabled, err := repo.HasEntitlement(ctx, tenantID, EntitlementDurableStreams)
	require.NoError(t, err)
	assert.False(t, enabled, "tenants without an entitlement row aren't entitled")

	none, err := repo.GetEntitlements(ctx, tenantID)
	require.NoError(t, err)
	assert.Equal(t, TenantEntitlements{}, none)

	want := TenantEntitlements{DurableStreams: true, ServerlessOperator: true, SandboxOperator: true}
	require.NoError(t, repo.SetEntitlements(ctx, tenantID, want))

	enabled, err = repo.HasEntitlement(ctx, tenantID, EntitlementDurableStreams)
	require.NoError(t, err)
	assert.True(t, enabled)

	enabled, err = repo.HasEntitlement(ctx, tenantID, EntitlementServerlessOperator)
	require.NoError(t, err)
	assert.True(t, enabled)

	enabled, err = repo.HasEntitlement(ctx, tenantID, EntitlementSandboxOperator)
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

func TestUpdateLabsEntitlementsPreservesOtherFlags(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()
	logger := zerolog.Nop()
	repo := newTenantEntitlementRepository(&sharedRepository{pool: fairpool.Ungated(pool), l: &logger, queries: sqlcv1.New()})
	tenantID := createLimitTestTenant(t, pool)

	require.NoError(t, repo.SetEntitlements(ctx, tenantID, TenantEntitlements{AuditLogs: true}))

	enabled, err := repo.HasEntitlement(ctx, tenantID, EntitlementDurableStreams)
	require.NoError(t, err)
	assert.False(t, enabled)

	durableStreams := true
	sandboxes := true
	got, err := repo.UpdateLabsEntitlements(ctx, tenantID, LabsEntitlements{
		DurableStreams:  &durableStreams,
		SandboxOperator: &sandboxes,
	})
	require.NoError(t, err)
	assert.True(t, got.AuditLogs)
	assert.True(t, got.DurableStreams)
	assert.True(t, got.SandboxOperator)
	assert.False(t, got.ServerlessOperator)

	stored, err := repo.GetEntitlements(ctx, tenantID)
	require.NoError(t, err)
	assert.Equal(t, got, stored)

	enabled, err = repo.HasEntitlement(ctx, tenantID, EntitlementDurableStreams)
	require.NoError(t, err)
	assert.True(t, enabled, "the writer drops its entitlement cache")

	enabled, err = repo.HasEntitlement(ctx, tenantID, EntitlementSandboxOperator)
	require.NoError(t, err)
	assert.True(t, enabled)
}

func TestUpdateLabsEntitlementsConcurrentPatches(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()
	logger := zerolog.Nop()
	repo := newTenantEntitlementRepository(&sharedRepository{pool: fairpool.Ungated(pool), l: &logger, queries: sqlcv1.New()})
	tenantID := createLimitTestTenant(t, pool)

	require.NoError(t, repo.SetEntitlements(ctx, tenantID, TenantEntitlements{AuditLogs: true}))

	enabled := true
	patches := []LabsEntitlements{
		{DurableStreams: &enabled},
		{ServerlessOperator: &enabled},
		{SandboxOperator: &enabled},
	}

	var wg sync.WaitGroup
	errs := make(chan error, len(patches)*10)

	for range 10 {
		for _, patch := range patches {
			wg.Add(1)

			go func() {
				defer wg.Done()

				_, err := repo.UpdateLabsEntitlements(ctx, tenantID, patch)
				errs <- err
			}()
		}
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		require.NoError(t, err)
	}

	got, err := repo.GetEntitlements(ctx, tenantID)
	require.NoError(t, err)
	assert.Equal(t, TenantEntitlements{
		AuditLogs:          true,
		DurableStreams:     true,
		ServerlessOperator: true,
		SandboxOperator:    true,
	}, got)
}
