package streams

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/trace"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/telemetry"
)

const (
	publishBatchWorkers = 4

	// a batch stops at whichever limit it reaches first
	publishBatchMaxMessages = 100
	publishBatchMaxBytes    = 16 * 1024 * 1024

	publishCommitTimeout = 30 * time.Second
)

var errPublishBatcherStopped = errors.New("stream publisher is shutting down")

// publishBatcher commits publishes to Postgres. Each worker takes every
// publish already waiting, so publishes batch under load without waiting when idle.
type publishBatcher struct {
	streams v1.StreamsRepository
	pubsub  msgqueue.PubSub
	l       *zerolog.Logger

	pending chan *pendingPublish
	cancel  context.CancelFunc
	wg      sync.WaitGroup
}

// pendingPublish is one request's messages, which always share a commit so a
// producer's batch can't be split across concurrent transactions.
type pendingPublish struct {
	msgs []v1.TenantStreamMessage
	done chan publishResult

	// links the batch commit back to the publish waiting on it
	spanCtx    trace.SpanContext
	enqueuedAt time.Time
}

type publishResult struct {
	res []v1.OrderedStreamMessageResult
	err error

	batchSize       int
	commitStartedAt time.Time
}

func newPublishBatcher(streams v1.StreamsRepository, pubsub msgqueue.PubSub, l *zerolog.Logger, workers int) *publishBatcher {
	ctx, cancel := context.WithCancel(context.Background())

	b := &publishBatcher{
		streams: streams,
		pubsub:  pubsub,
		l:       l,
		pending: make(chan *pendingPublish, workers*publishBatchMaxMessages),
		cancel:  cancel,
	}

	for range workers {
		b.wg.Go(func() { b.work(ctx) })
	}

	return b
}

// publish returns once msgs are committed or rejected, with results in input
// order. If ctx ends first they may still commit, which a retry with the same
// sequences detects.
func (b *publishBatcher) publish(ctx context.Context, msgs ...v1.TenantStreamMessage) ([]v1.OrderedStreamMessageResult, error) {
	ctx, span := telemetry.NewSpan(ctx, "streams.publish-batcher.publish")
	defer span.End()

	p := &pendingPublish{msgs: msgs, done: make(chan publishResult, 1), spanCtx: span.SpanContext(), enqueuedAt: time.Now()}

	select {
	case b.pending <- p:
	case <-ctx.Done():
		recordSpanError(span, ctx.Err())
		return nil, ctx.Err()
	}

	select {
	case r := <-p.done:
		if !r.commitStartedAt.IsZero() {
			telemetry.WithAttributes(span,
				telemetry.AttributeKV{Key: "batch_size", Value: r.batchSize},
				telemetry.AttributeKV{Key: "queue_wait_ms", Value: r.commitStartedAt.Sub(p.enqueuedAt).Milliseconds()},
			)
		}

		recordSpanError(span, r.err)

		return r.res, r.err
	case <-ctx.Done():
		recordSpanError(span, ctx.Err())
		return nil, ctx.Err()
	}
}

func (b *publishBatcher) stop() {
	b.cancel()
	b.wg.Wait()

	for {
		select {
		case p := <-b.pending:
			p.done <- publishResult{err: errPublishBatcherStopped}
		default:
			return
		}
	}
}

func (b *publishBatcher) work(ctx context.Context) {
	for {
		var first *pendingPublish

		select {
		case <-ctx.Done():
			return
		case first = <-b.pending:
		}

		batch := []*pendingPublish{first}
		count, size := len(first.msgs), payloadBytes(first.msgs)

	collect:
		for count < publishBatchMaxMessages && size < publishBatchMaxBytes {
			select {
			case p := <-b.pending:
				batch = append(batch, p)
				count += len(p.msgs)
				size += payloadBytes(p.msgs)
			default:
				break collect
			}
		}

		b.commit(batch)
	}
}

func payloadBytes(msgs []v1.TenantStreamMessage) int {
	size := 0

	for _, m := range msgs {
		size += len(m.Opts.Payload)
	}

	return size
}

func (b *publishBatcher) commit(batch []*pendingPublish) {
	startedAt := time.Now()
	links := make([]trace.Link, 0, len(batch))
	var msgs []v1.TenantStreamMessage

	for _, p := range batch {
		if p.spanCtx.IsValid() {
			links = append(links, trace.Link{SpanContext: p.spanCtx})
		}

		msgs = append(msgs, p.msgs...)
	}

	// not a caller's context: one caller giving up must not abort the others' writes
	ctx, span := telemetry.NewSpanWithLinks(context.Background(), "streams.publish-batcher.commit", links)
	defer span.End()

	telemetry.WithAttributes(span,
		telemetry.AttributeKV{Key: "batch_size", Value: len(msgs)},
		telemetry.AttributeKV{Key: "publishes", Value: len(batch)},
		telemetry.AttributeKV{Key: "stream.payload_bytes", Value: payloadBytes(msgs)},
	)

	ctx, cancel := context.WithTimeout(ctx, publishCommitTimeout)
	defer cancel()

	results, err := b.streams.InsertOrderedStreamMessages(ctx, msgs)
	recordSpanError(span, err)

	offset := 0

	for _, p := range batch {
		if err != nil {
			p.done <- publishResult{err: err, batchSize: len(msgs), commitStartedAt: startedAt}
		} else {
			p.done <- publishResult{res: results[offset : offset+len(p.msgs)], batchSize: len(msgs), commitStartedAt: startedAt}
		}

		offset += len(p.msgs)
	}

	if err == nil {
		b.wake(ctx, msgs, results)
	}
}

// wake is best-effort: a dropped wake only delays delivery until the poller's
// fallback tick.
func (b *publishBatcher) wake(ctx context.Context, msgs []v1.TenantStreamMessage, results []v1.OrderedStreamMessageResult) {
	byTenant := make(map[uuid.UUID]map[msgqueue.StreamWake]struct{})

	for i, m := range msgs {
		if !results[i].Inserted {
			continue
		}

		if byTenant[m.TenantID] == nil {
			byTenant[m.TenantID] = make(map[msgqueue.StreamWake]struct{})
		}

		byTenant[m.TenantID][msgqueue.StreamWake{Namespace: m.Opts.Namespace, Topic: m.Opts.Topic}] = struct{}{}
	}

	for tenantId, topics := range byTenant {
		wakes := make([]msgqueue.StreamWake, 0, len(topics))

		for w := range topics {
			wakes = append(wakes, w)
		}

		if err := msgqueue.PubStreamWakes(ctx, b.pubsub, tenantId, wakes); err != nil {
			b.l.Debug().Ctx(ctx).Err(err).Msg("could not publish stream topic wake")
		}
	}
}
