package grpcoperator

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"connectrpc.com/connect"

	"github.com/hatchet-dev/hatchet/internal/services/operatorsvc"
	v1contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/internal/services/shared/rpcstream"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// startTimeout bounds how long a Listen stream may sit idle before its start message.
const startTimeout = 30 * time.Second

type recvResult struct {
	req *v1contracts.OperatorListenRequest
	err error
}

// Listen activates a registered worker for the lifetime of the stream and fans assigned actions
// out to it. One guarded sender owns the send side of the stream: actions go out through the
// dispatcher's fan-out and acks go out through the session handle, and both write through it.
// This goroutine consumes heartbeats, action deltas and pauses.
func (s *OperatorServiceImpl) Listen(ctx context.Context, stream *connect.BidiStream[v1contracts.OperatorListenRequest, v1contracts.OperatorListenResponse]) error {
	tenant, op, err := s.authorizeOperator(ctx)

	if err != nil {
		return err
	}

	return s.listen(ctx, stream, tenant, op)
}

// listenStream is the transport the Listen handler runs over, satisfied by the connect stream.
type listenStream interface {
	Receive() (*v1contracts.OperatorListenRequest, error)
	Send(*v1contracts.OperatorListenResponse) error
}

// listen runs the Listen stream for an authorized operator.
func (s *OperatorServiceImpl) listen(ctx context.Context, stream listenStream, tenant *sqlcv1.Tenant, op *sqlcv1.V1Operator) (err error) {
	// the dispatcher sends on this stream from other goroutines; Close runs after the session
	// is closed and before the handler returns, so no send can reach the stream after that
	sender := rpcstream.NewSender[v1contracts.OperatorListenResponse](ctx, stream)
	defer sender.Close()

	// Receive runs in its own goroutine because it is only interrupted by the stream ending,
	// not by ctx; this lets the handler observe the dispatcher's fin signal and context
	// cancellation while a Receive is pending. Once the handler returns, the request context
	// is cancelled and the body closed, which unblocks the pending channel send (or the next
	// Receive) and lets the goroutine exit.
	msgCh := make(chan recvResult)

	go func() {
		for {
			req, err := stream.Receive()

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
		return connect.NewError(connect.CodeDeadlineExceeded, errors.New("timed out waiting for the start message"))
	case <-ctx.Done():
		return nil
	}

	start := first.GetStart()

	if start == nil {
		return connect.NewError(connect.CodeInvalidArgument, errors.New("the first message on the Listen stream must be start"))
	}

	worker, err := s.authorizeWorker(ctx, tenant, op, start.WorkerId)

	if err != nil {
		return err
	}

	session, err := s.svc.OpenSession(ctx, tenant, op, worker.ID, operatorsvc.OpenOpts{
		Stream: sender,
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

	// The session is closed without a pause: a gRPC operator pauses its own worker on the
	// stream before it hangs up, and a stream that drops must leave the worker assignable so
	// the operator's next connection resumes a worker the scheduler can use.
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
						return connect.NewError(connect.CodeUnavailable, fmt.Errorf("could not acknowledge actions delta %d: %w", seq, err))
					}
				}
			case *v1contracts.OperatorListenRequest_Pause:
				// the pause is committed, and the session stops delivering, before the ack
				// goes out: the ack is the client's promise that nothing more arrives
				if err := session.Pause(ctx, msg.Pause.IsPaused); err != nil {
					return err
				}

				if err := session.Send(ctx, &v1contracts.OperatorListenResponse{
					Message: &v1contracts.OperatorListenResponse_PauseAck{PauseAck: &v1contracts.OperatorPauseAck{IsPaused: msg.Pause.IsPaused}},
				}); err != nil {
					l.Error().Ctx(ctx).Err(err).Bool("paused", msg.Pause.IsPaused).Msg("could not acknowledge operator pause")
					return connect.NewError(connect.CodeUnavailable, fmt.Errorf("could not acknowledge pause: %w", err))
				}

				l.Info().Ctx(ctx).Bool("paused", msg.Pause.IsPaused).Msg("operator worker pause state changed")
			case *v1contracts.OperatorListenRequest_Start:
				return connect.NewError(connect.CodeInvalidArgument, errors.New("the Listen stream is already started"))
			default:
				return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("unexpected message on the Listen stream: %T", msg))
			}
		}
	}
}
