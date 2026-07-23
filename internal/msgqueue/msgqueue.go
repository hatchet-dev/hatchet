package msgqueue

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/pkg/random"
)

type Queue interface {
	// Name returns the name of the queue.
	Name() string

	// Durable returns true if this queue should survive task queue restarts.
	Durable() bool

	// AutoDeleted returns true if this queue should be deleted when the last consumer unsubscribes.
	AutoDeleted() bool

	// Exclusive returns true if this queue should only be accessed by the current connection.
	Exclusive() bool

	// DLQ returns the queue's dead letter queue, if it exists.
	DLQ() Queue

	// IsDLQ returns true if the queue is a dead letter queue.
	IsDLQ() bool

	// We distinguish between a static DLQ or an automatic DLQ. An automatic DLQ will automatically retry messages
	// in a loop with a 5-second backoff, and each subscription to the regular Queue will include a subscription
	// to the DLQ.
	IsAutoDLQ() bool

	// IsExpirable refers to whether the queue itself is expirable
	IsExpirable() bool
}

type staticQueue string

const (
	TASK_PROCESSING_QUEUE        staticQueue = "task_processing_queue_v2"
	OLAP_QUEUE                   staticQueue = "olap_queue_v2"
	DISPATCHER_DEAD_LETTER_QUEUE staticQueue = "dispatcher_dlq_v2"
	TICKER_UPDATE_QUEUE          staticQueue = "ticker_update_queue_v2"
)

func (s staticQueue) Name() string {
	return string(s)
}

func (s staticQueue) Durable() bool {
	return true
}

func (s staticQueue) AutoDeleted() bool {
	return false
}

func (s staticQueue) Exclusive() bool {
	return false
}

func (s staticQueue) DLQ() Queue {
	name := fmt.Sprintf("%s_dlq", s)

	return dlq{
		staticQueue: staticQueue(name),
		isAutoDLQ:   true,
	}
}

func (s staticQueue) IsDLQ() bool {
	return false
}

func (s staticQueue) IsAutoDLQ() bool {
	return false
}

func (s staticQueue) IsExpirable() bool {
	return false
}

type dlq struct {
	staticQueue

	isAutoDLQ bool
}

func (d dlq) IsDLQ() bool {
	return true
}

func (d dlq) IsAutoDLQ() bool {
	return d.isAutoDLQ
}

func (d dlq) DLQ() Queue {
	return nil
}

func NewRandomStaticQueue() staticQueue {
	randBytes, _ := random.Generate(8)
	return staticQueue(fmt.Sprintf("random_static_queue_v2_%s", randBytes))
}

// dispatcherQueue is a type of queue which is durable and exclusive, but utilizes a per-queue TTL
// and per-message TTL + DLX to handle messages which are not consumed within a certain time period,
// for example if the dispatcher goes down.
type dispatcherQueue string

func (d dispatcherQueue) Name() string {
	return string(d)
}

func (d dispatcherQueue) Durable() bool {
	return true
}

func (d dispatcherQueue) AutoDeleted() bool {
	return false
}

func (d dispatcherQueue) Exclusive() bool {
	return true
}

func (d dispatcherQueue) DLQ() Queue {
	return dlq{
		staticQueue: DISPATCHER_DEAD_LETTER_QUEUE,
		// we maintain a completely separate DLQ for dispatcher queues
		isAutoDLQ: false,
	}
}

func (d dispatcherQueue) IsDLQ() bool {
	return false
}

func (d dispatcherQueue) IsAutoDLQ() bool {
	return false
}

func (d dispatcherQueue) IsExpirable() bool {
	return true
}

func QueueTypeFromDispatcherID(d uuid.UUID) dispatcherQueue {
	return dispatcherQueue(d.String() + "_dispatcher_v1")
}

// MsgHandler processes a received message. On the durable MessageQueue it is
// invoked as a pre-ack or post-ack hook; on the best-effort PubSub it is the
// sole handler with no ack semantics.
type MsgHandler func(task *Message) error

type MessageQueue interface {
	// Clone copies the message queue with a new instance.
	Clone() (func() error, MessageQueue, error)

	// SetQOS sets the quality of service for the message queue.
	SetQOS(prefetchCount int)

	// SendMessage sends a message to the message queue.
	SendMessage(ctx context.Context, queue Queue, msg *Message) error

	// Subscribe subscribes to the task queue. It returns a cleanup function that should be called when the
	// subscription is no longer needed.
	Subscribe(queue Queue, preAck MsgHandler, postAck MsgHandler) (func() error, error)

	// IsReady returns true if the task queue is ready to accept tasks.
	IsReady() bool
}

func NoOpHook(task *Message) error {
	return nil
}
