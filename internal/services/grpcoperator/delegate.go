package grpcoperator

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	v1contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/internal/services/shared/rpcstream"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// SendStepActionEvent reports task progress for an action delivered on a Listen stream. The
// event's worker must belong to the calling operator; the dispatcher then handles it exactly
// like an SDK worker's event.
func (s *OperatorServiceImpl) SendStepActionEvent(ctx context.Context, req *contracts.StepActionEvent) (*contracts.ActionEventResponse, error) {
	tenant, op, err := s.authorizeOperator(ctx)

	if err != nil {
		return nil, err
	}

	return s.svc.SendStepActionEvent(ctx, tenant, op, req)
}

// DurableTask is the durable task event stream. The operator is authorized once when the
// stream opens, and the worker named by the first register_worker message is checked for
// ownership before the dispatcher sees it; the dispatcher then owns the session. The SDK's
// durable listener registers exactly once per stream (it opens a new stream to register
// again), so a second register_worker is refused as a protocol error rather than re-checked:
// the stream stays bound to the worker it was authorized for.
func (s *OperatorServiceImpl) DurableTask(ctx context.Context, stream *connect.BidiStream[v1contracts.DurableTaskRequest, v1contracts.DurableTaskResponse]) error {
	tenant, op, err := s.authorizeOperator(ctx)

	if err != nil {
		return err
	}

	return s.durableTask(ctx, stream, tenant, op)
}

// durableStream is the transport the DurableTask handler runs over, satisfied by the connect
// stream.
type durableStream interface {
	Receive() (*v1contracts.DurableTaskRequest, error)
	Send(*v1contracts.DurableTaskResponse) error
}

// durableTask runs the DurableTask stream for an authorized operator.
func (s *OperatorServiceImpl) durableTask(ctx context.Context, stream durableStream, tenant *sqlcv1.Tenant, op *sqlcv1.V1Operator) error {
	// other goroutines send on this stream; Close runs after the dispatcher has torn the
	// session down and before the handler returns
	sender := rpcstream.NewSender[v1contracts.DurableTaskResponse](ctx, stream)
	defer sender.Close()

	registered := false

	receive := func() (*v1contracts.DurableTaskRequest, error) {
		req, err := stream.Receive()

		if err != nil {
			return nil, err
		}

		register := req.GetRegisterWorker()

		if registered {
			if register != nil {
				return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("the DurableTask stream is already registered to a worker"))
			}

			return req, nil
		}

		if register == nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("the first message on the DurableTask stream must be register_worker"))
		}

		if _, err := s.authorizeWorker(ctx, tenant, op, register.WorkerId); err != nil {
			return nil, err
		}

		registered = true

		return req, nil
	}

	return s.durable.DurableTaskWithReceive(ctx, receive, sender)
}
