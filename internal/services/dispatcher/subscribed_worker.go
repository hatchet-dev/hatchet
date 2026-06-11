package dispatcher

import (
	"time"

	"github.com/google/uuid"
	"github.com/hatchet-dev/hatchet/internal/services/shared/timeout_lock"
	"github.com/hatchet-dev/hatchet/pkg/operator"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
)

type subscribedWorker struct {
	// stream is the server side of the RPC stream
	stream contracts.Dispatcher_ListenServer

	// finished is used to signal closure of a client subscribing goroutine
	finished chan<- bool

	sendLock *timeout_lock.TimeoutLock

	workerId uuid.UUID

	pubBuffer *msgqueue.MQPubBuffer

	// optional: the operator backing this worker
	operator operator.Operator
}

func newGRPCSubscribedWorker(
	stream contracts.Dispatcher_ListenServer,
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
	fin chan<- bool,
	workerId uuid.UUID,
	pubBuffer *msgqueue.MQPubBuffer,
	operator operator.Operator,
) *subscribedWorker {
	return &subscribedWorker{
		finished:  fin,
		workerId:  workerId,
		pubBuffer: pubBuffer,
		operator:  operator,
	}
}
