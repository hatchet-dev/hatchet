package msgqueue

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// Topic identifies a best-effort pub/sub destination.
type Topic struct {
	name         string
	tenantStream bool
}

func (t Topic) Name() string {
	return t.name
}

// IsTenantStream returns true if this topic feeds the dispatcher's per-tenant
// gRPC streams (as opposed to a scheduler partition wake-up).
func (t Topic) IsTenantStream() bool {
	return t.tenantStream
}

// TenantTopic feeds the dispatcher's gRPC streams. The wire name matches the
// legacy tenant fanout exchange / NOTIFY name ("<uuid>_v1") so mixed-version
// fleets interoperate.
func TenantTopic(tenantId uuid.UUID) Topic {
	return Topic{
		name:         tenantId.String() + "_v1",
		tenantStream: true,
	}
}

// SchedulerPartitionTopic wakes the scheduler owning a partition. The wire
// name matches the legacy consumer queue ("<partitionId>_scheduler_v1") so
// mixed-version fleets interoperate.
func SchedulerPartitionTopic(partitionId string) Topic {
	return Topic{
		name: fmt.Sprintf("%s_scheduler_v1", partitionId),
	}
}

// PubSub is a best-effort, non-durable, at-most-once pub/sub mechanism.
//
// INVARIANT: implementations own their pooled resources (AMQP connections and
// channel pools, pgx pools) and never share them with the durable MessageQueue
// or the repository layer, since Pub can be called from within durable-write
// paths and sharing pools is a deadlock risk.
type PubSub interface {
	// Pub publishes a message to the topic. Delivery is best-effort: if no
	// subscriber is listening, the message is dropped.
	Pub(ctx context.Context, topic Topic, msg *Message) error

	// Sub subscribes to a topic. Delivery is at-most-once with no ack
	// semantics: handler errors are logged, never redelivered. It returns a
	// cleanup function that should be called when the subscription is no
	// longer needed.
	Sub(topic Topic, handler AckHook) (func() error, error)

	// IsReady returns true if the pub/sub mechanism is ready to accept messages.
	IsReady() bool
}

// gatedPubSub suppresses tenant-stream publishes when disabled, except
// task-stream-event (worker stream output must always flow — it has no other
// delivery path). Scheduler partition wake-ups are never gated.
type gatedPubSub struct {
	inner                   PubSub
	disableTenantStreamPubs bool
}

// NewGatedPubSub wraps a PubSub so that tenant-stream publishes are dropped
// when disableTenantStreamPubs is true, except task-stream-event messages.
// Scheduler partition wake-ups always pass through.
func NewGatedPubSub(inner PubSub, disableTenantStreamPubs bool) PubSub {
	if !disableTenantStreamPubs {
		return inner
	}

	return &gatedPubSub{
		inner:                   inner,
		disableTenantStreamPubs: disableTenantStreamPubs,
	}
}

func (g *gatedPubSub) Pub(ctx context.Context, topic Topic, msg *Message) error {
	if g.disableTenantStreamPubs && topic.IsTenantStream() && msg.ID != MsgIDTaskStreamEvent {
		return nil
	}

	return g.inner.Pub(ctx, topic, msg)
}

func (g *gatedPubSub) Sub(topic Topic, handler AckHook) (func() error, error) {
	return g.inner.Sub(topic, handler)
}

func (g *gatedPubSub) IsReady() bool {
	return g.inner.IsReady()
}
