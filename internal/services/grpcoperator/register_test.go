package grpcoperator

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	v1contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// The handler maps the request onto the operator service and its answer back onto the response;
// what registration does is covered by the operator service's own tests.
func TestRegisterAnswersWithTheAssignedIdentity(t *testing.T) {
	tenant := &sqlcv1.Tenant{ID: uuid.New()}
	svc := newTestService(t, nil)

	resp, err := svc.Register(tenantContext(tenant), &v1contracts.OperatorRegisterRequest{
		Name:       "my-operator",
		SlotConfig: map[string]int32{"default": 5},
	})

	require.NoError(t, err)
	assert.Equal(t, tenant.ID.String(), resp.TenantId)
	assert.False(t, resp.Resumed)

	op, err := svc.operators.GetOperatorById(t.Context(), uuid.MustParse(resp.OperatorId))
	require.NoError(t, err)
	assert.Equal(t, "my-operator", op.Name)
	assert.Equal(t, sqlcv1.V1OperatorKindGRPC, op.Kind)

	created := svc.workers.Created()
	require.Len(t, created, 1)
	assert.Equal(t, map[string]int32{"default": 5}, created[0].SlotConfig)

	workerId := resp.WorkerId

	resumed, err := svc.Register(tenantContext(tenant), &v1contracts.OperatorRegisterRequest{
		Name:     "my-operator",
		WorkerId: &workerId,
	})
	require.NoError(t, err)
	assert.True(t, resumed.Resumed)
	assert.Equal(t, workerId, resumed.WorkerId)
}

func TestRegisterRejects(t *testing.T) {
	tenant := &sqlcv1.Tenant{ID: uuid.New()}

	t.Run("missing tenant", func(t *testing.T) {
		svc := newTestService(t, nil)
		_, err := svc.Register(t.Context(), &v1contracts.OperatorRegisterRequest{Name: "op"})
		assert.Equal(t, codes.Unauthenticated, status.Code(err))
	})

	t.Run("invalid name", func(t *testing.T) {
		svc := newTestService(t, nil)
		_, err := svc.Register(tenantContext(tenant), &v1contracts.OperatorRegisterRequest{Name: "bad name!"})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
	})

	// the worker id is a protocol field, so a malformed one is refused before anything is
	// written; an empty one asks for a new worker
	t.Run("malformed worker id", func(t *testing.T) {
		svc := newTestService(t, nil)
		malformed := "not-a-uuid"

		_, err := svc.Register(tenantContext(tenant), &v1contracts.OperatorRegisterRequest{Name: "op", WorkerId: &malformed})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		assert.Zero(t, svc.operators.Count())
	})

	t.Run("empty worker id creates a worker", func(t *testing.T) {
		svc := newTestService(t, nil)
		empty := ""

		resp, err := svc.Register(tenantContext(tenant), &v1contracts.OperatorRegisterRequest{Name: "op", WorkerId: &empty})
		require.NoError(t, err)
		assert.False(t, resp.Resumed)
	})
}
