package streams

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
)

type fakeEntitlements struct {
	v1.TenantEntitlementRepository

	entitled map[uuid.UUID]bool
	calls    int
}

func (f *fakeEntitlements) IsDurableStreamsEnabled(_ context.Context, tenantId uuid.UUID) (bool, error) {
	f.calls++
	return f.entitled[tenantId], nil
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
		repo:     &entitlementRepository{entitlements: entitlements},
		entitled: expirable.NewLRU[uuid.UUID, bool](10, nil, time.Minute),
	}

	require.NoError(t, s.checkEntitled(context.Background(), entitledTenant))
	require.NoError(t, s.checkEntitled(context.Background(), entitledTenant))

	err := s.checkEntitled(context.Background(), otherTenant)
	assert.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))
	assert.ErrorContains(t, err, "durable streams are not enabled")

	assert.Equal(t, 2, entitlements.calls, "each tenant's entitlement is looked up once, then cached")
}
