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

	sendMu sync.Mutex

	workerId uuid.UUID

	backlogSize   int64
	backlogSizeMu sync.Mutex

	maxBacklogSize int64

	pubBuffer *msgqueue.MQPubBuffer
}

func newSubscribedWorker(
	stream contracts.Dispatcher_ListenServer,
	fin chan<- bool,
	workerId uuid.UUID,
	maxBacklogSize int64,
	pubBuffer *msgqueue.MQPubBuffer,
) *subscribedWorker {
	if maxBacklogSize <= 0 {
		maxBacklogSize = 20
	}

	return &subscribedWorker{
		stream:         stream,
		finished:       fin,
		workerId:       workerId,
		maxBacklogSize: maxBacklogSize,
		pubBuffer:      pubBuffer,
	}
}
