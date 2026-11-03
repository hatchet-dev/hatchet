package streams

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sync"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/telemetry"
)

// rows per page read
const subscribeCatchUpBatchSize = 500

type topicPollerKey struct {
	tenantId  uuid.UUID
	namespace string
	topic     string
}

// topicListener is one Subscribe RPC. cancel ends the RPC when a send fails,
// which unregisters the listener.
type topicListener struct {
	send   func(*contracts.StreamMessage) error
	cancel context.CancelFunc
}

// listenerQueueSize is how many frames a listener can fall behind before it's
// dropped; its client resubscribes from the last cursor it received.
const listenerQueueSize = 64

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

// run cancels the RPC once the queue is closed and drained, so a final hangup is written before the stream ends.
func (q *listenerQueue) run() {
	defer q.listener.cancel()

	for frame := range q.frames {
		if err := q.listener.send(frame); err != nil {
			q.listener.cancel()

			for range q.frames {
				continue
			}

			return
		}
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

var errListenerFellBehind = errors.New("stream listener fell too far behind")

// topicPoller is one topic's tail loop, shared by every Subscribe RPC on it.
type topicPoller struct {
	streams           v1.StreamsRepository
	l                 *zerolog.Logger
	key               topicPollerKey
	tailPollInterval  time.Duration
	idleHangupTimeout time.Duration

	mu        sync.Mutex
	listeners map[int]*listenerQueue
	nextID    int

	// one run's state, reset by startLocked
	running bool
	cancel  context.CancelFunc
	cursor  v1.StreamCursor
	// every message up to this reserved offset of the topicCreatedAt incarnation has been fanned out
	checkedOffset  int64
	topicCreatedAt time.Time
	lastActivityAt time.Time

	wake chan struct{}

	// Join calls and registered listeners; the registry removes the poller at zero. Guarded by the registry's mu.
	members int
}

func newTopicPoller(streams v1.StreamsRepository, l *zerolog.Logger, key topicPollerKey, tailPollInterval, idleHangupTimeout time.Duration) *topicPoller {
	return &topicPoller{
		streams:           streams,
		l:                 l,
		key:               key,
		tailPollInterval:  tailPollInterval,
		idleHangupTimeout: idleHangupTimeout,
		listeners:         make(map[int]*listenerQueue),
		wake:              make(chan struct{}, 1),
	}
}

// a wake already pending covers this one
func (p *topicPoller) wakeUp() {
	select {
	case p.wake <- struct{}{}:
	default:
	}
}

func (p *topicPoller) join(ctx context.Context, startCursor v1.StreamCursor, listener *topicListener) (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	q := startListenerQueue(listener)

	if err := p.attachLocked(ctx, startCursor, q); err != nil {
		close(q.frames)
		return 0, err
	}

	p.nextID++
	id := p.nextID
	p.listeners[id] = q

	return id, nil
}

func (p *topicPoller) attachLocked(ctx context.Context, startCursor v1.StreamCursor, q *listenerQueue) error {
	if !p.running {
		p.startLocked(startCursor)
	} else {
		if startCursor.After(p.cursor) {
			if _, err := p.pollLocked(ctx); err != nil {
				return err
			}
		}

		// rows up to p.cursor were already fanned out to earlier listeners only
		if p.cursor.After(startCursor) {
			return p.catchUpLocked(ctx, startCursor, q)
		}
	}

	return nil
}

func (p *topicPoller) leave(id int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.removeListenerLocked(id)

	if len(p.listeners) == 0 {
		p.stopLocked()
	}
}

// removeListenerLocked lets the listener's queue drain, then ends its RPC.
func (p *topicPoller) removeListenerLocked(id int) {
	if q, ok := p.listeners[id]; ok {
		delete(p.listeners, id)
		close(q.frames)
	}
}

// startLocked begins a run from cursor, discarding everything an earlier run cached. A no-op while running.
func (p *topicPoller) startLocked(cursor v1.StreamCursor) {
	if p.running {
		return
	}

	ctx, cancel := context.WithCancel(context.Background())

	p.running = true
	p.cancel = cancel
	p.cursor = cursor
	p.checkedOffset = 0
	p.topicCreatedAt = time.Time{}
	p.lastActivityAt = time.Now()

	go p.run(ctx)
}

// stopLocked ends the current run. A no-op once stopped.
func (p *topicPoller) stopLocked() {
	if !p.running {
		return
	}

	p.running = false
	p.cancel()
	p.cancel = nil
}

func (p *topicPoller) run(ctx context.Context) {
	ticker := time.NewTicker(p.tailPollInterval)
	defer ticker.Stop()

	for {
		trigger := "tick"

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		case <-p.wake:
			trigger = "wake"
		}

		startedAt := time.Now()

		p.mu.Lock()

		// the run may have stopped, and another started, while this tick waited for the lock
		if ctx.Err() != nil {
			p.mu.Unlock()
			return
		}

		listeners := len(p.listeners)
		stats, err := p.pollIfPublishedLocked(ctx)
		if err == nil {
			p.hangUpIfIdleLocked()
		}
		p.mu.Unlock()

		// most polls find nothing, so only ones that delivered or failed are recorded
		if err != nil || stats.entries > 0 {
			p.recordPoll(startedAt, trigger, listeners, stats, err)
		}

		if err != nil {
			p.l.Error().Ctx(ctx).Err(err).Msg("stream topic poll failed")
		}
	}
}

// hangUpIfIdleLocked stops the poller once its topic has been quiet for
// idleHangupTimeout, sending each listener the cursor to resume from.
func (p *topicPoller) hangUpIfIdleLocked() {
	if p.idleHangupTimeout <= 0 || len(p.listeners) == 0 {
		return
	}

	if time.Since(p.lastActivityAt) < p.idleHangupTimeout {
		return
	}

	encodedCursor, err := v1.EncodeStreamCursor(p.cursor)

	if err != nil {
		p.l.Error().Err(err).Msg("could not encode cursor for stream idle hangup")
		return
	}

	hangup := &contracts.StreamMessage{Cursor: encodedCursor, Hangup: true}

	for id, q := range p.listeners {
		if !q.enqueue(hangup) {
			q.listener.cancel()
		}

		p.removeListenerLocked(id)
	}

	p.stopLocked()
}

// sendStats counts what a send callback delivered, once per frame however many listeners it fans out to.
type sendStats struct {
	entries      int
	payloadBytes int
}

func (s *sendStats) counting(send func(*contracts.StreamMessage) error) func(*contracts.StreamMessage) error {
	return func(out *contracts.StreamMessage) error {
		s.entries += len(out.Entries)

		for _, e := range out.Entries {
			s.payloadBytes += len(e.Payload)
		}

		return send(out)
	}
}

func (p *topicPoller) recordPoll(startedAt time.Time, trigger string, listeners int, stats sendStats, err error) {
	_, span := telemetry.NewSpanAt(context.Background(), "streams.topic-poller.poll", startedAt)
	defer span.End()

	telemetry.WithAttributes(span,
		telemetry.AttributeKV{Key: "tenant.id", Value: p.key.tenantId},
		telemetry.AttributeKV{Key: "stream.namespace", Value: p.key.namespace},
		telemetry.AttributeKV{Key: "stream.topic", Value: p.key.topic},
		telemetry.AttributeKV{Key: "trigger", Value: trigger},
		telemetry.AttributeKV{Key: "listeners", Value: listeners},
		telemetry.AttributeKV{Key: "entries", Value: stats.entries},
		telemetry.AttributeKV{Key: "stream.payload_bytes", Value: stats.payloadBytes},
	)

	recordSpanError(span, err)
}

// pollIfPublishedLocked skips the message read, which spans every retained partition, while nothing was published.
func (p *topicPoller) pollIfPublishedLocked(ctx context.Context) (sendStats, error) {
	info, err := p.streams.GetTopicInfo(ctx, p.key.tenantId, p.key.namespace, p.key.topic)

	if err != nil {
		return sendStats{}, err
	}

	// the idle sweep deletes topics, and a publish recreating one restarts its offsets at 1
	if !info.CreatedAt.Equal(p.topicCreatedAt) {
		if !p.topicCreatedAt.IsZero() {
			p.cursor = v1.StreamCursor{Namespace: p.key.namespace, Topic: p.key.topic}
		}

		p.topicCreatedAt = info.CreatedAt
		p.checkedOffset = 0
	}

	// compared to offsets, not the cursor, since a rejected publish reserves one it never stores
	if info.LastOffset <= p.checkedOffset {
		return sendStats{}, nil
	}

	stats, err := p.pollLocked(ctx)

	if err == nil {
		p.checkedOffset = info.LastOffset
	}

	return stats, err
}

// pollLocked sends every listener the rows after the poller's cursor.
func (p *topicPoller) pollLocked(ctx context.Context) (sendStats, error) {
	var stats sendStats

	last, err := sendRange(ctx, p.streams, p.key, p.cursor, math.MaxInt64, stats.counting(func(out *contracts.StreamMessage) error {
		for id, q := range p.listeners {
			if !q.enqueue(out) {
				p.l.Warn().Ctx(ctx).Msg("dropping stream listener that fell too far behind; it resumes from its cursor")
				q.listener.cancel()
				p.removeListenerLocked(id)
			}
		}

		return nil
	}))

	if last.After(p.cursor) {
		p.lastActivityAt = time.Now()
		p.cursor = last
	}

	return stats, err
}

// catchUpLocked sends listener the rows in (from, p.cursor] that the poller
// fanned out before it joined.
func (p *topicPoller) catchUpLocked(ctx context.Context, from v1.StreamCursor, q *listenerQueue) error {
	_, err := sendRange(ctx, p.streams, p.key, from, p.cursor.ID, func(out *contracts.StreamMessage) error {
		if !q.enqueue(out) {
			return connect.NewError(connect.CodeUnavailable, errListenerFellBehind)
		}

		return nil
	})

	return err
}

// sendRange sends the rows in (from, maxID]. It returns the cursor of the last
// row sent (from if none), even alongside an error.
func sendRange(ctx context.Context, repo v1.StreamsRepository, key topicPollerKey, from v1.StreamCursor, maxID int64, send func(*contracts.StreamMessage) error) (v1.StreamCursor, error) {
	for {
		msgs, err := repo.ListMessagesAfterCursor(ctx, key.tenantId, v1.ListStreamMessagesOpts{
			Namespace: key.namespace,
			Topic:     key.topic,
			Cursor:    from,
			Limit:     subscribeCatchUpBatchSize,
		})

		if err != nil {
			return from, err
		}

		entries := make([]*contracts.StreamEntry, 0, len(msgs))
		pageEnd := from
		pageBytes := 0

		for _, m := range msgs {
			pageBytes += len(m.Payload)
		}

		// a page cut short by the byte budget can hold fewer rows than the limit
		done := len(msgs) < subscribeCatchUpBatchSize && pageBytes < v1.MaxListStreamMessagesBytes

		for _, m := range msgs {
			if m.ID > maxID {
				done = true
				break
			}

			next := v1.StreamCursor{Namespace: key.namespace, Topic: key.topic, CreatedAt: m.InsertedAt.Time, ID: m.ID}

			encodedCursor, err := v1.EncodeStreamCursor(next)

			if err != nil {
				return from, err
			}

			entry := &contracts.StreamEntry{
				Payload:   m.Payload,
				Cursor:    encodedCursor,
				CreatedAt: timestamppb.New(m.InsertedAt.Time),
			}

			if m.PayloadID != nil {
				entry.PayloadRef, err = v1.EncodeStreamPayloadRef(v1.StreamPayloadRef{ID: *m.PayloadID, CreatedAt: m.PayloadInsertedAt.Time})

				if err != nil {
					return from, err
				}
			}

			entries = append(entries, entry)

			pageEnd = next
		}

		for _, chunk := range chunkStreamEntries(entries) {
			if err := send(&contracts.StreamMessage{Entries: chunk}); err != nil {
				return from, err
			}
		}

		from = pageEnd

		if done {
			return from, nil
		}
	}
}

// topicPollerRegistry holds one topicPoller per topic with a listener.
type topicPollerRegistry struct {
	mu      sync.Mutex
	pollers map[topicPollerKey]*topicPoller

	// wakeTargets mirrors pollers so the wake handler never waits on mu,
	// which Join holds across database reads
	wakeTargets sync.Map // topicPollerKey -> *topicPoller

	// one wake subscription for every poller; nil until the first Join. Guarded by mu.
	unsubWake func() error

	streams           v1.StreamsRepository
	pubsub            msgqueue.PubSub
	l                 *zerolog.Logger
	tailPollInterval  time.Duration
	idleHangupTimeout time.Duration
}

func newTopicPollerRegistry(streams v1.StreamsRepository, pubsub msgqueue.PubSub, l *zerolog.Logger, tailPollInterval, idleHangupTimeout time.Duration) *topicPollerRegistry {
	return &topicPollerRegistry{
		pollers:           make(map[topicPollerKey]*topicPoller),
		streams:           streams,
		pubsub:            pubsub,
		l:                 l,
		tailPollInterval:  tailPollInterval,
		idleHangupTimeout: idleHangupTimeout,
	}
}

// Join sends listener every message after startCursor, then attaches it to
// key's poller. The caller checks startCursor is retained.
func (r *topicPollerRegistry) Join(ctx context.Context, key topicPollerKey, startCursor v1.StreamCursor, listener *topicListener) (unregister func(), err error) {
	ctx, span := telemetry.NewSpan(ctx, "streams.topic-poller.join")
	defer func() {
		recordSpanError(span, err)
		span.End()
	}()

	var replayed sendStats

	// replayed before taking any lock so a long backlog can't stall live
	// delivery to the topic's other listeners; join backfills the remainder
	startCursor, err = sendRange(ctx, r.streams, key, startCursor, math.MaxInt64, replayed.counting(listener.send))

	telemetry.WithAttributes(span,
		telemetry.AttributeKV{Key: "replayed_entries", Value: replayed.entries},
		telemetry.AttributeKV{Key: "replayed_payload_bytes", Value: replayed.payloadBytes},
	)

	if err != nil {
		return nil, err
	}

	p := r.acquire(key)

	// outside r.mu, so a poller busy with a slow read can't stall Joins on other topics
	id, err := p.join(ctx, startCursor, listener)

	if err != nil {
		r.release(key, p)
		return nil, err
	}

	var once sync.Once

	return func() {
		once.Do(func() {
			p.leave(id)
			r.release(key, p)
		})
	}, nil
}

// acquire counts a member on key's poller, creating it if needed, so it outlives a Join in progress.
func (r *topicPollerRegistry) acquire(key topicPollerKey) *topicPoller {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.subscribeWakesLocked()

	p, ok := r.pollers[key]

	if !ok {
		p = newTopicPoller(r.streams, r.l, key, r.tailPollInterval, r.idleHangupTimeout)
		r.pollers[key] = p
		r.wakeTargets.Store(key, p)
	}

	p.members++

	return p
}

func (r *topicPollerRegistry) release(key topicPollerKey, p *topicPoller) {
	r.mu.Lock()
	defer r.mu.Unlock()

	p.members--

	if p.members == 0 && r.pollers[key] == p {
		delete(r.pollers, key)
		r.wakeTargets.Delete(key)
	}
}

// on failure pollers still tick, and the next Join retries
func (r *topicPollerRegistry) subscribeWakesLocked() {
	if r.unsubWake != nil {
		return
	}

	unsub, err := r.pubsub.Sub(msgqueue.StreamWakeTopic(), r.handleWake)

	if err != nil {
		r.l.Warn().Err(err).Msg("could not subscribe to stream wakes; tailing topics will poll only")
		return
	}

	r.unsubWake = unsub
}

func (r *topicPollerRegistry) handleWake(msg *msgqueue.Message) error {
	for _, payload := range msg.Payloads {
		var wake msgqueue.StreamWake

		if err := json.Unmarshal(payload, &wake); err != nil {
			return fmt.Errorf("could not decode stream wake: %w", err)
		}

		if p, ok := r.wakeTargets.Load(topicPollerKey{tenantId: msg.TenantID, namespace: wake.Namespace, topic: wake.Topic}); ok {
			p.(*topicPoller).wakeUp()
		}
	}

	return nil
}

func (r *topicPollerRegistry) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.unsubWake == nil {
		return nil
	}

	err := r.unsubWake()
	r.unsubWake = nil

	return err
}
