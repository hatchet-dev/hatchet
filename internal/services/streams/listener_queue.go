package streams

import (
	"context"
	"errors"

	contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
)

// topicListener is one Subscribe RPC. cancel ends the RPC when a send fails,
// which unregisters the listener.
type topicListener struct {
	send   func(*contracts.StreamMessage) error
	cancel context.CancelFunc
}

// listenerQueueSize is how many frames a listener can fall behind before it's
// dropped; its client resubscribes from the last cursor it received.
const listenerQueueSize = 64

var errListenerFellBehind = errors.New("stream listener fell too far behind")

// listenerQueue sends to one listener from its own goroutine, so a reader that
// stops reading blocks only itself, never the poller's lock.
type listenerQueue struct {
	listener *topicListener
	frames   chan *contracts.StreamMessage
}

func startListenerQueue(listener *topicListener) *listenerQueue {
	q := &listenerQueue{listener: listener, frames: make(chan *contracts.StreamMessage, listenerQueueSize)}

	go q.run()

	return q
}

// run ends the RPC once the queue is closed and drained, so a final hangup is
// written first, or as soon as a send fails.
func (q *listenerQueue) run() {
	for frame := range q.frames {
		if err := q.listener.send(frame); err != nil {
			break
		}
	}

	q.listener.cancel()

	// after a failed send, discards frames until the poller removes this listener
	for range q.frames {
		continue
	}
}

// enqueue never blocks; false means the listener has fallen too far behind.
func (q *listenerQueue) enqueue(frame *contracts.StreamMessage) bool {
	select {
	case q.frames <- frame:
		return true
	default:
		return false
	}
}
