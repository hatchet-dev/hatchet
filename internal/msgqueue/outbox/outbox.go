package outbox

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/hatchet-dev/pgoutbox"
	outboxsqlc "github.com/hatchet-dev/pgoutbox/sqlc"
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/internal/queueutils"
	"github.com/hatchet-dev/hatchet/pkg/logger"
)

const relayFlushConcurrency = 10

// Topic returns the pgoutbox topic for a queue relayed through the outbox. The "mq."
// prefix distinguishes message queue topics from other topics (e.g. concurrency
// strategies) which share the outbox tables.
func Topic(q msgqueue.Queue) string {
	return "mq." + q.Name()
}

// Relay drains outbox-staged messages and republishes them to their message queue.
// Producers stage messages under a queue's topic (see Topic) inside their database
// transactions; after commit, the relay delivers them through the delegate message
// queue, so consumers are unaffected by the outbox. Every replica runs a relay —
// concurrent relays drain disjoint batches.
type Relay struct {
	delegate msgqueue.MessageQueue
	pubsub   msgqueue.PubSub
	outbox   pgoutbox.Outbox

	l *zerolog.Logger

	queues []msgqueue.Queue

	subscribeBatchSize int
	pollInterval       time.Duration
}

type Opt func(*Relay)

// WithQueue relays the given queue's outbox topic.
func WithQueue(q msgqueue.Queue) Opt {
	return func(r *Relay) {
		r.queues = append(r.queues, q)
	}
}

func WithSubscribeConfig(batchSize int, pollInterval time.Duration) Opt {
	return func(r *Relay) {
		if batchSize > 0 {
			r.subscribeBatchSize = batchSize
		}

		if pollInterval > 0 {
			r.pollInterval = pollInterval
		}
	}
}

func WithLogger(l *zerolog.Logger) Opt {
	return func(r *Relay) {
		r.l = l
	}
}

// WithPubSub attaches the best-effort pub/sub: relayed tenant-scoped messages
// which the dispatcher's streams consume are mirrored to the tenant topic.
func WithPubSub(ps msgqueue.PubSub) Opt {
	return func(r *Relay) {
		r.pubsub = ps
	}
}

func NewRelay(delegate msgqueue.MessageQueue, ob pgoutbox.Outbox, fs ...Opt) *Relay {
	l := logger.NewDefaultLogger("outbox-relay")

	r := &Relay{
		delegate:           delegate,
		outbox:             ob,
		subscribeBatchSize: 100,
		pollInterval:       5 * time.Second,
		l:                  &l,
	}

	for _, f := range fs {
		f(r)
	}

	return r
}

// Start registers a relay flusher for each queue and subscribes to its topic. The
// returned cleanup cancels the subscriptions and waits for them to exit.
func (r *Relay) Start() (func() error, error) {
	ctx, cancel := context.WithCancel(context.Background())

	wg := sync.WaitGroup{}

	for _, q := range r.queues {
		topic := Topic(q)

		r.outbox.AddFlusher(topic, &relayFlusher{
			delegate: r.delegate,
			pubsub:   r.pubsub,
			queue:    q,
			l:        r.l,
		})

		wg.Add(1)

		go func() {
			defer wg.Done()

			retries := 0

			for ctx.Err() == nil {
				err := r.outbox.Subscribe(
					ctx,
					topic,
					pgoutbox.WithPollInterval(r.pollInterval),
					pgoutbox.WithProcessOpts(pgoutbox.WithBatchSize(r.subscribeBatchSize)),
				)

				if ctx.Err() != nil {
					return
				}

				if err != nil {
					r.l.Error().Err(err).Msgf("outbox relay subscribe exited for topic %s, retrying", topic)
				}

				retries++
				queueutils.SleepWithExponentialBackoff(100*time.Millisecond, 5*time.Second, retries)
			}
		}()
	}

	return func() error {
		cancel()
		wg.Wait()
		return nil
	}, nil
}

// relayFlusher republishes staged messages to the delegate message queue. It must not
// use the flush transaction (ctx.Tx()): the delegate publish is external I/O, and the
// delegate may itself open transactions on the same pool.
type relayFlusher struct {
	delegate msgqueue.MessageQueue
	pubsub   msgqueue.PubSub
	queue    msgqueue.Queue
	l        *zerolog.Logger
}

func (f *relayFlusher) Flush(ctx pgoutbox.FlushContext, msgs []*outboxsqlc.Message) error {
	eg := errgroup.Group{}
	eg.SetLimit(relayFlushConcurrency)

	for _, m := range msgs {
		eg.Go(func() error {
			msg := &msgqueue.Message{}

			if err := json.Unmarshal(m.Payload, msg); err != nil {
				// drop unparseable messages so they don't wedge the topic; this matches
				// the rabbitmq consumer, which rejects unparseable bodies without requeue
				f.l.Error().Err(err).Int64("outbox_message_id", m.ID).Msgf("dropping unparseable outbox message for queue %s", f.queue.Name())
				return nil
			}

			if f.pubsub == nil {
				return f.delegate.SendMessage(ctx, f.queue, msg)
			}

			// PubTenantMessage sends durably and mirrors stream-relevant messages
			// (e.g. created-task) to the tenant topic for the dispatcher's streams
			return msgqueue.PubTenantMessage(ctx, f.l, f.delegate, f.pubsub, f.queue, msg)
		})
	}

	return eg.Wait()
}
