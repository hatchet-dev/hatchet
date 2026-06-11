package dispatcher

import (
	"time"

	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/internal/services/shared/timeout_lock"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
)

type subscribedWorker struct {
	stream    contracts.Dispatcher_ListenServer
	finished  chan<- bool
	sendLock  *timeout_lock.TimeoutLock
	pubBuffer *msgqueue.MQPubBuffer
	workerId  uuid.UUID
}

func newSubscribedWorker(
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
