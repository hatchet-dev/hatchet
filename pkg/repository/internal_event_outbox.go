package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/hatchet-dev/pgoutbox"
	outboxsqlc "github.com/hatchet-dev/pgoutbox/sqlc"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/pkg/integrations/metrics/prometheus"
)

// internalEventOutboxBatchSize keeps a flush short: the outbox holds its topic row lock, and so blocks
// every other engine's pass over the shard, until the batch's matches are processed.
const internalEventOutboxBatchSize = 100

// InternalEventHandler processes one tenant's internal events. It must be idempotent: a batch whose
// processing succeeded is processed again if the outbox transaction then fails to commit.
type InternalEventHandler func(ctx context.Context, tenantId uuid.UUID, events []*InternalTaskEvent) error

// InternalEventOutbox writes the internal events a task completion produces to a pgoutbox topic in
// the completion's own transaction, so a crash after the commit can't lose the trigger for the next
// DAG step. Events are spread over several topics because pgoutbox processes one batch per topic at
// a time.
type InternalEventOutbox struct {
	outbox       pgoutbox.Outbox
	shards       int
	pollInterval time.Duration
	l            *zerolog.Logger

	// kicks wake this process's drain loop for a shard right after a commit, so the common path
	// doesn't wait for the poll; the poll only covers batches whose writer died before draining them.
	kicks     []chan struct{}
	startOnce sync.Once
}

type internalEventOutboxMessage struct {
	TenantID uuid.UUID           `json:"tenant_id"`
	Events   []InternalTaskEvent `json:"events"`
}

func NewInternalEventOutbox(outbox pgoutbox.Outbox, shards int, pollInterval time.Duration, l *zerolog.Logger) *InternalEventOutbox {
	if shards <= 0 {
		shards = 1
	}

	kicks := make([]chan struct{}, shards)

	for i := range kicks {
		kicks[i] = make(chan struct{}, 1)
	}

	return &InternalEventOutbox{
		outbox:       outbox,
		shards:       shards,
		pollInterval: pollInterval,
		l:            l,
		kicks:        kicks,
	}
}

func internalEventOutboxTopic(shard int) string {
	return fmt.Sprintf("v1_internal_events.%d", shard)
}

// shardFor keys on the tenant so one tenant's events stay ordered within a shard, which is the order
// the durable task queue delivered them in.
func (o *InternalEventOutbox) shardFor(tenantId uuid.UUID) int {
	h := fnv.New32a()
	_, _ = h.Write(tenantId[:])

	return int(h.Sum32() % uint32(o.shards)) // nolint: gosec
}

func (o *InternalEventOutbox) enqueue(ctx context.Context, tx pgx.Tx, tenantId uuid.UUID, events []InternalTaskEvent) error {
	payload, err := json.Marshal(internalEventOutboxMessage{TenantID: tenantId, Events: events})

	if err != nil {
		return fmt.Errorf("could not marshal internal events: %w", err)
	}

	return o.outbox.AddMessages(ctx, tx, internalEventOutboxTopic(o.shardFor(tenantId)), []pgoutbox.MessageOpts{{Payload: payload}})
}

// Kick wakes the drain loop for the tenant's shard. It never blocks: a pending kick already covers
// anything committed before the loop's next pass.
func (o *InternalEventOutbox) Kick(tenantId uuid.UUID) {
	select {
	case o.kicks[o.shardFor(tenantId)] <- struct{}{}:
	default:
	}
}

// Start drains every shard until ctx is cancelled. Only the first call starts the loops, so any
// process that can apply internal events may call it.
func (o *InternalEventOutbox) Start(ctx context.Context, handler InternalEventHandler) {
	o.startOnce.Do(func() {
		for shard := 0; shard < o.shards; shard++ {
			topic := internalEventOutboxTopic(shard)
			o.outbox.AddFlusher(topic, &internalEventFlusher{handler: handler})

			go o.drain(ctx, topic, o.kicks[shard])
		}
	})
}

func (o *InternalEventOutbox) drain(ctx context.Context, topic string, kick <-chan struct{}) {
	ticker := time.NewTicker(o.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-kick:
		case <-ticker.C:
		}

		for ctx.Err() == nil {
			msgs, err := o.outbox.ProcessMessages(ctx, topic, pgoutbox.WithBatchSize(internalEventOutboxBatchSize))

			if err != nil {
				// the batch stays in the outbox and is retried on the next kick or poll
				prometheus.InternalEventOutboxBatches.WithLabelValues("error").Inc()
				o.l.Error().Err(err).Str("topic", topic).Msg("could not process internal event outbox batch")

				break
			}

			if len(msgs) == 0 {
				break
			}

			prometheus.InternalEventOutboxBatches.WithLabelValues("ok").Inc()
		}
	}
}

type internalEventFlusher struct {
	handler InternalEventHandler
}

// Flush fails the whole batch on any error, so every message is retried; a message that can never be
// processed holds up its shard, the same way a poison message is redelivered forever on the task queue.
func (f *internalEventFlusher) Flush(ctx pgoutbox.FlushContext, msgs []*outboxsqlc.Message) error {
	byTenant := make(map[uuid.UUID][]*InternalTaskEvent)
	tenantOrder := make([]uuid.UUID, 0)

	for _, msg := range msgs {
		var m internalEventOutboxMessage

		if err := json.Unmarshal(msg.Payload, &m); err != nil {
			return fmt.Errorf("could not decode internal event outbox message %d: %w", msg.ID, err)
		}

		if _, ok := byTenant[m.TenantID]; !ok {
			tenantOrder = append(tenantOrder, m.TenantID)
		}

		for i := range m.Events {
			byTenant[m.TenantID] = append(byTenant[m.TenantID], &m.Events[i])
		}

		prometheus.InternalEventOutboxLag.Observe(time.Since(msg.InsertedAt.Time).Seconds())
	}

	for _, tenantId := range tenantOrder {
		if err := f.handler(ctx, tenantId, byTenant[tenantId]); err != nil {
			return fmt.Errorf("could not process internal events for tenant %s: %w", tenantId, err)
		}
	}

	return nil
}
