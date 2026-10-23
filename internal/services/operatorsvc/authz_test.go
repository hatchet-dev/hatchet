package operatorsvc_test

import (
	"testing"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/internal/services/operatorsvc/operatorsvctest"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func TestAuthorizeOperator(t *testing.T) {
	tenant := &sqlcv1.Tenant{ID: uuid.New()}
	otherTenant := &sqlcv1.Tenant{ID: uuid.New()}

	grpcOp := &sqlcv1.V1Operator{ID: uuid.New(), TenantID: tenant.ID, Name: "grpc-op", Kind: sqlcv1.V1OperatorKindGRPC, LeasingManager: sqlcv1.V1OperatorLeasingManagerSELF}
	dagOp := &sqlcv1.V1Operator{ID: uuid.New(), TenantID: tenant.ID, Name: "dag-op", Kind: sqlcv1.V1OperatorKindDAG, LeasingManager: sqlcv1.V1OperatorLeasingManagerDISPATCHER}
	// a contract operator a dispatcher claims is driven by the claimer, never over the wire
	dispatcherOp := &sqlcv1.V1Operator{ID: uuid.New(), TenantID: tenant.ID, Name: "dispatcher-op", Kind: sqlcv1.V1OperatorKindGRPC, LeasingManager: sqlcv1.V1OperatorLeasingManagerDISPATCHER}

	cases := []struct {
		tenant     *sqlcv1.Tenant
		name       string
		operatorId uuid.UUID
		wantCode   connect.Code
	}{
		{name: "missing tenant", tenant: nil, operatorId: grpcOp.ID, wantCode: connect.CodeUnauthenticated},
		{name: "unknown operator", tenant: tenant, operatorId: uuid.New(), wantCode: connect.CodePermissionDenied},
		{name: "tenant mismatch", tenant: otherTenant, operatorId: grpcOp.ID, wantCode: connect.CodePermissionDenied},
		{name: "wrong kind", tenant: tenant, operatorId: dagOp.ID, wantCode: connect.CodePermissionDenied},
		{name: "wrong leasing manager", tenant: tenant, operatorId: dispatcherOp.ID, wantCode: connect.CodePermissionDenied},
		{name: "authorized", tenant: tenant, operatorId: grpcOp.ID, wantCode: 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := newTestService(t, operatorsvctest.NewOperatorStore(grpcOp, dagOp, dispatcherOp))

			op, err := svc.AuthorizeOperator(t.Context(), tc.tenant, tc.operatorId)

			if tc.wantCode == 0 {
				require.NoError(t, err)
				assert.Equal(t, grpcOp.ID, op.ID)

				return
			}

			require.Error(t, err)
			assert.Nil(t, op)
			assert.Equal(t, tc.wantCode, connect.CodeOf(err), err.Error())
		})
	}
}

func TestAuthorizeOperatorCachesLookups(t *testing.T) {
	tenant := &sqlcv1.Tenant{ID: uuid.New()}
	grpcOp := &sqlcv1.V1Operator{ID: uuid.New(), TenantID: tenant.ID, Name: "grpc-op", Kind: sqlcv1.V1OperatorKindGRPC, LeasingManager: sqlcv1.V1OperatorLeasingManagerSELF}

	store := operatorsvctest.NewOperatorStore(grpcOp)
	svc := newTestService(t, store)

	for i := 0; i < 3; i++ {
		op, err := svc.AuthorizeOperator(t.Context(), tenant, grpcOp.ID)
		require.NoError(t, err)
		assert.Equal(t, grpcOp.ID, op.ID)
	}

	assert.Equal(t, int64(1), store.GetCalls(), "cache hit should avoid a second repository call")
}

func TestAuthorizeOperatorDoesNotCacheMisses(t *testing.T) {
	tenant := &sqlcv1.Tenant{ID: uuid.New()}
	grpcOp := &sqlcv1.V1Operator{ID: uuid.New(), TenantID: tenant.ID, Name: "grpc-op", Kind: sqlcv1.V1OperatorKindGRPC, LeasingManager: sqlcv1.V1OperatorLeasingManagerSELF}

	store := operatorsvctest.NewOperatorStore()
	svc := newTestService(t, store)

	_, err := svc.AuthorizeOperator(t.Context(), tenant, grpcOp.ID)
	require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))

	// the operator is created after the first call; the miss must not be sticky
	store.Put(grpcOp)

	op, err := svc.AuthorizeOperator(t.Context(), tenant, grpcOp.ID)
	require.NoError(t, err)
	assert.Equal(t, grpcOp.ID, op.ID)
	assert.Equal(t, int64(2), store.GetCalls())
}

func TestAuthorizeWorker(t *testing.T) {
	tenant := &sqlcv1.Tenant{ID: uuid.New()}
	op := &sqlcv1.V1Operator{ID: uuid.New(), TenantID: tenant.ID, Name: "grpc-op", Kind: sqlcv1.V1OperatorKindGRPC, LeasingManager: sqlcv1.V1OperatorLeasingManagerSELF}
	otherOp := &sqlcv1.V1Operator{ID: uuid.New(), TenantID: tenant.ID, Name: "other-op", Kind: sqlcv1.V1OperatorKindGRPC, LeasingManager: sqlcv1.V1OperatorLeasingManagerSELF}

	svc := newTestService(t, nil)
	ownWorker := svc.workers.Add(&sqlcv1.Worker{ID: uuid.New(), TenantId: tenant.ID, OperatorId: &op.ID})
	otherWorker := svc.workers.Add(&sqlcv1.Worker{ID: uuid.New(), TenantId: tenant.ID, OperatorId: &otherOp.ID})
	sdkWorker := svc.workers.Add(&sqlcv1.Worker{ID: uuid.New(), TenantId: tenant.ID})

	_, err := svc.AuthorizeWorker(t.Context(), nil, op.ID, ownWorker.ID)
	assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))

	_, err = svc.AuthorizeWorker(t.Context(), tenant, op.ID, otherWorker.ID)
	assert.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))

	_, err = svc.AuthorizeWorker(t.Context(), tenant, op.ID, sdkWorker.ID)
	assert.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err), "a worker without an operator is not owned by anyone")

	_, err = svc.AuthorizeWorker(t.Context(), tenant, op.ID, uuid.New())
	assert.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))

	w, err := svc.AuthorizeWorker(t.Context(), tenant, op.ID, ownWorker.ID)
	require.NoError(t, err)
	assert.Equal(t, ownWorker.ID, w.ID)
}
