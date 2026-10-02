package dispatcher

import (
	"context"

	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	v1contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/internal/services/shared/rpcstream"
	"github.com/hatchet-dev/hatchet/pkg/operator"
)

// OperatorStreamSession is a live dispatcher session backed by an OperatorService Listen stream
// that an operator service owns. The dispatcher fans assigned actions out on the stream; the
// owner sends its own protocol messages through Send. Both go through the one sender that
// guards the stream, hangs up when Fin fires, and pauses delivery through SetPaused.
type OperatorStreamSession struct {
	worker  *subscribedWorker
	sender  *rpcstream.Sender[v1contracts.OperatorListenResponse]
	fin     <-chan bool
	release func()
}

// operatorListenStream carries assigned actions on a Listen stream, whose message type wraps
// them, so the fan-out can treat the stream like an SDK worker's.
type operatorListenStream struct {
	sender *rpcstream.Sender[v1contracts.OperatorListenResponse]
}

func (s operatorListenStream) Send(action *contracts.AssignedAction) error {
	return s.sender.Send(&v1contracts.OperatorListenResponse{
		Message: &v1contracts.OperatorListenResponse_Action{Action: action},
	})
}

// AddOperatorStreamSession registers a Listen stream owned by an out-of-process operator as a
// live session for workerId, so the dispatcher fans assigned actions out to it like any SDK
// worker. sender is the stream's guarded sender, which the handler closes before it returns.
// The caller chooses sessionId so the dispatcher's session key is the same id it records on
// the worker row as the listener session fence.
//
// The dispatcher signals Fin when it wants the stream hung up (the shutdown drain in Start's
// cleanup). The caller must call Release when its handler exits; Release removes the session
// and lets a concurrent drain skip it, so a handler that has stopped selecting on Fin never
// blocks the drain. A send that is in progress when Release runs finishes or fails on its own
// once the handler closes the sender; Release does not wait for it.
func (d *DispatcherImpl) AddOperatorStreamSession(
	ctx context.Context,
	workerId uuid.UUID,
	sessionId uuid.UUID,
	sender *rpcstream.Sender[v1contracts.OperatorListenResponse],
) *OperatorStreamSession {
	finCh := make(chan bool)

	stream := rpcstream.NewSender[contracts.AssignedAction](ctx, operatorListenStream{sender: sender})
	worker := newGRPCSubscribedWorker(stream, finCh, workerId, d.defaultMaxWorkerLockAcquisitionTime, d.pubBuffer)
	worker.done = make(chan struct{})

	d.workers.Add(workerId, sessionId, worker)

	return &OperatorStreamSession{
		worker: worker,
		sender: sender,
		fin:    finCh,
		release: func() {
			worker.markDone()
			d.workers.DeleteForSession(workerId, sessionId)
		},
	}
}

// Fin fires when the dispatcher wants the stream hung up.
func (s *OperatorStreamSession) Fin() <-chan bool {
	return s.fin
}

// Send writes msg on the stream, serialised with the dispatcher's own action sends. It fails
// with errSessionReleased after Release and with rpcstream.ErrClosed once the handler has
// closed the sender.
func (s *OperatorStreamSession) Send(ctx context.Context, msg *v1contracts.OperatorListenResponse) error {
	select {
	case <-s.worker.done:
		return errSessionReleased
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	return s.sender.Send(msg)
}

// SetPaused makes the dispatcher return every start assigned to the worker to the queue
// instead of sending it, until SetPaused(false). The owner sets it before it acknowledges a
// pause to the operator, so an action the scheduler assigned before it observed the pause is
// requeued rather than delivered after the ack. Cancels are still sent.
func (s *OperatorStreamSession) SetPaused(paused bool) {
	s.worker.setPaused(paused)
}

// Release removes the session from the dispatcher. It is idempotent.
func (s *OperatorStreamSession) Release() {
	s.release()
}

// OperatorHandlerSession is a live dispatcher session backed by an operator running in this
// process: assigned actions are handed to its handler directly, with no encoding and no stream.
// There is no Fin signal, because there is no stream to hang up, and the shutdown drain skips
// these sessions: the host that opened the session owns its teardown.
type OperatorHandlerSession struct {
	worker  *subscribedWorker
	release func()
}

// AddOperatorSession registers an in-process operator as a live session for workerId, so the
// dispatcher routes assigned actions to it like any other worker. The caller chooses sessionId
// so the dispatcher's session key is the same id it records on the worker row as the listener
// session fence. The caller must call Release when the session ends.
func (d *DispatcherImpl) AddOperatorSession(
	workerId uuid.UUID,
	sessionId uuid.UUID,
	handler operator.ActionHandler,
) *OperatorHandlerSession {
	worker := newOperatorSubscribedWorker(workerId, d.pubBuffer, handler)

	d.workers.Add(workerId, sessionId, worker)

	return &OperatorHandlerSession{
		worker: worker,
		release: func() {
			d.workers.DeleteForSession(workerId, sessionId)
		},
	}
}

// SetPaused is the same as OperatorStreamSession.SetPaused: starts assigned to the worker are
// returned to the queue instead of handed to the handler.
func (s *OperatorHandlerSession) SetPaused(paused bool) {
	s.worker.setPaused(paused)
}

// Release removes the session from the dispatcher. It is idempotent.
func (s *OperatorHandlerSession) Release() {
	s.release()
}
