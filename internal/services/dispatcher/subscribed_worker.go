package dispatcher

import (
	"sync"

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
	sendMuAcquisitionTimeout int64

	workerId uuid.UUID

	pubBuffer *msgqueue.MQPubBuffer
}

func newSubscribedWorker(
	stream contracts.Dispatcher_ListenServer,
	fin chan<- bool,
	workerId uuid.UUID,
	maxLockAcquisitionTimeMS int64,
	pubBuffer *msgqueue.MQPubBuffer,
) *subscribedWorker {

	return &subscribedWorker{
		stream:                   stream,
		finished:                 fin,
		workerId:                 workerId,
		pubBuffer:                pubBuffer,
		sendMuAcquisitionTimeout: maxLockAcquisitionTimeMS,
	}
}
