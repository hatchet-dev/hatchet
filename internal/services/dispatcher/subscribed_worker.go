package dispatcher

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	"github.com/hatchet-dev/hatchet/internal/services/shared/timeout_lock"
	"github.com/hatchet-dev/hatchet/pkg/operator"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
)

type subscribedWorker struct {
	// stream is the server stream actions are encoded onto; nil for operator-backed workers
	stream    grpc.ServerStream
	finished  chan<- bool
	sendLock  *timeout_lock.TimeoutLock
	pubBuffer *msgqueue.MQPubBuffer

	// wrap converts an assigned action into the stream's server message type before it is
	// encoded; nil when the stream's server message type is AssignedAction itself
	wrap func(*contracts.AssignedAction) proto.Message

	// done is closed by markDone when the session's owner releases it, so a shutdown drain
	// that captured the session before the release does not block on finished; nil for
	// sessions whose owner always selects on finished
	done     chan struct{}
	doneOnce sync.Once

	// handler is the direct delivery target for an in-process operator session; nil for
	// stream-backed workers
	handler  operator.ActionHandler
	workerId uuid.UUID

	// paused is set by an operator session that has paused its worker: a start assigned to the
	// worker is returned to the queue instead of delivered, the way a failed send is, so that
	// once the operator holds the pause's ack nothing more arrives. Cancels still go through,
	// since the paused operator may still be draining the runs they name.
	paused atomic.Bool
}

func newGRPCSubscribedWorker(
	stream grpc.ServerStream,
	fin chan<- bool,
	workerId uuid.UUID,
	maxLockAcquisitionTime time.Duration,
	pubBuffer *msgqueue.MQPubBuffer,
) *subscribedWorker {
	lock := timeout_lock.NewTimeoutLock(maxLockAcquisitionTime)
	return &subscribedWorker{
		stream:    stream,
		finished:  fin,
		workerId:  workerId,
		pubBuffer: pubBuffer,
		sendLock:  lock,
	}
}

func newOperatorSubscribedWorker(
	workerId uuid.UUID,
	pubBuffer *msgqueue.MQPubBuffer,
	handler operator.ActionHandler,
) *subscribedWorker {
	return &subscribedWorker{
		workerId:  workerId,
		pubBuffer: pubBuffer,
		handler:   handler,
	}
}

// setPaused makes the worker return every start it is asked to deliver to the queue, or
// deliver again.
func (worker *subscribedWorker) setPaused(paused bool) {
	worker.paused.Store(paused)
}

// markDone records that the session's owner has released it.
func (worker *subscribedWorker) markDone() {
	if worker.done == nil {
		return
	}

	worker.doneOnce.Do(func() { close(worker.done) })
}

// requestFin asks the stream's owner to hang up. It blocks until the owner receives the
// signal or, for sessions that carry done, until the owner has released the session.
func (worker *subscribedWorker) requestFin() {
	select {
	case worker.finished <- true:
	case <-worker.done:
	}
}
