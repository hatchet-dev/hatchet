package msgqueue

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"maps"
	"slices"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// TopicKind identifies the kind of pub/sub destination a Topic refers to.
type TopicKind string

const (
	// TopicKindTenantStream feeds the dispatcher's per-tenant gRPC streams.
	TopicKindTenantStream TopicKind = "tenant-stream"

	// TopicKindSchedulerPartition wakes the scheduler owning a partition.
	TopicKindSchedulerPartition TopicKind = "scheduler-partition"

	// TopicKindStreamWake fans durable-stream wakes out to every engine.
	TopicKindStreamWake TopicKind = "stream-wake"
)

// Topic identifies a best-effort pub/sub destination.
type Topic struct {
	name string
	kind TopicKind
}

func (t Topic) Name() string {
	return t.name
}

func (t Topic) Kind() TopicKind {
	return t.kind
}

// TenantTopic feeds the dispatcher's gRPC streams. The wire name matches the
// legacy tenant fanout exchange / NOTIFY name ("<uuid>_v1") so mixed-version
// fleets interoperate.
func TenantTopic(tenantId uuid.UUID) Topic {
	return Topic{
		name: tenantId.String() + "_v1",
		kind: TopicKindTenantStream,
	}
}

// SchedulerPartitionTopic wakes the scheduler owning a partition. The wire
// name matches the legacy consumer queue ("<partitionId>_scheduler_v1") so
// mixed-version fleets interoperate.
func SchedulerPartitionTopic(partitionId string) Topic {
	return Topic{
		name: fmt.Sprintf("%s_scheduler_v1", partitionId),
		kind: TopicKindSchedulerPartition,
	}
}

// StreamWakeTopic carries every durable-stream wake, each engine holding one
// subscription for all of its tailed topics. The topics woken travel in the
// payload (see StreamWake), so user-chosen names never reach a subject,
// queue, or channel name.
func StreamWakeTopic() Topic {
	return Topic{
		name: "stream_wake_v1",
		kind: TopicKindStreamWake,
	}
}

// StreamWake is one payload of a StreamWakeTopic message; the tenant is the
// message's TenantID.
type StreamWake struct {
	Namespace string `json:"namespace"`
	Topic     string `json:"topic"`
}

// maxStreamWakeBytes keeps each wake message well under pg_notify's 8KB
// limit, past which the postgres pubsub delivers to only one subscriber.
const maxStreamWakeBytes = 4 * 1024

// streamWakeWireSize approximates w's size inside a published message: its
// JSON, base64-encoded as a Message payload, plus separators.
func streamWakeWireSize(w StreamWake) int {
	const jsonOverhead = len(`{"namespace":"","topic":""}`)

	return base64.StdEncoding.EncodedLen(len(w.Namespace)+len(w.Topic)+jsonOverhead) + 3
}

// PubStreamWakes wakes every engine tailing one of tenantId's wakes, in as
// few messages as stay under maxStreamWakeBytes.
func PubStreamWakes(ctx context.Context, ps PubSub, tenantId uuid.UUID, wakes []StreamWake) error {
	var errs error

	for len(wakes) > 0 {
		n, size := 0, 0

		// names are capped well below maxStreamWakeBytes, so n >= 1
		for n < len(wakes) && (n == 0 || size+streamWakeWireSize(wakes[n]) <= maxStreamWakeBytes) {
			size += streamWakeWireSize(wakes[n])
			n++
		}

		wakeMsg, err := NewTenantMessage(tenantId, MsgIDStreamMessage, true, false, wakes[:n]...)

		if err != nil {
			return err
		}

		errs = errors.Join(errs, ps.Pub(ctx, StreamWakeTopic(), wakeMsg))
		wakes = wakes[n:]
	}

	return errs
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
	//
	// Fanout to multiple subscribers of the same topic is backend-dependent:
	// rabbitmq delivers every message to every subscriber, while the postgres
	// implementation delivers messages over 8KB to only one subscriber (see
	// postgres.PubSub.Sub).
	Sub(topic Topic, handler MsgHandler) (func() error, error)

	// IsReady returns true if the pub/sub mechanism is ready to accept messages.
	IsReady() bool
}

// tenantStreamMsgIDs enumerates the message IDs the dispatcher's gRPC streams
// consume from tenant topics (see workflowEventConverters and
// workflowRunMatchers in the dispatcher). Publishing any other ID to a
// tenant topic is pure waste: the streams are the only consumers.
//
// Guarded against drift by TestTenantStreamMsgIDsInSync in the dispatcher
// package: an ID consumed there but missing here is silently never published.
var tenantStreamMsgIDs = map[string]struct{}{
	MsgIDCreatedTask:                  {},
	MsgIDTaskCompleted:                {},
	MsgIDTaskFailed:                   {},
	MsgIDTaskCancelled:                {},
	MsgIDTaskStreamEvent:              {},
	MsgIDWorkflowRunFinished:          {},
	MsgIDWorkflowRunFinishedCandidate: {},
}

// TenantStreamMsgIDs returns the message IDs PubTenantMessage publishes to
// tenant-stream topics, sorted. It exists so the dispatcher can assert its
// stream consumers stay in sync with this allowlist.
func TenantStreamMsgIDs() []string {
	return slices.Sorted(maps.Keys(tenantStreamMsgIDs))
}

// PubTenantMessage writes a tenant-scoped message to its destinations: a
// durable send to queue via mq (when queue is non-nil), plus a publish to the
// tenant stream when the message ID is one the dispatcher's streams consume.
//
// Durable send errors are returned. Stream publish errors are best-effort
// (logged, never propagated) when the message also had a durable destination;
// when the stream is the message's only delivery path (queue == nil), the
// publish error is returned so callers can decide.
func PubTenantMessage(ctx context.Context, l *zerolog.Logger, mq MessageQueue, ps PubSub, queue Queue, msg *Message) error {
	if queue != nil {
		if err := mq.SendMessage(ctx, queue, msg); err != nil {
			return err
		}
	}

	if _, ok := tenantStreamMsgIDs[msg.ID]; !ok || msg.TenantID == uuid.Nil {
		return nil
	}

	if err := ps.Pub(ctx, TenantTopic(msg.TenantID), msg); err != nil {
		if queue == nil {
			return err
		}

		l.Warn().Ctx(ctx).Err(err).Str("message_id", msg.ID).Msg("could not publish message to tenant stream")
	}

	return nil
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
	if g.disableTenantStreamPubs && topic.Kind() == TopicKindTenantStream && msg.ID != MsgIDTaskStreamEvent {
		return nil
	}

	return g.inner.Pub(ctx, topic, msg)
}

func (g *gatedPubSub) Sub(topic Topic, handler MsgHandler) (func() error, error) {
	return g.inner.Sub(topic, handler)
}

func (g *gatedPubSub) IsReady() bool {
	return g.inner.IsReady()
}
