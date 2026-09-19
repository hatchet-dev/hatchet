package grpcoperator

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	v1contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

type fakeDurableStream struct {
	grpc.ServerStream

	ctx  context.Context
	recv chan *v1contracts.DurableTaskRequest
}

func (f *fakeDurableStream) Context() context.Context { return f.ctx }

func (f *fakeDurableStream) Recv() (*v1contracts.DurableTaskRequest, error) {
	return <-f.recv, nil
}

func (f *fakeDurableStream) Send(*v1contracts.DurableTaskResponse) error { return nil }

func registerWorkerMsg(workerId string) *v1contracts.DurableTaskRequest {
	return &v1contracts.DurableTaskRequest{Message: &v1contracts.DurableTaskRequest_RegisterWorker{
		RegisterWorker: &v1contracts.DurableTaskRequestRegisterWorker{WorkerId: workerId},
	}}
}

func TestSendStepActionEventChecksWorkerOwnership(t *testing.T) {
	tenant := &sqlcv1.Tenant{ID: uuid.New()}
	svc := newTestService(t, nil)
	ctx, _, worker := registeredOperator(t, svc, tenant)

	otherOp := uuid.New()
	other := svc.workers.Add(&sqlcv1.Worker{ID: uuid.New(), TenantId: tenant.ID, OperatorId: &otherOp})

	_, err := svc.SendStepActionEvent(tenantContext(tenant), &contracts.StepActionEvent{WorkerId: worker.ID.String()})
	assert.Equal(t, codes.InvalidArgument, status.Code(err), "operator metadata is required")

	_, err = svc.SendStepActionEvent(ctx, &contracts.StepActionEvent{WorkerId: other.ID.String()})
	assert.Equal(t, codes.PermissionDenied, status.Code(err))

	resp, err := svc.SendStepActionEvent(ctx, &contracts.StepActionEvent{WorkerId: worker.ID.String()})
	require.NoError(t, err)
	assert.Equal(t, worker.ID.String(), resp.WorkerId)
	assert.Len(t, svc.dispatcher.StepCalls(), 1)
}

func TestDurableTaskChecksWorkerOwnership(t *testing.T) {
	tenant := &sqlcv1.Tenant{ID: uuid.New()}

	t.Run("operator metadata required", func(t *testing.T) {
		svc := newTestService(t, nil)
		stream := &fakeDurableStream{ctx: tenantContext(tenant), recv: make(chan *v1contracts.DurableTaskRequest, 1)}

		err := svc.DurableTask(stream)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		assert.Nil(t, svc.dispatcher.DurableRegister(), "the dispatcher never sees an unauthorized stream")
	})

	t.Run("another operator's worker is rejected", func(t *testing.T) {
		svc := newTestService(t, nil)
		ctx, _, _ := registeredOperator(t, svc, tenant)
		otherOp := uuid.New()
		other := svc.workers.Add(&sqlcv1.Worker{ID: uuid.New(), TenantId: tenant.ID, OperatorId: &otherOp})

		stream := &fakeDurableStream{ctx: ctx, recv: make(chan *v1contracts.DurableTaskRequest, 1)}
		stream.recv <- registerWorkerMsg(other.ID.String())

		err := svc.DurableTask(stream)
		assert.Equal(t, codes.PermissionDenied, status.Code(err), err)
		assert.Nil(t, svc.dispatcher.DurableRegister(), "the register message is not delegated when ownership fails")
	})

	t.Run("first message must register", func(t *testing.T) {
		svc := newTestService(t, nil)
		ctx, _, _ := registeredOperator(t, svc, tenant)

		stream := &fakeDurableStream{ctx: ctx, recv: make(chan *v1contracts.DurableTaskRequest, 1)}
		stream.recv <- &v1contracts.DurableTaskRequest{Message: &v1contracts.DurableTaskRequest_Memo{Memo: &v1contracts.DurableTaskMemoRequest{}}}

		err := svc.DurableTask(stream)
		assert.Equal(t, codes.InvalidArgument, status.Code(err), err)
	})

	t.Run("own worker is delegated", func(t *testing.T) {
		svc := newTestService(t, nil)
		ctx, _, worker := registeredOperator(t, svc, tenant)

		stream := &fakeDurableStream{ctx: ctx, recv: make(chan *v1contracts.DurableTaskRequest, 1)}
		stream.recv <- registerWorkerMsg(worker.ID.String())

		require.NoError(t, svc.DurableTask(stream))
		require.NotNil(t, svc.dispatcher.DurableRegister())
		assert.Equal(t, worker.ID.String(), svc.dispatcher.DurableRegister().GetRegisterWorker().WorkerId)
	})
}
