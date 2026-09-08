package dispatcher

import (
	"sync"
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

	// optional: the operator backing this worker
	operator operator.Operator
	workerId uuid.UUID
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
	operator operator.Operator,
) *subscribedWorker {
	return &subscribedWorker{
		workerId:  workerId,
		pubBuffer: pubBuffer,
		operator:  operator,
	}
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
