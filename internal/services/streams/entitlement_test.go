package streams

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
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

func TestStreamsRejectTenantsWithoutTheEntitlement(t *testing.T) {
	entitledTenant, otherTenant := uuid.New(), uuid.New()
	entitlements := &fakeEntitlements{entitled: map[uuid.UUID]bool{entitledTenant: true}}

	s := &ServiceImpl{
		repo: &entitlementRepository{entitlements: entitlements},
	}

	asTenant := func(id uuid.UUID) context.Context {
		return context.WithValue(context.Background(), "tenant", &sqlcv1.Tenant{ID: id}) // nolint:staticcheck
	}

	// no topic: an entitled tenant gets past the check and fails validation instead
	_, err := s.GetTopicMetadata(asTenant(entitledTenant), &contracts.GetStreamTopicMetadataRequest{})
	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))

	_, err = s.GetTopicMetadata(asTenant(otherTenant), &contracts.GetStreamTopicMetadataRequest{Topic: "t"})
	assert.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))
	assert.ErrorContains(t, err, "durable streams are not enabled")
}

// The in-process entry makes the same check as the RPC: a tenant without the entitlement gets
// a stream that ends at once with PermissionDenied as its result, and a context without a
// tenant is refused before anything runs.
func TestRegisterSubscribeRejectsTenantsWithoutTheEntitlement(t *testing.T) {
	otherTenant := uuid.New()

	s := &ServiceImpl{
		repo: &entitlementRepository{entitlements: &fakeEntitlements{entitled: map[uuid.UUID]bool{}}},
	}

	_, err := s.RegisterSubscribe(context.Background(), &contracts.SubscribeStreamRequest{Namespace: "ns", Topic: "t"})
	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))

	ctx := context.WithValue(context.Background(), "tenant", &sqlcv1.Tenant{ID: otherTenant}) // nolint:staticcheck

	cs, err := s.RegisterSubscribe(ctx, &contracts.SubscribeStreamRequest{Namespace: "ns", Topic: "t"})
	require.NoError(t, err)
	assert.Nil(t, cs.Requests, "Subscribe is a server stream")

	for range cs.Responses {
		t.Fatal("a refused subscription sends nothing")
	}

	assert.Equal(t, connect.CodePermissionDenied, connect.CodeOf(cs.Err))
	assert.ErrorContains(t, cs.Err, "durable streams are not enabled")
}
