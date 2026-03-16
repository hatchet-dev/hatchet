package dispatcher

import (
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
)

type subscribedWorker struct {
	// stream is the server side of the RPC stream
	stream contracts.Dispatcher_ListenServer

	// finished is used to signal closure of a client subscribing goroutine
	finished chan<- bool

	sendMu                   sync.Mutex
	sendMuAcquisitionTimeout time.Duration

	workerId uuid.UUID

	pubBuffer *msgqueue.MQPubBuffer
}

func newSubscribedWorker(
	stream contracts.Dispatcher_ListenServer,
	fin chan<- bool,
	workerId uuid.UUID,
	maxLockAcquisitionTime time.Duration,
	pubBuffer *msgqueue.MQPubBuffer,
) *subscribedWorker {

	return &subscribedWorker{
		stream:                   stream,
		finished:                 fin,
		workerId:                 workerId,
		pubBuffer:                pubBuffer,
		sendMuAcquisitionTimeout: maxLockAcquisitionTime,
	}
}
