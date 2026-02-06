package dispatcher

import (
	"sync"

	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
)

type subscribedWorker struct {
	stream         contracts.Dispatcher_ListenServer
	finished       chan<- bool
	pubBuffer      *msgqueue.MQPubBuffer
	backlogSize    int64
	maxBacklogSize int64
	sendMu         sync.Mutex
	backlogSizeMu  sync.Mutex
	workerId       uuid.UUID
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
