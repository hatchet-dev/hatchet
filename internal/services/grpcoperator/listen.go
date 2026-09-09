package grpcoperator

import (
	"errors"
	"io"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	"github.com/hatchet-dev/hatchet/internal/services/operatorsvc"
	v1contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
)

// startTimeout bounds how long a Listen stream may sit idle before its start message.
const startTimeout = 30 * time.Second

type recvResult struct {
	req *v1contracts.OperatorListenRequest
	err error
}

// wrapAssignedAction converts a dispatcher action into the Listen stream's server message.
func wrapAssignedAction(action *contracts.AssignedAction) proto.Message {
	return &v1contracts.OperatorListenResponse{
		Message: &v1contracts.OperatorListenResponse_Action{Action: action},
	}
}

// Listen activates a registered worker for the lifetime of the stream and fans assigned actions
// out to it. The dispatcher session owns the send side of the stream: actions go out through
// its fan-out and delta acks go out through the session handle, so the two never overlap. This
// goroutine consumes heartbeats and action deltas.
func (s *OperatorServiceImpl) Listen(stream v1contracts.OperatorService_ListenServer) (err error) {
	ctx := stream.Context()

	tenant, op, err := s.authorizeOperator(ctx)

	if err != nil {
		return err
	}

	// Recv runs in its own goroutine because it is only interrupted by the stream ending, not
	// by ctx; this lets the handler observe the dispatcher's fin signal and context
	// cancellation while a Recv is pending. Once the handler returns, gRPC cancels ctx, which
	// unblocks the pending channel send (or the next Recv) and lets the goroutine exit.
	msgCh := make(chan recvResult)

	go func() {
		for {
			req, err := stream.Recv()

			select {
			case msgCh <- recvResult{req: req, err: err}:
			case <-ctx.Done():
				return
			}

			if err != nil {
				return
			}
		}
	}()

	var first *v1contracts.OperatorListenRequest

	select {
	case res := <-msgCh:
		if res.err != nil {
			if errors.Is(res.err, io.EOF) {
				return nil
			}

			return res.err
		}

		first = res.req
	case <-time.After(startTimeout):
		return status.Error(codes.DeadlineExceeded, "timed out waiting for the start message")
	case <-ctx.Done():
		return nil
	}

	start := first.GetStart()

	if start == nil {
		return status.Error(codes.InvalidArgument, "the first message on the Listen stream must be start")
	}

	worker, err := s.authorizeWorker(ctx, tenant, op, start.WorkerId)

	if err != nil {
		return err
	}

	session, err := s.svc.OpenSession(ctx, tenant, op, worker.ID, operatorsvc.OpenOpts{
		Stream: stream,
		Wrap:   wrapAssignedAction,
		Worker: worker,
	})

	if err != nil {
		return err
	}

	l := s.l.With().
		Str("tenant_id", tenant.ID.String()).
		Str("operator_name", op.Name).
		Str("operator_id", op.ID.String()).
		Str("worker_id", worker.ID.String()).
		Logger()

	// The session is closed without a pause: a gRPC operator pauses its own worker through
	// PauseWorker before it hangs up, and a stream that drops must leave the worker assignable
	// so the operator's next connection resumes a worker the scheduler can use.
	defer func() {
		closeErr := session.Close(ctx, operatorsvc.WithoutPause())

		if closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	fin := session.Fin()

	for {
		select {
		case <-fin:
			l.Debug().Ctx(ctx).Msg("dispatcher closed the operator stream")
			return nil
		case <-ctx.Done():
			l.Debug().Ctx(ctx).Msg("operator worker disconnected")
			return nil
		case res := <-msgCh:
			if res.err != nil {
				if errors.Is(res.err, io.EOF) {
					l.Debug().Ctx(ctx).Msg("operator worker closed the stream")
					return nil
				}

				return res.err
			}

			switch msg := res.req.Message.(type) {
			case *v1contracts.OperatorListenRequest_Heartbeat:
				if err := session.Heartbeat(ctx, time.Now().UTC()); err != nil {
					l.Error().Ctx(ctx).Err(err).Msg("could not update worker heartbeat")
				}
			case *v1contracts.OperatorListenRequest_Actions:
				if _, err := session.ApplyDelta(ctx, msg.Actions.Add, msg.Actions.Remove); err != nil {
					return err
				}

				// the ack is the client's signal that the delta is committed; a client that never
				// receives it resends the delta after its next reconnect, so an ack that cannot be
				// written ends the stream rather than leaving the delta unconfirmed
				if seq := msg.Actions.Sequence; seq != 0 {
					if err := session.Send(ctx, &v1contracts.OperatorListenResponse{
						Message: &v1contracts.OperatorListenResponse_Ack{Ack: &v1contracts.OperatorActionsAck{Sequence: seq}},
					}); err != nil {
						l.Error().Ctx(ctx).Err(err).Uint64("sequence", seq).Msg("could not acknowledge operator actions delta")
						return status.Errorf(codes.Unavailable, "could not acknowledge actions delta %d: %s", seq, err.Error())
					}
				}
			case *v1contracts.OperatorListenRequest_Start:
				return status.Error(codes.InvalidArgument, "the Listen stream is already started")
			default:
				return status.Errorf(codes.InvalidArgument, "unexpected message on the Listen stream: %T", msg)
			}
		}
	}
}
