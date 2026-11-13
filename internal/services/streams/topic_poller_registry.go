package streams

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/telemetry"
)

// topicPollerRegistry holds each topic's live poller.
type topicPollerRegistry struct {
	mu      sync.Mutex
	pollers map[topicPollerKey]*topicPoller

	// wakeTargets mirrors pollers so the wake handler never waits on mu
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

	for {
		p := r.livePoller(key)

		// outside r.mu, so a poller busy with a slow read can't stall Joins on other topics
		id, err := p.join(ctx, startCursor, listener)

		if errors.Is(err, errPollerRetired) {
			continue
		}

		if err != nil {
			return nil, err
		}

		var once sync.Once

		return func() { once.Do(func() { p.leave(id) }) }, nil
	}
}

// livePoller returns key's poller, starting a new one in place of one that retired.
func (r *topicPollerRegistry) livePoller(key topicPollerKey) *topicPoller {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.subscribeWakesLocked()

	if p, ok := r.pollers[key]; ok && !p.retired.Load() {
		return p
	}

	p := newTopicPoller(r.streams, r.l, key, r.tailPollInterval, r.idleHangupTimeout, r.remove)
	r.pollers[key] = p
	r.wakeTargets.Store(key, p)

	return p
}

// remove runs under the retiring poller's mu; the registry never takes a poller's mu, so that order can't deadlock.
func (r *topicPollerRegistry) remove(p *topicPoller) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.pollers[p.key] == p {
		delete(r.pollers, p.key)
		r.wakeTargets.Delete(p.key)
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
