package grpcoperator

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	"github.com/hatchet-dev/hatchet/internal/services/operatorsvc/operatorsvctest"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// The operator id travels in the request header; parsing it is this package's job, and deciding
// whether the operator may be used is the operator service's. The header is only visible through
// connect's handler context, so these cases go over the wire and use SendStepActionEvent, the
// unary RPC that authorizes the operator first.
func TestAuthorizeOperatorReadsMetadata(t *testing.T) {
	tenant := &sqlcv1.Tenant{ID: uuid.New()}
	grpcOp := &sqlcv1.V1Operator{ID: uuid.New(), TenantID: tenant.ID, Name: "grpc-op", Kind: sqlcv1.V1OperatorKindGRPC, LeasingManager: sqlcv1.V1OperatorLeasingManagerSELF}

	withHeader := func(key, value string) context.Context {
		ctx, info := connect.NewClientContext(context.Background())
		info.RequestHeader().Set(key, value)

		return ctx
	}

	cases := []struct {
		ctx      context.Context
		tenant   *sqlcv1.Tenant
		name     string
		wantCode connect.Code // zero means the call succeeds
	}{
		{
			name:     "missing tenant",
			tenant:   nil,
			ctx:      operatorCallContext(context.Background(), grpcOp.ID.String()),
			wantCode: connect.CodeUnauthenticated,
		},
		{
			name:     "no incoming metadata",
			tenant:   tenant,
			ctx:      context.Background(),
			wantCode: connect.CodeInvalidArgument,
		},
		{
			name:     "metadata without operator id",
			tenant:   tenant,
			ctx:      withHeader("other-key", "value"),
			wantCode: connect.CodeInvalidArgument,
		},
		{
			name:     "malformed operator id",
			tenant:   tenant,
			ctx:      operatorCallContext(context.Background(), "not-a-uuid"),
			wantCode: connect.CodeInvalidArgument,
		},
		{
			name:     "unknown operator",
			tenant:   tenant,
			ctx:      operatorCallContext(context.Background(), uuid.NewString()),
			wantCode: connect.CodePermissionDenied,
		},
		{
			name:     "authorized",
			tenant:   tenant,
			ctx:      operatorCallContext(context.Background(), grpcOp.ID.String()),
			wantCode: 0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := newTestService(t, operatorsvctest.NewOperatorStore(grpcOp))
			worker := svc.workers.Add(&sqlcv1.Worker{ID: uuid.New(), TenantId: tenant.ID, OperatorId: &grpcOp.ID})
			client := operatorClient(t, svc, tc.tenant)

			resp, err := client.SendStepActionEvent(tc.ctx, &contracts.StepActionEvent{WorkerId: worker.ID.String()})

			if tc.wantCode == 0 {
				// the worker is owned by grpcOp under tenant, so the event only goes through
				// when authorization resolved that operator and tenant
				require.NoError(t, err)
				assert.Equal(t, worker.ID.String(), resp.WorkerId)
				assert.Len(t, svc.dispatcher.StepCalls(), 1)

				return
			}

			require.Error(t, err)
			assert.Nil(t, resp)
			assert.Equal(t, tc.wantCode, connect.CodeOf(err), err.Error())
			assert.Empty(t, svc.dispatcher.StepCalls(), "nothing reaches the dispatcher without authorization")
		})
	}
}

// A worker id that is not a uuid is a protocol error, not an authorization failure.
func TestAuthorizeWorkerRejectsMalformedId(t *testing.T) {
	tenant := &sqlcv1.Tenant{ID: uuid.New()}
	svc := newTestService(t, nil)
	_, op, worker := registeredOperator(t, svc, tenant)

	_, err := svc.authorizeWorker(t.Context(), tenant, op, "not-a-uuid")
	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))

	got, err := svc.authorizeWorker(t.Context(), tenant, op, worker.ID.String())
	require.NoError(t, err)
	assert.Equal(t, worker.ID, got.ID)
}
