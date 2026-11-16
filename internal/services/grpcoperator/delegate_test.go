package grpcoperator

import (
	"testing"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	v1contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// fakeDurableStream feeds client messages to the run half of the DurableTask handler.
type fakeDurableStream struct {
	recv chan *v1contracts.DurableTaskRequest
}

func (f *fakeDurableStream) Receive() (*v1contracts.DurableTaskRequest, error) {
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
	_, op, worker := registeredOperator(t, svc, tenant)

	otherOp := uuid.New()
	other := svc.workers.Add(&sqlcv1.Worker{ID: uuid.New(), TenantId: tenant.ID, OperatorId: &otherOp})

	// the operator header is read by the real handler, so that case goes over the wire
	client := operatorClient(t, svc, tenant)

	_, err := client.SendStepActionEvent(t.Context(), &contracts.StepActionEvent{WorkerId: worker.ID.String()})
	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err), "operator metadata is required")

	_, err = client.SendStepActionEvent(operatorCallContext(t.Context(), op.ID.String()), &contracts.StepActionEvent{WorkerId: other.ID.String()})
	assert.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))

	resp, err := client.SendStepActionEvent(operatorCallContext(t.Context(), op.ID.String()), &contracts.StepActionEvent{WorkerId: worker.ID.String()})
	require.NoError(t, err)
	assert.Equal(t, worker.ID.String(), resp.WorkerId)
	assert.Len(t, svc.dispatcher.StepCalls(), 1)
}

func TestDurableTaskChecksWorkerOwnership(t *testing.T) {
	tenant := &sqlcv1.Tenant{ID: uuid.New()}

	// the operator header is read by the real handler, so this case goes over the wire
	t.Run("operator metadata required", func(t *testing.T) {
		svc := newTestService(t, nil)
		client := operatorClient(t, svc, tenant)

		stream, err := client.DurableTask(t.Context())
		require.NoError(t, err)

		// the handler refuses the stream before reading it, so the send may already see it
		// closed; the refusal itself comes back on Receive
		_ = stream.Send(registerWorkerMsg(uuid.NewString()))

		_, err = stream.Receive()
		assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err), err)
		assert.Nil(t, svc.recorder.Register(), "the dispatcher never sees an unauthorized stream")
	})

	t.Run("another operator's worker is rejected", func(t *testing.T) {
		svc := newTestService(t, nil)
		ctx, op, _ := registeredOperator(t, svc, tenant)
		otherOp := uuid.New()
		other := svc.workers.Add(&sqlcv1.Worker{ID: uuid.New(), TenantId: tenant.ID, OperatorId: &otherOp})

		stream := &fakeDurableStream{recv: make(chan *v1contracts.DurableTaskRequest, 1)}
		stream.recv <- registerWorkerMsg(other.ID.String())

		err := svc.durableTask(ctx, stream, tenant, op)
		assert.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err), err)
		assert.Nil(t, svc.recorder.Register(), "the register message is not delegated when ownership fails")
	})

	t.Run("first message must register", func(t *testing.T) {
		svc := newTestService(t, nil)
		ctx, op, _ := registeredOperator(t, svc, tenant)

		stream := &fakeDurableStream{recv: make(chan *v1contracts.DurableTaskRequest, 1)}
		stream.recv <- &v1contracts.DurableTaskRequest{Message: &v1contracts.DurableTaskRequest_Memo{Memo: &v1contracts.DurableTaskMemoRequest{}}}

		err := svc.durableTask(ctx, stream, tenant, op)
		assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err), err)
	})

	t.Run("own worker is delegated", func(t *testing.T) {
		svc := newTestService(t, nil)
		ctx, op, worker := registeredOperator(t, svc, tenant)

		stream := &fakeDurableStream{recv: make(chan *v1contracts.DurableTaskRequest, 1)}
		stream.recv <- registerWorkerMsg(worker.ID.String())

		require.NoError(t, svc.durableTask(ctx, stream, tenant, op))
		require.NotNil(t, svc.recorder.Register())
		assert.Equal(t, worker.ID.String(), svc.recorder.Register().GetRegisterWorker().WorkerId)
	})
}
