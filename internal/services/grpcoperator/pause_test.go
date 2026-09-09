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

// Pausing is a call of its own so an operator that is draining knows the pause is committed
// before it stops answering, and so it works while the Listen stream is being torn down.
func TestPauseWorker(t *testing.T) {
	tenant := &sqlcv1.Tenant{ID: uuid.New()}
	svc := newTestService(t, nil)
	ctx, _, worker := registeredOperator(t, svc, tenant)

	resp, err := svc.PauseWorker(ctx, &v1contracts.OperatorPauseWorkerRequest{WorkerId: worker.ID.String(), Paused: true})
	require.NoError(t, err)
	assert.Equal(t, worker.ID.String(), resp.WorkerId)
	assert.True(t, resp.Paused)
	assert.True(t, svc.workers.IsPaused(worker.ID))

	resp, err = svc.PauseWorker(ctx, &v1contracts.OperatorPauseWorkerRequest{WorkerId: worker.ID.String()})
	require.NoError(t, err)
	assert.False(t, resp.Paused)
	assert.False(t, svc.workers.IsPaused(worker.ID), "the same call unpauses")
}

func TestPauseWorkerChecksOwnership(t *testing.T) {
	tenant := &sqlcv1.Tenant{ID: uuid.New()}
	svc := newTestService(t, nil)
	ctx, _, worker := registeredOperator(t, svc, tenant)

	otherOp := uuid.New()
	other := svc.workers.Add(&sqlcv1.Worker{ID: uuid.New(), TenantId: tenant.ID, OperatorId: &otherOp})

	_, err := svc.PauseWorker(tenantContext(tenant), &v1contracts.OperatorPauseWorkerRequest{WorkerId: worker.ID.String(), Paused: true})
	assert.Equal(t, codes.InvalidArgument, status.Code(err), "operator metadata is required")

	_, err = svc.PauseWorker(ctx, &v1contracts.OperatorPauseWorkerRequest{WorkerId: other.ID.String(), Paused: true})
	assert.Equal(t, codes.PermissionDenied, status.Code(err))

	_, err = svc.PauseWorker(ctx, &v1contracts.OperatorPauseWorkerRequest{WorkerId: "not-a-uuid", Paused: true})
	assert.Equal(t, codes.InvalidArgument, status.Code(err))

	assert.False(t, svc.workers.IsPaused(other.ID), "a refused call changes nothing")
}
