package dispatcher

import (
	"time"

	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/internal/services/shared/timeout_lock"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
)

type subscribedWorker struct {
	stream     contracts.Dispatcher_ListenServer
	finished   chan<- bool
	sendLock   *timeout_lock.TimeoutLock
	olapOutbox *v1.OLAPOutbox
	workerId   uuid.UUID
}

func newSubscribedWorker(
	stream contracts.Dispatcher_ListenServer,
	fin chan<- bool,
	workerId uuid.UUID,
	maxLockAcquisitionTime time.Duration,
	olapOutbox *v1.OLAPOutbox,
) *subscribedWorker {
	lock := timeout_lock.NewTimeoutLock(maxLockAcquisitionTime)
	return &subscribedWorker{
		stream:     stream,
		finished:   fin,
		workerId:   workerId,
		olapOutbox: olapOutbox,
		sendLock:   lock,
	}
}
