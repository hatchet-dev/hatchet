package dispatcher

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
)

// OperatorStreamSession is a live dispatcher session backed by a gRPC stream that an operator
// service owns. The dispatcher fans assigned actions out on the stream; the owner sends its own
// protocol messages through Send, which shares the per-stream send serialisation with the
// fan-out, and hangs the stream up when Fin fires.
type OperatorStreamSession struct {
	worker  *subscribedWorker
	fin     <-chan bool
	release func()
}

// AddOperatorStreamSession registers a gRPC stream owned by an out-of-process operator as a live
// session for workerId, so the dispatcher fans assigned actions out to it like any SDK worker.
// wrap converts each assigned action into the stream's server message type before it is
// encoded; nil means the stream's server message type is AssignedAction itself. The caller
// chooses sessionId so the dispatcher's session key is the same id it records on the worker
// row as the listener session fence.
//
// The dispatcher signals Fin when it wants the stream hung up (the shutdown drain in Start's
// cleanup). The caller must call Release when its handler exits; Release removes the session
// and lets a concurrent drain skip it, so a handler that has stopped selecting on Fin never
// blocks the drain. A send that is in progress when Release runs finishes or fails on its own
// once the handler returns and gRPC tears the stream down; Release does not wait for it.
func (d *DispatcherImpl) AddOperatorStreamSession(
	workerId uuid.UUID,
	sessionId uuid.UUID,
	stream grpc.ServerStream,
	wrap func(*contracts.AssignedAction) proto.Message,
) *OperatorStreamSession {
	finCh := make(chan bool)

	worker := newGRPCSubscribedWorker(stream, finCh, workerId, d.defaultMaxWorkerLockAcquisitionTime, d.pubBuffer)
	worker.wrap = wrap
	worker.done = make(chan struct{})

	d.workers.Add(workerId, sessionId, worker)

	return &OperatorStreamSession{
		worker: worker,
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
// with errFlowControlActive when the stream is blocked by flow control for longer than the
// dispatcher's send lock timeout, and with errSessionReleased after Release.
func (s *OperatorStreamSession) Send(ctx context.Context, msg proto.Message) error {
	return s.worker.sendMsg(ctx, msg)
}

// Release removes the session from the dispatcher. It is idempotent.
func (s *OperatorStreamSession) Release() {
	s.release()
}
