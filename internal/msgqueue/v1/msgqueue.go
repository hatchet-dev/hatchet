package v1

import (
	"context"
	"fmt"

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

	// FanoutExchangeKey returns which exchange the queue should be subscribed to. This is only currently relevant
	// to tenant pub/sub queues.
	//
	// In RabbitMQ terminology, the existence of a subscriber key means that the queue is bound to a fanout
	// exchange, and a new random queue is generated for each connection when connections are retried.
	FanoutExchangeKey() string

	// DLQ returns the queue's dead letter queue, if it exists.
	DLQ() Queue

	// IsDLQ returns true if the queue is a dead letter queue.
	IsDLQ() bool
}

type staticQueue string

const (
	TASK_PROCESSING_QUEUE staticQueue = "task_processing_queue_v2"
	OLAP_QUEUE            staticQueue = "olap_queue_v2"
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

func (s staticQueue) FanoutExchangeKey() string {
	return ""
}

func (s staticQueue) IsDLQ() bool {
	return false
}

type dlq struct {
	staticQueue
}

func (d dlq) IsDLQ() bool {
	return true
}

func (d dlq) DLQ() Queue {
	return nil
}

func (s staticQueue) DLQ() Queue {
	name := fmt.Sprintf("%s_dlq", s)

	return dlq{
		staticQueue: staticQueue(name),
	}
}

func NewRandomStaticQueue() staticQueue {
	randBytes, _ := random.Generate(8)
	return staticQueue(fmt.Sprintf("random_static_queue_v2_%s", randBytes))
}

type consumerQueue string

func (s consumerQueue) Name() string {
	return string(s)
}

func (n consumerQueue) Durable() bool {
	return false
}

func (n consumerQueue) AutoDeleted() bool {
	return true
}

func (n consumerQueue) Exclusive() bool {
	return true
}

func (n consumerQueue) FanoutExchangeKey() string {
	return ""
}

func (n consumerQueue) DLQ() Queue {
	return nil
}

func (n consumerQueue) IsDLQ() bool {
	return false
}

func QueueTypeFromDispatcherID(d string) consumerQueue {
	return consumerQueue(d + "_v1")
}

func QueueTypeFromTickerID(t string) consumerQueue {
	return consumerQueue(t + "_v1")
}

const (
	Scheduler = "scheduler"
)

func QueueTypeFromPartitionIDAndController(p, controller string) consumerQueue {
	return consumerQueue(fmt.Sprintf("%s_%s_v1", p, controller))
}

type fanoutQueue struct {
	consumerQueue
}

// The fanout exchange key for a consumer is the name of the consumer queue.
func (f fanoutQueue) FanoutExchangeKey() string {
	return f.consumerQueue.Name()
}

func TenantEventConsumerQueue(t string) fanoutQueue {
	// generate a unique queue name for the tenant
	return fanoutQueue{
		consumerQueue: consumerQueue(GetTenantExchangeName(t)),
	}
}

func GetTenantExchangeName(t string) string {
	return t + "_v1"
}

type AckHook func(task *Message) error

type MessageQueue interface {
	// Clone copies the message queue with a new instance.
	Clone() (func() error, MessageQueue)

	// SetQOS sets the quality of service for the message queue.
	SetQOS(prefetchCount int)

	// SendMessage sends a message to the message queue.
	SendMessage(ctx context.Context, queue Queue, msg *Message) error

	// Subscribe subscribes to the task queue. It returns a cleanup function that should be called when the
	// subscription is no longer needed.
	Subscribe(queue Queue, preAck AckHook, postAck AckHook) (func() error, error)

	// RegisterTenant registers a new pub/sub mechanism for a tenant. This should be called when a
	// new tenant is created. If this is not called, implementors should ensure that there's a check
	// on the first message to a tenant to ensure that the tenant is registered, and store the tenant
	// in an LRU cache which lives in-memory.
	RegisterTenant(ctx context.Context, tenantId string) error

	// IsReady returns true if the task queue is ready to accept tasks.
	IsReady() bool
}

func NoOpHook(task *Message) error {
	return nil
}
