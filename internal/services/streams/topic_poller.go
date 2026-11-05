package streams

import (
	"context"
	"errors"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"google.golang.org/protobuf/types/known/timestamppb"

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

// errPollerRetired sends a Join back to the registry for the topic's live poller.
var errPollerRetired = errors.New("topic poller retired")

// topicPoller is one topic's tail loop, shared by every Subscribe RPC on it. It
// starts with its first listener and retires for good once it has none, idles
// out, or the topic's offsets restart; the registry starts a new one for the next Join.
type topicPoller struct {
	streams           v1.StreamsRepository
	l                 *zerolog.Logger
	key               topicPollerKey
	tailPollInterval  time.Duration
	idleHangupTimeout time.Duration
	onRetire          func(*topicPoller)

	wake chan struct{}

	// set under mu, read without it by the registry
	retired atomic.Bool

	mu        sync.Mutex
	listeners map[int]*listenerQueue
	nextID    int
	cancel    context.CancelFunc
	// every listener has received everything up to it
	cursor v1.StreamCursor
	// every message up to this reserved offset has been fanned out
	checkedOffset  int64
	lastActivityAt time.Time
}

func newTopicPoller(streams v1.StreamsRepository, l *zerolog.Logger, key topicPollerKey, tailPollInterval, idleHangupTimeout time.Duration, onRetire func(*topicPoller)) *topicPoller {
	return &topicPoller{
		streams:           streams,
		l:                 l,
		key:               key,
		tailPollInterval:  tailPollInterval,
		idleHangupTimeout: idleHangupTimeout,
		onRetire:          onRetire,
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

// join attaches listener at cursor, where its replay ended.
func (p *topicPoller) join(ctx context.Context, cursor v1.StreamCursor, listener *topicListener) (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.retired.Load() {
		return 0, errPollerRetired
	}

	q := startListenerQueue(listener)

	if p.cancel == nil {
		p.startLocked(cursor)
	} else if err := p.catchUpLocked(ctx, cursor, q); err != nil {
		close(q.frames)
		return 0, err
	}

	p.nextID++
	id := p.nextID
	p.listeners[id] = q

	return id, nil
}

func (p *topicPoller) leave(id int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.removeListenerLocked(id)

	if len(p.listeners) == 0 {
		p.retireLocked(nil)
	}
}

// removeListenerLocked lets the listener's queue drain, then ends its RPC.
func (p *topicPoller) removeListenerLocked(id int) {
	if q, ok := p.listeners[id]; ok {
		delete(p.listeners, id)
		close(q.frames)
	}
}

func (p *topicPoller) startLocked(cursor v1.StreamCursor) {
	ctx, cancel := context.WithCancel(context.Background())

	p.cancel = cancel
	p.cursor = cursor
	p.lastActivityAt = time.Now()

	go p.run(ctx)
}

// retireLocked ends every listener's RPC and stops the poller for good. With
// resumeAt, each gets a hangup, which ends the client's subscription; without it,
// the RPC just ends, and the client resubscribes from the last cursor it received.
func (p *topicPoller) retireLocked(resumeAt *v1.StreamCursor) {
	if p.retired.Load() {
		return
	}

	var hangupErr error

	if resumeAt != nil {
		var encodedCursor string

		if encodedCursor, hangupErr = v1.EncodeStreamCursor(*resumeAt); hangupErr != nil {
			// without a cursor to send, clients resume from the last one they received
			p.l.Error().Err(hangupErr).Msg("could not encode cursor for stream hangup")
		} else {
			p.fanOutLocked(&contracts.StreamMessage{Cursor: encodedCursor, Hangup: true})
		}
	}

	for id, q := range p.listeners {
		if hangupErr != nil {
			q.listener.cancel()
		}

		p.removeListenerLocked(id)
	}

	p.retired.Store(true)

	if p.cancel != nil {
		p.cancel()
	}

	p.onRetire(p)
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

		// retired while this tick waited for the lock
		if p.retired.Load() {
			p.mu.Unlock()
			return
		}

		listeners := len(p.listeners)
		stats, err := p.pollIfPublishedLocked(ctx)
		if err == nil && p.idleHangupTimeout > 0 && time.Since(p.lastActivityAt) >= p.idleHangupTimeout {
			p.retireLocked(&p.cursor)
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

// pollIfPublishedLocked skips the message read, which spans every retained partition, while nothing was published.
func (p *topicPoller) pollIfPublishedLocked(ctx context.Context) (sendStats, error) {
	info, err := p.streams.GetTopicInfo(ctx, p.key.tenantId, p.key.namespace, p.key.topic)

	if err != nil {
		return sendStats{}, err
	}

	// offsets only go back when the idle sweep deleted the topic and a publish recreated
	// it; ending the RPCs makes each client resume from its own last cursor on a new poller
	if info.LastOffset < p.checkedOffset {
		p.retireLocked(nil)
		return sendStats{}, nil
	}

	// compared to offsets, not the cursor, since a rejected publish reserves one it never stores
	if info.LastOffset == p.checkedOffset {
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
		p.fanOutLocked(out)
		return nil
	}))

	if last.After(p.cursor) {
		p.lastActivityAt = time.Now()
		p.cursor = last
	}

	return stats, err
}

// fanOutLocked queues frame for every listener, dropping any too far behind to
// take it; their clients resume from the last cursor they received.
func (p *topicPoller) fanOutLocked(frame *contracts.StreamMessage) {
	for id, q := range p.listeners {
		if !q.enqueue(frame) {
			p.l.Warn().Msg("dropping stream listener that fell too far behind")
			q.listener.cancel()
			p.removeListenerLocked(id)
		}
	}
}

// catchUpLocked brings earlier listeners up to where q's replay ended, then sends
// q the rows in (from, p.cursor] that the poller fanned out before it joined.
func (p *topicPoller) catchUpLocked(ctx context.Context, from v1.StreamCursor, q *listenerQueue) error {
	if from.After(p.cursor) {
		if _, err := p.pollLocked(ctx); err != nil {
			return err
		}
	}

	if !p.cursor.After(from) {
		return nil
	}

	_, err := sendRange(ctx, p.streams, p.key, from, p.cursor.ID, func(out *contracts.StreamMessage) error {
		if !q.enqueue(out) {
			return connect.NewError(connect.CodeUnavailable, errListenerFellBehind)
		}

		return nil
	})

	return err
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
