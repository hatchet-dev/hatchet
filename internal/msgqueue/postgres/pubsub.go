package postgres

import (
	"context"
	"encoding/json"
	"time"

	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"

	"github.com/hatchet-dev/hatchet/internal/cache"
	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/internal/queueutils"
	"github.com/hatchet-dev/hatchet/pkg/logger"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
)

// fallbackPollBatchSize is the max number of >8KB fallback rows drained per poll.
const fallbackPollBatchSize = 100

// PubSub implements msgqueue.PubSub over Postgres LISTEN/NOTIFY. Topic names
// match the legacy non-durable queue / NOTIFY names, and all traffic
// multiplexes over the single "hatchet_listener" channel, so mixed-version
// fleets interoperate.
//
// INVARIANT: the repo passed to NewPubSub must be built on a dedicated pool
// from the direct (non-pgbouncer) database URL — never the shared repository
// pool. Pub can be called from within durable-write paths, and LISTEN does not
// survive transaction pooling.
type PubSub struct {
	repo v1.MessageQueueRepository
	l    *zerolog.Logger

	// ttlCache dedupes queue-row upserts, which are needed so >8KB payloads can
	// fall back to durable rows
	ttlCache *cache.TTLCache[string, bool]
}

type PubSubOpt func(*PubSubOpts)

type PubSubOpts struct {
	l *zerolog.Logger
}

func defaultPubSubOpts() *PubSubOpts {
	l := logger.NewDefaultLogger("postgres-pubsub")

	return &PubSubOpts{
		l: &l,
	}
}

func WithPubSubLogger(l *zerolog.Logger) PubSubOpt {
	return func(opts *PubSubOpts) {
		opts.l = l
	}
}

// NewPubSub creates a new Postgres-backed PubSub over the given message queue
// repository.
func NewPubSub(repo v1.MessageQueueRepository, fs ...PubSubOpt) (func() error, *PubSub, error) {
	opts := defaultPubSubOpts()

	for _, f := range fs {
		f(opts)
	}

	c := cache.NewTTL[string, bool]()

	p := &PubSub{
		repo:     repo,
		l:        opts.l,
		ttlCache: c,
	}

	return func() error {
		c.Stop()
		return nil
	}, p, nil
}

func (p *PubSub) IsReady() bool {
	return true
}

// Pub publishes a message to the topic via NOTIFY. Payloads whose wrapped
// message exceeds pg_notify's 8KB limit fall back to a short-lived message
// queue row (see MessageQueueRepository.Notify), which subscribers drain with
// a ~1s poll — task-stream-event payloads routinely exceed 8KB and stream
// delivery must not regress.
func (p *PubSub) Pub(ctx context.Context, topic msgqueue.Topic, msg *msgqueue.Message) error {
	// upsert the queue row so >8KB fallback rows have a queue to land on
	err := p.ensureQueue(ctx, topic)

	if err != nil {
		return err
	}

	eg := errgroup.Group{}

	for _, payload := range msg.Payloads {
		msgCp := *msg
		msgCp.Payloads = [][]byte{payload}

		msgBytes, err := json.Marshal(&msgCp)

		if err == nil {
			eg.Go(func() error {
				// Notify will automatically fall back to database storage if the
				// wrapped message exceeds pg_notify's 8KB limit
				return p.repo.Notify(ctx, topic.Name(), string(msgBytes))
			})
		} else {
			p.l.Error().Ctx(ctx).Err(err).Msg("error marshalling message")
		}
	}

	return eg.Wait()
}

// Sub subscribes to a topic. Inline NOTIFY payloads are handled directly;
// non-JSON notifications and a 1s ticker wake a poll that drains >8KB fallback
// rows. Delivery is at-most-once: handler errors are logged, never redelivered.
func (p *PubSub) Sub(topic msgqueue.Topic, handler msgqueue.AckHook) (func() error, error) {
	err := p.ensureQueue(context.Background(), topic)

	if err != nil {
		return nil, err
	}

	subscribeCtx, cancel := context.WithCancel(context.Background())

	// update the lastActive time on the queue row every 60 seconds so the
	// auto-delete reaper doesn't collect it while we're subscribed
	go func() {
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-subscribeCtx.Done():
				return
			case <-ticker.C:
				err := p.repo.UpdateQueueLastActive(subscribeCtx, topic.Name())

				if err != nil {
					p.l.Error().Err(err).Msg("error updating lastActive time")
				}
			}
		}
	}()

	handleMsg := func(task *msgqueue.Message, ackId *int64) {
		if ackId != nil {
			if err := p.repo.AckMessage(subscribeCtx, *ackId); err != nil {
				p.l.Error().Err(err).Msg("error acking message")
			}
		}

		if err := handler(task); err != nil {
			p.l.Error().Err(err).Msgf("error handling pubsub message %s", task.ID)
		}
	}

	// poll for >8KB fallback rows
	op := queueutils.NewOperationPool(p.l, 60*time.Second, "postgres-pubsub", queueutils.OpMethod(func(ctx context.Context, _ string) (bool, error) {
		messages, err := p.repo.ReadMessages(subscribeCtx, topic.Name(), fallbackPollBatchSize)

		if err != nil {
			p.l.Error().Err(err).Msg("error reading fallback messages")
			return false, err
		}

		for _, message := range messages {
			var task msgqueue.Message

			if err := json.Unmarshal(message.Payload, &task); err != nil {
				p.l.Error().Err(err).Msg("error unmarshalling fallback message")
				continue
			}

			handleMsg(&task, &message.ID)
		}

		return len(messages) == fallbackPollBatchSize, nil
	}))

	newMsgCh := make(chan struct{}, 1)

	go func() {
		err := p.repo.Listen(subscribeCtx, topic.Name(), func(ctx context.Context, notification *v1.PubSubMessage) error {
			// messages small enough for pg_notify arrive inline as JSON
			if len(notification.Payload) >= 1 && notification.Payload[0] == '{' {
				var task msgqueue.Message

				if err := json.Unmarshal(notification.Payload, &task); err != nil {
					p.l.Error().Err(err).Msg("error unmarshalling message")
					return err
				}

				handleMsg(&task, nil)
				return nil
			}

			// anything else is a wake-up signal to drain fallback rows
			select {
			case newMsgCh <- struct{}{}:
			default:
			}

			return nil
		})

		if err != nil && subscribeCtx.Err() == nil {
			p.l.Error().Err(err).Msg("error listening for new messages")
		}
	}()

	ticker := time.NewTicker(time.Second)

	go func() {
		defer ticker.Stop()

		for {
			select {
			case <-subscribeCtx.Done():
				return
			case <-ticker.C:
				op.RunOrContinue(topic.Name())
			case <-newMsgCh:
				op.RunOrContinue(topic.Name())
			}
		}
	}()

	return func() error {
		cancel()
		return nil
	}, nil
}

func (p *PubSub) ensureQueue(ctx context.Context, topic msgqueue.Topic) error {
	if valid, exists := p.ttlCache.Get(topic.Name()); valid && exists {
		return nil
	}

	err := p.repo.BindQueue(ctx, topic.Name(), false, true, false, nil)

	if err != nil {
		p.l.Error().Err(err).Msg("error binding queue")
		return err
	}

	p.ttlCache.Set(topic.Name(), true, time.Second*15)

	return nil
}
