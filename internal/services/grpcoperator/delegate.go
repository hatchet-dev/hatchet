package grpcoperator

import (
	"context"
	"sync"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	v1contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
)

// SendStepActionEvent reports task progress for an action delivered on a Listen stream. The
// event's worker must belong to the calling operator; the dispatcher then handles it exactly
// like an SDK worker's event.
func (s *OperatorServiceImpl) SendStepActionEvent(ctx context.Context, req *contracts.StepActionEvent) (*contracts.ActionEventResponse, error) {
	op, err := s.authorizeOperator(ctx)

	if err != nil {
		return nil, err
	}

	if _, err := s.authorizeOperatorWorker(ctx, op, req.WorkerId); err != nil {
		return nil, err
	}

	return s.dispatcher.SendStepActionEvent(ctx, req)
}

// DurableTask is the durable task event stream. The operator is authorized once when the
// stream opens, and the worker named by the first register_worker message is checked for
// ownership before the dispatcher's V1 service sees it; the dispatcher then owns the stream.
// The SDK's durable listener registers exactly once per stream (it opens a new stream to
// register again), so a second register_worker is refused as a protocol error rather than
// re-checked: the stream stays bound to the worker it was authorized for.
func (s *OperatorServiceImpl) DurableTask(stream v1contracts.OperatorService_DurableTaskServer) error {
	op, err := s.authorizeOperator(stream.Context())

	if err != nil {
		return err
	}

	return s.dispatcher.DurableTask(&ownershipCheckedDurableStream{
		OperatorService_DurableTaskServer: stream,
		check: func(ctx context.Context, workerId string) error {
			_, err := s.authorizeOperatorWorker(ctx, op, workerId)
			return err
		},
	})
}

// ownershipCheckedDurableStream intercepts the first register_worker message on a durable task
// stream and runs the worker ownership check on its worker id before handing the message to the
// dispatcher. Any later register_worker is refused with InvalidArgument. The generated
// OperatorService_DurableTaskServer and V1Dispatcher_DurableTaskServer interfaces have the same
// method set, so the wrapped stream passes through otherwise unchanged.
type ownershipCheckedDurableStream struct {
	v1contracts.OperatorService_DurableTaskServer

	check func(ctx context.Context, workerId string) error

	mu      sync.Mutex
	checked bool
}

func (w *ownershipCheckedDurableStream) Recv() (*v1contracts.DurableTaskRequest, error) {
	req, err := w.OperatorService_DurableTaskServer.Recv()

	if err != nil {
		return nil, err
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	register := req.GetRegisterWorker()

	if w.checked {
		if register != nil {
			return nil, status.Error(codes.InvalidArgument, "the DurableTask stream is already registered to a worker")
		}

		return req, nil
	}

	if register == nil {
		return nil, status.Error(codes.InvalidArgument, "the first message on the DurableTask stream must be register_worker")
	}

	if err := w.check(w.Context(), register.WorkerId); err != nil {
		return nil, err
	}

	w.checked = true

	return req, nil
}
