package repository

import (
	"context"
	"encoding/json"
	"errors"
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

// InternalEventOutboxShards is fixed rather than configurable: every engine drains exactly these topics,
// so a different count on some engines would leave the other topics undrained and split a tenant's
// events across two shards, losing their order.
const InternalEventOutboxShards = 8

// InternalEventHandler processes one tenant's internal events. It must be idempotent: a batch whose
// processing succeeded is processed again if the outbox transaction then fails to commit.
type InternalEventHandler func(ctx context.Context, tenantId uuid.UUID, events []*InternalTaskEvent) error

// InternalEventOutbox writes the internal events a task completion produces to a pgoutbox topic in
// the completion's own transaction, so a crash after the commit can't lose the trigger for the next
// DAG step. Events are spread over several topics because pgoutbox processes one batch per topic at
// a time.
//
// Every process that can apply internal events (one with a task controller or the dispatcher's direct
// writer) drains the topics whether or not it writes to them, so turning writes off, or a rollout where
// only some engines write, never strands events that are already in the outbox.
type InternalEventOutbox struct {
	outbox       pgoutbox.Outbox
	shards       int
	pollInterval time.Duration
	writes       bool
	l            *zerolog.Logger

	// copies of idleDrainInterval and drainStopTimeout, so tests don't wait on them
	idleInterval time.Duration
	stopTimeout  time.Duration

	// kicks wake this process's drain loop for a shard right after a commit, so the common path
	// doesn't wait for the poll; the poll only covers batches whose writer died before draining them.
	kicks []chan struct{}

	flushersOnce sync.Once

	// mu guards the handlers and the loops' lifecycle. The loops run while any handler is registered:
	// in a process with both a task controller and the dispatcher's direct-write handler, either can
	// stop first, and the other's handler keeps the outbox drained.
	mu       sync.Mutex
	handlers []*registeredHandler
	gen      *drainGeneration
	closed   bool

	// inFlight is read-held while a batch runs a handler, so a stopping controller can wait until
	// nothing still runs its handler before tearing down what that handler uses.
	inFlight sync.RWMutex
}

type registeredHandler struct {
	handler InternalEventHandler
}

type internalEventOutboxMessage struct {
	TenantID uuid.UUID           `json:"tenant_id"`
	Events   []InternalTaskEvent `json:"events"`
}

// NewInternalEventOutbox only routes completions through the outbox when writes is true; otherwise they
// keep sending internal events over the task queue and the outbox only drains.
func NewInternalEventOutbox(outbox pgoutbox.Outbox, writes bool, pollInterval time.Duration, l *zerolog.Logger) *InternalEventOutbox {
	shards := InternalEventOutboxShards

	kicks := make([]chan struct{}, shards)

	for i := range kicks {
		kicks[i] = make(chan struct{}, 1)
	}

	return &InternalEventOutbox{
		outbox:       outbox,
		shards:       shards,
		pollInterval: pollInterval,
		writes:       writes,
		l:            l,
		idleInterval: idleDrainInterval,
		stopTimeout:  drainStopTimeout,
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

func (o *InternalEventOutbox) writesEnabled() bool {
	return o != nil && o.writes
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

// idleDrainInterval is how often an engine that doesn't write to the outbox drains it. Only events from
// engines that write, and whose own loops died with them, wait on it, so a slow poll from a single loop
// keeps the outbox's cost near zero while the feature is off.
const idleDrainInterval = 10 * time.Second

// drainStopTimeout bounds how long stopping waits for a batch already running before cancelling it.
// A cancelled batch rolls back and is processed again, but a handler cancelled after committing its
// matches can't redo the signalling that follows them, so letting the batch finish is preferred. It is
// short because it adds to the gRPC stop and the direct writer's drain within the pod's grace period.
const drainStopTimeout = 5 * time.Second

var errNoInternalEventHandler = errors.New("no internal event handler is registered")

// drainGeneration is one run of the loops, from the first Start to the last stop. Each has its own
// WaitGroup so a Start racing the previous run's stop never reuses a WaitGroup that is still waited on.
type drainGeneration struct {
	// stopLoops ends the loops after their current batch; abort cancels that batch too
	stopLoops context.CancelFunc
	abort     context.CancelFunc
	loops     sync.WaitGroup
}

func (g *drainGeneration) stop(timeout time.Duration) {
	g.stopLoops()

	done := make(chan struct{})

	go func() {
		g.loops.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(timeout):
		g.abort()
		<-done
	}

	g.abort()
}

// Start registers handler and drains every shard until the returned stop is called for every
// registered handler. Stop returns once no batch still runs this handler, and once the loops have
// exited if it was the last one, so the caller can then tear down what the handler uses. After Close,
// Start registers nothing and starts no loops, since the outbox's pool is closing.
func (o *InternalEventOutbox) Start(handler InternalEventHandler) (stop func()) {
	topics := make([]string, o.shards)

	for shard := range topics {
		topics[shard] = internalEventOutboxTopic(shard)
	}

	o.flushersOnce.Do(func() {
		for _, topic := range topics {
			o.outbox.AddFlusher(topic, &internalEventFlusher{o: o})
		}
	})

	reg := &registeredHandler{handler: handler}

	o.mu.Lock()
	defer o.mu.Unlock()

	if o.closed {
		return func() {}
	}

	o.handlers = append(o.handlers, reg)

	if o.gen == nil {
		loopCtx, stopLoops := context.WithCancel(context.Background())
		flushCtx, abort := context.WithCancel(context.Background())
		gen := &drainGeneration{stopLoops: stopLoops, abort: abort}
		o.gen = gen

		if o.writes {
			for shard, topic := range topics {
				gen.loops.Go(func() { o.drain(loopCtx, flushCtx, topic, o.kicks[shard]) })
			}
		} else {
			gen.loops.Go(func() { o.drainIdle(loopCtx, flushCtx, topics) })
		}
	}

	var once sync.Once

	return func() { once.Do(func() { o.unregister(reg) }) }
}

func (o *InternalEventOutbox) unregister(reg *registeredHandler) {
	o.mu.Lock()

	for i, h := range o.handlers {
		if h == reg {
			o.handlers = append(o.handlers[:i], o.handlers[i+1:]...)
			break
		}
	}

	var gen *drainGeneration

	if len(o.handlers) == 0 {
		gen = o.gen
		o.gen = nil
	}

	o.mu.Unlock()

	if gen != nil {
		gen.stop(o.stopTimeout)
		return
	}

	o.inFlight.Lock()
	o.inFlight.Unlock() // nolint: staticcheck
}

// Close stops the loops whatever is still registered, and keeps them from starting again. It runs
// before the outbox's pool closes, as a backstop for a registered handler whose owner never stopped
// it: a startup that failed after the direct writer was built, or a direct writer whose drain timed out.
func (o *InternalEventOutbox) Close() {
	if o == nil {
		return
	}

	o.mu.Lock()
	o.closed = true
	o.handlers = nil
	gen := o.gen
	o.gen = nil
	o.mu.Unlock()

	if gen != nil {
		gen.stop(o.stopTimeout)
	}
}

// currentHandler picks the oldest registered handler; every handler applies events the same way.
func (o *InternalEventOutbox) currentHandler() InternalEventHandler {
	o.mu.Lock()
	defer o.mu.Unlock()

	if len(o.handlers) == 0 {
		return nil
	}

	return o.handlers[0].handler
}

func (o *InternalEventOutbox) drainIdle(loopCtx, flushCtx context.Context, topics []string) {
	ticker := time.NewTicker(o.idleInterval)
	defer ticker.Stop()

	for {
		select {
		case <-loopCtx.Done():
			return
		case <-ticker.C:
		}

		for _, topic := range topics {
			o.drainTopic(loopCtx, flushCtx, topic)
		}
	}
}

func (o *InternalEventOutbox) drain(loopCtx, flushCtx context.Context, topic string, kick <-chan struct{}) {
	ticker := time.NewTicker(o.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-loopCtx.Done():
			return
		case <-kick:
		case <-ticker.C:
		}

		o.drainTopic(loopCtx, flushCtx, topic)
	}
}

func (o *InternalEventOutbox) drainTopic(loopCtx, flushCtx context.Context, topic string) {
	for loopCtx.Err() == nil {
		msgs, err := o.outbox.ProcessMessages(flushCtx, topic, pgoutbox.WithBatchSize(internalEventOutboxBatchSize))

		// a batch cancelled by a stop, or picked up just after the last handler left, is shutdown and
		// not an error; it stays in the outbox for the next run
		if err != nil && (flushCtx.Err() != nil || errors.Is(err, errNoInternalEventHandler)) {
			return
		}

		if err != nil {
			// the batch stays in the outbox and is retried on the next kick or poll
			prometheus.InternalEventOutboxBatches.WithLabelValues("error").Inc()
			o.l.Error().Err(err).Str("topic", topic).Msg("could not process internal event outbox batch")

			return
		}

		if len(msgs) == 0 {
			return
		}

		prometheus.InternalEventOutboxBatches.WithLabelValues("ok").Inc()
	}
}

type internalEventFlusher struct {
	o *InternalEventOutbox
}

// Flush fails the whole batch on any error, so every message is retried; a message that can never be
// processed holds up its shard, the same way a poison message is redelivered forever on the task queue.
func (f *internalEventFlusher) Flush(ctx pgoutbox.FlushContext, msgs []*outboxsqlc.Message) error {
	f.o.inFlight.RLock()
	defer f.o.inFlight.RUnlock()

	handler := f.o.currentHandler()

	if handler == nil {
		return errNoInternalEventHandler
	}

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
		if err := handler(ctx, tenantId, byTenant[tenantId]); err != nil {
			return fmt.Errorf("could not process internal events for tenant %s: %w", tenantId, err)
		}
	}

	return nil
}
