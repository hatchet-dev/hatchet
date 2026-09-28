package streams

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
)

// topicPollerKey identifies one durable-stream topic's shared tail poller.
type topicPollerKey struct {
	tenantId  uuid.UUID
	namespace string
	topic     string
}

// topicListener is one Subscribe RPC's registration with a shared topicPoller.
// send delivers a message to that RPC's own stream; cancel is that RPC's own
// context.CancelFunc, called if send ever fails so the RPC goroutine (blocked
// on <-ctx.Done()) notices and tears itself down, which in turn unregisters
// this listener.
type topicListener struct {
	send   func(*contracts.StreamMessage) error
	cancel context.CancelFunc
}

// topicPoller runs a single tail loop (ticker + pubsub wake + Postgres poll)
// for one (tenant, namespace, topic), shared by every Subscribe RPC currently
// tailing it.
type topicPoller struct {
	streams           v1.StreamsRepository
	pubsub            msgqueue.PubSub
	l                 *zerolog.Logger
	key               topicPollerKey
	tailPollInterval  time.Duration
	idleHangupTimeout time.Duration

	mu             sync.Mutex
	running        bool
	cursor         v1.StreamCursor
	listeners      map[int]*topicListener
	nextID         int
	cancel         context.CancelFunc
	lastActivityAt time.Time
}

func newTopicPoller(streams v1.StreamsRepository, pubsub msgqueue.PubSub, l *zerolog.Logger, key topicPollerKey, tailPollInterval, idleHangupTimeout time.Duration) *topicPoller {
	return &topicPoller{
		streams:           streams,
		pubsub:            pubsub,
		l:                 l,
		key:               key,
		tailPollInterval:  tailPollInterval,
		idleHangupTimeout: idleHangupTimeout,
		listeners:         make(map[int]*topicListener),
	}
}

func (p *topicPoller) join(ctx context.Context, startCursor v1.StreamCursor, listener *topicListener) (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.running {
		p.running = true
		p.cursor = startCursor
		p.lastActivityAt = time.Now()
		p.startLocked()
	} else if startCursor.After(p.cursor) {
		if err := p.pollLocked(ctx); err != nil {
			return 0, err
		}
	}

	p.nextID++
	id := p.nextID
	p.listeners[id] = listener

	return id, nil
}

func (p *topicPoller) leave(id int) (idle bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	delete(p.listeners, id)

	if len(p.listeners) == 0 {
		p.running = false

		if p.cancel != nil {
			p.cancel()
			p.cancel = nil
		}

		return true
	}

	return false
}

// startLocked launches the background tail loop.
func (p *topicPoller) startLocked() {
	ctx, cancel := context.WithCancel(context.Background())
	p.cancel = cancel

	wake := make(chan struct{}, 1)

	unsub, err := p.pubsub.Sub(msgqueue.StreamTopic(p.key.tenantId, p.key.namespace, p.key.topic), func(*msgqueue.Message) error {
		select {
		case wake <- struct{}{}:
		default:
		}

		return nil
	})

	if err != nil {
		p.l.Warn().Ctx(ctx).Err(err).Msg("could not subscribe to stream topic wake channel; falling back to polling only")
		unsub = func() error { return nil }
	}

	go func() {
		defer func() {
			_ = unsub()
		}()

		ticker := time.NewTicker(p.tailPollInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			case <-wake:
			}

			p.mu.Lock()
			err := p.pollLocked(ctx)
			if err == nil {
				p.hangUpIfIdleLocked()
			}
			p.mu.Unlock()

			if err != nil {
				p.l.Error().Ctx(ctx).Err(err).Msg("stream topic poll failed")
			}

			// hangUpIfIdleLocked's p.cancel() closes ctx, so an idle stop is
			// picked up by the next loop iteration's ctx.Done() case.
		}
	}()
}

// hangUpIfIdleLocked sends a final hangup=true message (carrying the current
// cursor) to every listener and stops the poller once idleHangupTimeout has
// passed with no new messages, rather than polling an inactive topic
// forever. Must be called with p.mu held.
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

	for id, l := range p.listeners {
		if sendErr := l.send(hangup); sendErr != nil {
			p.l.Debug().Err(sendErr).Msg("stream listener send failed during idle hangup")
		}

		delete(p.listeners, id)
		l.cancel()
	}

	p.running = false

	if p.cancel != nil {
		p.cancel()
		p.cancel = nil
	}
}

// pollLocked runs the keyset query from the poller's current cursor and fans
// each batch out to every currently-registered listener, in order. Must be
// called with p.mu held.
func (p *topicPoller) pollLocked(ctx context.Context) error {
	for {
		msgs, err := p.streams.ListMessagesAfterCursor(ctx, p.key.tenantId, v1.ListStreamMessagesOpts{
			Namespace: p.key.namespace,
			Topic:     p.key.topic,
			Cursor:    p.cursor,
			Limit:     subscribeCatchUpBatchSize,
		})

		if err != nil {
			return err
		}

		if len(msgs) > 0 {
			p.lastActivityAt = time.Now()
		}

		entries := make([]*contracts.StreamEntry, 0, len(msgs))

		for _, m := range msgs {
			next := v1.StreamCursor{Namespace: p.key.namespace, Topic: p.key.topic, CreatedAt: m.InsertedAt.Time, ID: m.ID}

			encodedCursor, err := v1.EncodeStreamCursor(next)

			if err != nil {
				return err
			}

			entries = append(entries, &contracts.StreamEntry{
				Payload:   m.Payload,
				Cursor:    encodedCursor,
				CreatedAt: timestamppb.New(m.InsertedAt.Time),
			})

			p.cursor = next
		}

		for _, chunk := range chunkStreamEntries(entries) {
			out := &contracts.StreamMessage{Entries: chunk}

			for id, l := range p.listeners {
				if sendErr := l.send(out); sendErr != nil {
					p.l.Debug().Ctx(ctx).Err(sendErr).Msg("removing stream listener after send failure")
					delete(p.listeners, id)
					l.cancel()
				}
			}
		}

		if len(msgs) < subscribeCatchUpBatchSize {
			return nil
		}
	}
}

// topicPollerRegistry is the process-wide set of active topicPollers, one per
// (tenant, namespace, topic) currently being tailed by at least one Subscribe
// call.
type topicPollerRegistry struct {
	mu                sync.Mutex
	pollers           map[topicPollerKey]*topicPoller
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

// Join registers listener as a tail-subscriber of key's shared poller,
// creating the poller if this is the first listener for it, and returns a
// function that unregisters it.
func (r *topicPollerRegistry) Join(ctx context.Context, key topicPollerKey, startCursor v1.StreamCursor, listener *topicListener) (unregister func(), err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	p, ok := r.pollers[key]

	if !ok {
		p = newTopicPoller(r.streams, r.pubsub, r.l, key, r.tailPollInterval, r.idleHangupTimeout)
		r.pollers[key] = p
	}

	id, err := p.join(ctx, startCursor, listener)

	if err != nil {
		return nil, err
	}

	return func() {
		r.mu.Lock()
		defer r.mu.Unlock()

		if p.leave(id) {
			delete(r.pollers, key)
		}
	}, nil
}
