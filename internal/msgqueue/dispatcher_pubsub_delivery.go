package msgqueue

import (
	"context"
	"math/rand/v2"

	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/pkg/integrations/metrics/prometheus"
)

// dispatcherPubSubDeliveryQueue sends every message for a dispatcher over pub/sub instead of the
// dispatcher's durable queue, and passes everything else through. Wrapping the queue rather than each
// sender moves all of them (scheduler, task controller, durable callbacks) at once.
type dispatcherPubSubDeliveryQueue struct {
	MessageQueue

	pubsub      PubSub
	dropPercent int
}

// NewDispatcherPubSubDeliveryQueue loses the durable queue's dead-letter path: a message for a dispatcher
// that is gone is dropped instead of being failed back to the scheduler. Unacknowledged assignments
// are recovered by the dispatcher's database pull and by ReassignUnackedTasks instead.
func NewDispatcherPubSubDeliveryQueue(mq MessageQueue, pubsub PubSub, dropPercent int) MessageQueue {
	return &dispatcherPubSubDeliveryQueue{
		MessageQueue: mq,
		pubsub:       pubsub,
		dropPercent:  dropPercent,
	}
}

func (q *dispatcherPubSubDeliveryQueue) Clone() (func() error, MessageQueue, error) {
	cleanup, cloned, err := q.MessageQueue.Clone()

	if err != nil {
		return nil, nil, err
	}

	return cleanup, NewDispatcherPubSubDeliveryQueue(cloned, q.pubsub, q.dropPercent), nil
}

func (q *dispatcherPubSubDeliveryQueue) SendMessage(ctx context.Context, queue Queue, msg *Message) error {
	dq, ok := queue.(dispatcherQueue)

	if !ok {
		return q.MessageQueue.SendMessage(ctx, queue, msg)
	}

	dispatcherId, err := dq.dispatcherID()

	if err != nil {
		return q.MessageQueue.SendMessage(ctx, queue, msg)
	}

	if q.dropPercent > 0 && rand.IntN(100) < q.dropPercent { // nolint: gosec
		prometheus.DispatcherPubSubDeliveryMessages.WithLabelValues(msg.ID, "dropped").Inc()
		return nil
	}

	if err := q.pubsub.Pub(ctx, DispatcherTopic(dispatcherId), msg); err != nil {
		prometheus.DispatcherPubSubDeliveryMessages.WithLabelValues(msg.ID, "error").Inc()
		return err
	}

	prometheus.DispatcherPubSubDeliveryMessages.WithLabelValues(msg.ID, "sent").Inc()

	return nil
}

func (d dispatcherQueue) dispatcherID() (uuid.UUID, error) {
	const suffix = "_dispatcher_v1"

	name := string(d)

	return uuid.Parse(name[:len(name)-len(suffix)])
}
