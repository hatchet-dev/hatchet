package grpcoperator

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	v1contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func TestRegisterCreatesWorker(t *testing.T) {
	tenant := &sqlcv1.Tenant{ID: uuid.New()}
	svc := newTestService(t, nil)

	label := "gpu"

	resp, err := svc.Register(tenantContext(tenant), &v1contracts.OperatorRegisterRequest{
		Name:       "my-operator",
		SlotConfig: map[string]int32{"default": 5},
		Labels:     map[string]*contracts.WorkerLabels{"kind": {StrValue: &label}},
	})

	require.NoError(t, err)
	assert.Equal(t, tenant.ID.String(), resp.TenantId)
	assert.False(t, resp.Resumed)

	op, err := svc.operators.GetOperatorById(tenantContext(tenant), uuid.MustParse(resp.OperatorId))
	require.NoError(t, err)
	assert.Equal(t, "my-operator", op.Name)
	assert.Equal(t, sqlcv1.V1OperatorKindGRPC, op.Kind)

	require.Len(t, svc.workers.created, 1)
	created := svc.workers.created[0]
	assert.Equal(t, op.Name, created.Name)
	assert.Equal(t, svc.dispatcherId, created.DispatcherId)
	require.NotNil(t, created.OperatorId)
	assert.Equal(t, op.ID, *created.OperatorId)
	assert.Empty(t, created.Actions, "registration never registers actions")
	assert.Equal(t, map[string]int32{"default": 5}, created.SlotConfig)

	workerId := uuid.MustParse(resp.WorkerId)
	require.Len(t, svc.workers.labels[workerId], 1)
	assert.Equal(t, "kind", svc.workers.labels[workerId][0].Key)

	// a second registration of the same name reuses the operator and creates another worker
	again, err := svc.Register(tenantContext(tenant), &v1contracts.OperatorRegisterRequest{Name: "my-operator"})
	require.NoError(t, err)
	assert.Equal(t, resp.OperatorId, again.OperatorId)
	assert.NotEqual(t, resp.WorkerId, again.WorkerId)
	assert.False(t, again.Resumed)
	assert.Equal(t, map[string]int32{repository.SlotTypeDefault: defaultSlotCount}, svc.workers.created[1].SlotConfig, "empty slot config defaults")
}

func TestRegisterResumesOwnWorker(t *testing.T) {
	tenant := &sqlcv1.Tenant{ID: uuid.New()}
	svc := newTestService(t, nil)

	first, err := svc.Register(tenantContext(tenant), &v1contracts.OperatorRegisterRequest{Name: "my-operator"})
	require.NoError(t, err)

	resumed, err := svc.Register(tenantContext(tenant), &v1contracts.OperatorRegisterRequest{
		Name:     "my-operator",
		WorkerId: &first.WorkerId,
	})
	require.NoError(t, err)
	assert.True(t, resumed.Resumed)
	assert.Equal(t, first.WorkerId, resumed.WorkerId)
	assert.Equal(t, first.OperatorId, resumed.OperatorId)
	assert.Len(t, svc.workers.created, 1, "resume must not create a worker")
}

func TestRegisterDoesNotResumeAnotherOperatorsWorker(t *testing.T) {
	tenant := &sqlcv1.Tenant{ID: uuid.New()}
	svc := newTestService(t, nil)

	other, err := svc.Register(tenantContext(tenant), &v1contracts.OperatorRegisterRequest{Name: "other-operator"})
	require.NoError(t, err)

	unknown := uuid.NewString()

	cases := []struct {
		name     string
		workerId string
	}{
		{name: "another operator's worker", workerId: other.WorkerId},
		{name: "unknown worker", workerId: unknown},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			before := len(svc.workers.created)

			resp, err := svc.Register(tenantContext(tenant), &v1contracts.OperatorRegisterRequest{
				Name:     "my-operator",
				WorkerId: &tc.workerId,
			})
			require.NoError(t, err)
			assert.False(t, resp.Resumed)
			assert.NotEqual(t, tc.workerId, resp.WorkerId)
			assert.Len(t, svc.workers.created, before+1, "a worker that cannot be resumed is replaced by a new one")
		})
	}

	malformed := "not-a-uuid"
	_, err = svc.Register(tenantContext(tenant), &v1contracts.OperatorRegisterRequest{Name: "my-operator", WorkerId: &malformed})
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestRegisterRejects(t *testing.T) {
	tenant := &sqlcv1.Tenant{ID: uuid.New()}

	t.Run("missing tenant", func(t *testing.T) {
		svc := newTestService(t, nil)
		_, err := svc.Register(t.Context(), &v1contracts.OperatorRegisterRequest{Name: "op"})
		assert.Equal(t, codes.Unauthenticated, status.Code(err))
	})

	for _, name := range []string{"", "bad name!"} {
		t.Run("invalid name "+name, func(t *testing.T) {
			svc := newTestService(t, nil)
			_, err := svc.Register(tenantContext(tenant), &v1contracts.OperatorRegisterRequest{Name: name})
			assert.Equal(t, codes.InvalidArgument, status.Code(err))
			assert.Empty(t, svc.operators.operators, "validation runs before the upsert")
		})
	}
}
