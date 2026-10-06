package streams

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
)

type fakeEntitlements struct {
	v1.TenantEntitlementRepository

	entitled map[uuid.UUID]bool
}

func (f *fakeEntitlements) HasEntitlement(_ context.Context, tenantId uuid.UUID, entitlement v1.Entitlement) (bool, error) {
	return entitlement == v1.EntitlementDurableStreams && f.entitled[tenantId], nil
}

type entitlementRepository struct {
	v1.Repository
	entitlements *fakeEntitlements
}

func (r *entitlementRepository) TenantEntitlement() v1.TenantEntitlementRepository {
	return r.entitlements
}

func TestCheckEntitled(t *testing.T) {
	entitledTenant, otherTenant := uuid.New(), uuid.New()
	entitlements := &fakeEntitlements{entitled: map[uuid.UUID]bool{entitledTenant: true}}

	s := &ServiceImpl{
		repo: &entitlementRepository{entitlements: entitlements},
	}

	require.NoError(t, s.checkEntitled(context.Background(), entitledTenant))

	err := s.checkEntitled(context.Background(), otherTenant)
	assert.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))
	assert.ErrorContains(t, err, "durable streams are not enabled")
}
