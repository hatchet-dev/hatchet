package grpcoperator

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/hatchet-dev/hatchet/internal/services/operatorsvc/operatorsvctest"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// The operator id travels in gRPC metadata; parsing it is this package's job, and deciding
// whether the operator may be used is the operator service's.
func TestAuthorizeOperatorReadsMetadata(t *testing.T) {
	tenant := &sqlcv1.Tenant{ID: uuid.New()}
	grpcOp := &sqlcv1.V1Operator{ID: uuid.New(), TenantID: tenant.ID, Name: "grpc-op", Kind: sqlcv1.V1OperatorKindGRPC}

	cases := []struct {
		ctx      context.Context
		name     string
		wantCode codes.Code
	}{
		{
			name:     "missing tenant",
			ctx:      metadata.NewIncomingContext(context.Background(), metadata.Pairs(OperatorIdMetadataKey, grpcOp.ID.String())),
			wantCode: codes.Unauthenticated,
		},
		{
			name:     "no incoming metadata",
			ctx:      tenantContext(tenant),
			wantCode: codes.InvalidArgument,
		},
		{
			name:     "metadata without operator id",
			ctx:      metadata.NewIncomingContext(tenantContext(tenant), metadata.Pairs("other-key", "value")),
			wantCode: codes.InvalidArgument,
		},
		{
			name:     "malformed operator id",
			ctx:      operatorContext(tenant, "not-a-uuid"),
			wantCode: codes.InvalidArgument,
		},
		{
			name:     "unknown operator",
			ctx:      operatorContext(tenant, uuid.NewString()),
			wantCode: codes.PermissionDenied,
		},
		{
			name:     "authorized",
			ctx:      operatorContext(tenant, grpcOp.ID.String()),
			wantCode: codes.OK,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := newTestService(t, operatorsvctest.NewOperatorStore(grpcOp))

			gotTenant, op, err := svc.authorizeOperator(tc.ctx)

			if tc.wantCode == codes.OK {
				require.NoError(t, err)
				assert.Equal(t, grpcOp.ID, op.ID)
				assert.Equal(t, tenant.ID, gotTenant.ID)

				return
			}

			require.Error(t, err)
			assert.Nil(t, op)
			assert.Equal(t, tc.wantCode, status.Code(err), err.Error())
		})
	}
}

// A worker id that is not a uuid is a protocol error, not an authorization failure.
func TestAuthorizeWorkerRejectsMalformedId(t *testing.T) {
	tenant := &sqlcv1.Tenant{ID: uuid.New()}
	svc := newTestService(t, nil)
	_, op, worker := registeredOperator(t, svc, tenant)

	_, err := svc.authorizeWorker(t.Context(), tenant, op, "not-a-uuid")
	assert.Equal(t, codes.InvalidArgument, status.Code(err))

	got, err := svc.authorizeWorker(t.Context(), tenant, op, worker.ID.String())
	require.NoError(t, err)
	assert.Equal(t, worker.ID, got.ID)
}
