package dispatcher

import (
	"context"
	"sync"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	"github.com/hatchet-dev/hatchet/pkg/telemetry"
)

type subscribedWorker struct {
	// stream is the server side of the RPC stream
	stream contracts.Dispatcher_ListenServer

	// finished is used to signal closure of a client subscribing goroutine
	finished chan<- bool

	sendMu sync.Mutex

	workerId string

	backlogSize   int64
	backlogSizeMu sync.Mutex

	maxBacklogSize int64

	pubBuffer *msgqueue.MQPubBuffer
}

func newSubscribedWorker(
	stream contracts.Dispatcher_ListenServer,
	fin chan<- bool,
	workerId string,
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

// TODO looks like alexander removed these local methods...
func (worker *subscribedWorker) StartBatch(
	ctx context.Context,
	action *contracts.AssignedAction,
) error {
	_, span := telemetry.NewSpan(ctx, "start-batch")
	defer span.End()

	worker.sendMu.Lock()
	defer worker.sendMu.Unlock()

	return worker.stream.Send(action)
}
