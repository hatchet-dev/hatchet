package grpcoperator

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	v1contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/analytics"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

const (
	// startTimeout bounds how long a Listen stream may sit idle before its start message.
	startTimeout = 30 * time.Second

	// heartbeatWriteInterval throttles heartbeat writes to the database; clients heartbeat
	// every 4 seconds, so this only matters for misbehaving clients.
	heartbeatWriteInterval = time.Second

	// defaultNotifyInterval throttles scheduler notifications while action deltas keep
	// arriving: the scheduler reloads the worker's action set on each notify, so a burst of
	// deltas is folded into one reload per second.
	defaultNotifyInterval = time.Second

	// deactivateTimeout bounds the detached deactivation write once the client is gone.
	deactivateTimeout = 20 * time.Second

	// MaxActionsPerDelta caps the ids (adds plus removes) one OperatorActionsDelta may carry.
	MaxActionsPerDelta = 1000
)

// actionsDeltaOpts carries the validation rules for an actions delta.
type actionsDeltaOpts struct {
	Add    []string `validate:"dive,actionId"`
	Remove []string `validate:"dive,actionId"`
}

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

	op, err := s.authorizeOperator(ctx)

	if err != nil {
		return err
	}

	// the tenant is present: authorizeOperator already required it
	tenant, _ := tenantFromContext(ctx)

	s.analytics.Count(ctx, analytics.Worker, analytics.Listen, analytics.Props("operator_kind", string(sqlcv1.V1OperatorKindGRPC)))

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

	worker, err := s.authorizeOperatorWorker(ctx, op, start.WorkerId)

	if err != nil {
		return err
	}

	l := s.l.With().
		Str("tenant_id", tenant.ID.String()).
		Str("operator_name", op.Name).
		Str("operator_id", op.ID.String()).
		Str("worker_id", worker.ID.String()).
		Logger()

	// the stream that delivers actions lives on this dispatcher, so a worker resumed after a
	// reconnect to another engine replica is re-pinned here
	if worker.DispatcherId == nil || *worker.DispatcherId != s.dispatcherId {
		dispatcherId := s.dispatcherId

		_, err = s.workers.UpdateWorker(ctx, tenant.ID, worker.ID, &repository.UpdateWorkerOpts{
			DispatcherId: &dispatcherId,
		})

		if err != nil {
			l.Error().Ctx(ctx).Err(err).Msg("could not update worker dispatcher")
			return err
		}
	}

	// The session id is both the dispatcher's session key and the listener fence on the worker
	// row: activation records it, and the deactivation below only succeeds while it is still
	// the id on the row, so a session superseded by a newer listener on the same worker never
	// marks the live session's worker inactive.
	sessionId := uuid.New()

	_, err = s.workers.ActivateWorkerListener(ctx, tenant.ID, worker.ID, sessionId)

	if err != nil {
		l.Error().Ctx(ctx).Err(err).Msgf("could not activate worker for listener session %s", sessionId)
		return err
	}

	// Deactivation runs detached from ctx because the most common exit is the client
	// disconnecting, at which point ctx is already cancelled.
	defer func() {
		deactivateCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), deactivateTimeout)
		defer cancel()

		deactivateErr := s.deactivateWorkerListener(deactivateCtx, &l, tenant.ID, worker.ID, sessionId)

		if deactivateErr != nil && err == nil {
			err = deactivateErr
		}
	}()

	session := s.dispatcher.AddOperatorStreamSession(worker.ID, sessionId, stream, wrapAssignedAction)
	// the release runs before the deferred deactivation so the session is gone from the
	// dispatcher before the worker is marked inactive
	defer session.Release()

	fin := session.Fin()

	// the session notify goes through the notifier so a burst of deltas right after start folds
	// into the same throttle window
	notifier := newThrottledNotifier(ctx, s.dispatcher, tenant, worker.ID, s.notifyInterval)
	defer notifier.stop()

	notifier.fire()

	l.Info().Ctx(ctx).Msg("operator worker listening")

	var lastHeartbeatWrite time.Time

	for {
		select {
		case <-fin:
			l.Debug().Ctx(ctx).Msg("dispatcher closed the operator stream")
			return nil
		case <-ctx.Done():
			l.Debug().Ctx(ctx).Msg("operator worker disconnected")
			return nil
		case <-notifier.due():
			notifier.fire()
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
				now := time.Now().UTC()

				if now.Sub(lastHeartbeatWrite) < heartbeatWriteInterval {
					continue
				}

				lastHeartbeatWrite = now

				if err := s.workers.UpdateWorkerHeartbeat(ctx, tenant.ID, worker.ID, now); err != nil {
					l.Error().Ctx(ctx).Err(err).Msg("could not update worker heartbeat")
				}
			case *v1contracts.OperatorListenRequest_Actions:
				changed, err := s.applyActionsDelta(ctx, &l, tenant.ID, worker.ID, msg.Actions)

				if err != nil {
					return err
				}

				if changed {
					notifier.request()
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

// deactivateWorkerListener marks the worker inactive on behalf of the listener session. A
// superseded session (one whose id is no longer recorded on the worker) has nothing to do,
// because the newer session owns the worker's active flag.
func (s *OperatorServiceImpl) deactivateWorkerListener(ctx context.Context, l *zerolog.Logger, tenantId, workerId, sessionId uuid.UUID) error {
	_, err := s.workers.DeactivateWorkerListener(ctx, tenantId, workerId, sessionId)

	if err == nil {
		return nil
	}

	if errors.Is(err, pgx.ErrNoRows) {
		l.Debug().Ctx(ctx).Msgf("listener session %s was superseded by a newer session, leaving worker active", sessionId)
		return nil
	}

	l.Error().Ctx(ctx).Err(err).Msgf("could not deactivate worker for listener session %s", sessionId)
	return err
}

// applyActionsDelta validates and applies one delta to the worker's action set. It reports
// whether the set changed; a delta that only repeats what the worker already has needs no
// scheduler notification.
func (s *OperatorServiceImpl) applyActionsDelta(ctx context.Context, l *zerolog.Logger, tenantId, workerId uuid.UUID, delta *v1contracts.OperatorActionsDelta) (bool, error) {
	if n := len(delta.Add) + len(delta.Remove); n > MaxActionsPerDelta {
		return false, status.Errorf(codes.InvalidArgument, "actions delta carries %d ids, the limit is %d per message", n, MaxActionsPerDelta)
	}

	if err := s.v.Validate(actionsDeltaOpts{Add: delta.Add, Remove: delta.Remove}); err != nil {
		return false, status.Errorf(codes.InvalidArgument, "invalid actions delta: %s", err.Error())
	}

	changed := false

	if len(delta.Add) > 0 {
		added, err := s.workers.AddWorkerActions(ctx, tenantId, workerId, delta.Add)

		if err != nil {
			l.Error().Ctx(ctx).Err(err).Msg("could not add worker actions")
			return false, err
		}

		changed = changed || added > 0
	}

	if len(delta.Remove) > 0 {
		removed, err := s.workers.RemoveWorkerActions(ctx, tenantId, workerId, delta.Remove)

		if err != nil {
			l.Error().Ctx(ctx).Err(err).Msg("could not remove worker actions")
			return false, err
		}

		changed = changed || removed > 0
	}

	l.Debug().Ctx(ctx).
		Int("add", len(delta.Add)).
		Int("remove", len(delta.Remove)).
		Bool("changed", changed).
		Msg("applied operator actions delta")

	return changed, nil
}

// throttledNotifier coalesces scheduler notifications for one worker: the first request fires
// immediately, and requests that arrive within interval of the last fire are folded into one
// notification sent when the interval elapses.
type throttledNotifier struct {
	ctx      context.Context
	d        dispatcherBackend
	tenant   *sqlcv1.Tenant
	workerId uuid.UUID
	interval time.Duration

	lastFire time.Time
	timer    *time.Timer
	pending  bool
}

func newThrottledNotifier(ctx context.Context, d dispatcherBackend, tenant *sqlcv1.Tenant, workerId uuid.UUID, interval time.Duration) *throttledNotifier {
	return &throttledNotifier{ctx: ctx, d: d, tenant: tenant, workerId: workerId, interval: interval}
}

// request asks for a notification. It runs on the Listen goroutine, like due and fire.
func (n *throttledNotifier) request() {
	since := time.Since(n.lastFire)

	if since >= n.interval {
		n.fire()
		return
	}

	if !n.pending {
		n.pending = true
		n.timer = time.NewTimer(n.interval - since)
	}
}

// due returns the channel that fires when a pending notification is ready; nil when nothing
// is pending, which never selects.
func (n *throttledNotifier) due() <-chan time.Time {
	if n.timer == nil {
		return nil
	}

	return n.timer.C
}

func (n *throttledNotifier) fire() {
	n.stop()
	n.lastFire = time.Now()
	n.d.NotifyNewWorker(n.ctx, n.tenant, n.workerId)
}

func (n *throttledNotifier) stop() {
	if n.timer != nil {
		n.timer.Stop()
		n.timer = nil
	}

	n.pending = false
}
