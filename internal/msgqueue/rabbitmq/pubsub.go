package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"sync"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/internal/queueutils"
	"github.com/hatchet-dev/hatchet/pkg/logger"
	"github.com/hatchet-dev/hatchet/pkg/random"
)

// PubSub implements msgqueue.PubSub over RabbitMQ. Tenant topics map to the
// legacy per-tenant fanout exchanges ("<uuid>_v1") and scheduler partition
// topics map to the scheduler's exclusive queue on the default exchange, so
// mixed-version fleets interoperate.
//
// INVARIANT: the PubSub owns its AMQP connections and channel pools and never
// shares them with the durable MessageQueueImpl, since Pub can be called from
// within durable-write paths.
type PubSub struct {
	ctx      context.Context
	identity string
	l        *zerolog.Logger

	pubChannels *channelPool
	subChannels *channelPool

	// lru cache of tenant exchanges we've already declared
	exchangeCache *lru.Cache[string, bool]

	compressor

	maxPayloadSize int
}

type PubSubOpt func(*PubSubOpts)

type PubSubOpts struct {
	l                    *zerolog.Logger
	url                  string
	maxPubChannels       int32
	maxSubChannels       int32
	compressionEnabled   bool
	compressionThreshold int
}

func defaultPubSubOpts() *PubSubOpts {
	l := logger.NewDefaultLogger("rabbitmq-pubsub")

	return &PubSubOpts{
		l:              &l,
		maxPubChannels: 10,
		maxSubChannels: 20,
	}
}

func WithPubSubURL(url string) PubSubOpt {
	return func(opts *PubSubOpts) {
		opts.url = url
	}
}

func WithPubSubLogger(l *zerolog.Logger) PubSubOpt {
	return func(opts *PubSubOpts) {
		opts.l = l
	}
}

func WithPubSubMaxPubChannels(maxChannels int32) PubSubOpt {
	return func(opts *PubSubOpts) {
		if maxChannels > 0 {
			opts.maxPubChannels = maxChannels
		}
	}
}

func WithPubSubMaxSubChannels(maxChannels int32) PubSubOpt {
	return func(opts *PubSubOpts) {
		if maxChannels > 0 {
			opts.maxSubChannels = maxChannels
		}
	}
}

// WithPubSubGzip enables gzip compression of payloads. Compression settings
// must match the durable MessageQueue's settings for wire compatibility, since
// both sides publish to the same tenant exchanges in mixed-version fleets.
func WithPubSubGzip(enabled bool, threshold int) PubSubOpt {
	return func(opts *PubSubOpts) {
		opts.compressionEnabled = enabled

		if threshold <= 0 {
			threshold = 5 * 1024 // default to 5KB
		}

		opts.compressionThreshold = threshold
	}
}

// NewPubSub creates a new RabbitMQ-backed PubSub with its own connections and
// channel pools.
func NewPubSub(fs ...PubSubOpt) (func() error, *PubSub, error) {
	ctx, cancel := context.WithCancel(context.Background())

	opts := defaultPubSubOpts()

	for _, f := range fs {
		f(opts)
	}

	newLogger := opts.l.With().Str("service", "rabbitmq-pubsub").Logger()
	opts.l = &newLogger

	pubChannelPool, err := newChannelPool(ctx, opts.l, opts.url, opts.maxPubChannels)

	if err != nil {
		cancel()
		return nil, nil, err
	}

	subChannelPool, err := newChannelPool(ctx, opts.l, opts.url, opts.maxSubChannels)

	if err != nil {
		pubChannelPool.Close()
		cancel()
		return nil, nil, err
	}

	p := &PubSub{
		ctx:         ctx,
		identity:    identity(),
		l:           opts.l,
		pubChannels: pubChannelPool,
		subChannels: subChannelPool,
		compressor: compressor{
			compressionEnabled:   opts.compressionEnabled,
			compressionThreshold: opts.compressionThreshold,
		},
		maxPayloadSize: 16 * 1024 * 1024, // 16 MB
	}

	p.exchangeCache, _ = lru.New[string, bool](2000) //nolint:errcheck // this only returns an error if the size is less than 0

	return func() error {
		cancel()
		pubChannelPool.Close()
		subChannelPool.Close()
		return nil
	}, p, nil
}

func (p *PubSub) IsReady() bool {
	return p.pubChannels.hasActiveConnection() && p.subChannels.hasActiveConnection()
}

// Pub publishes a message to the topic. Delivery is best-effort and
// non-persistent: if no subscriber queue is bound, the message is dropped.
func (p *PubSub) Pub(ctx context.Context, topic msgqueue.Topic, msg *msgqueue.Message) error {
	// compress exactly as the durable pubMessage does: in mixed-version fleets,
	// old engines' mirrored copies land on the same exchanges compressed, so
	// subscribers must handle both and publishers must match settings. We work
	// on a shallow copy: callers dual-publish the same *Message they hand to
	// the durable queue, which may not have serialized it yet.
	if len(msg.Payloads) > 0 && !msg.Compressed {
		compressionResult, err := p.compressPayloads(msg.Payloads)

		if err != nil {
			p.l.Error().Msgf("error compressing payloads: %v", err)
			return fmt.Errorf("failed to compress payloads: %w", err)
		}

		if compressionResult.WasCompressed {
			msgCp := *msg
			msgCp.Payloads = compressionResult.Payloads
			msgCp.Compressed = true
			msg = &msgCp
		}
	}

	var pub *amqp.Channel
	var acquireErr error

	for range PUB_ACQUIRE_CHANNEL_RETRIES {
		poolCh, err := p.pubChannels.Acquire(ctx)

		if err != nil {
			p.l.Error().Msgf("[PubSub.Pub] cannot acquire channel: %v", err)
			return err
		}

		pub = poolCh.Value()

		// the channel exception may be async (after a previous pub has been sent),
		// so we might have acquired a closed channel from the pool
		if pub.IsClosed() {
			poolCh.Destroy()
			acquireErr = fmt.Errorf("channel is closed")
			pub = nil
			continue
		}

		defer poolCh.Release()
		break
	}

	if pub == nil {
		return acquireErr
	}

	body, err := json.Marshal(msg)

	if err != nil {
		p.l.Error().Msgf("error marshaling pubsub message: %v", err)
		return err
	}

	if len(body) > p.maxPayloadSize {
		if len(msg.Payloads) == 1 {
			return fmt.Errorf("message size %d bytes exceeds maximum allowed size of %d bytes", len(body), p.maxPayloadSize)
		}

		// split the payloads in half and publish recursively until each chunk is
		// under the max size (same strategy as the durable pubMessage)
		payloadsPerChunk := max(len(msg.Payloads)/2, 1)

		for chunk := range slices.Chunk(msg.Payloads, payloadsPerChunk) {
			err := p.Pub(ctx, topic, &msgqueue.Message{
				ID:                msg.ID,
				Payloads:          chunk,
				TenantID:          msg.TenantID,
				ImmediatelyExpire: msg.ImmediatelyExpire,
				Persistent:        msg.Persistent,
				OtelCarrier:       msg.OtelCarrier,
				Retries:           msg.Retries, // nolint: staticcheck
				Compressed:        msg.Compressed,
			})

			if err != nil {
				return err
			}
		}

		return nil
	}

	exchange := ""
	routingKey := topic.Name()

	if topic.Kind() == msgqueue.TopicKindTenantStream {
		// tenant streams ride the per-tenant fanout exchange; declare it lazily
		if _, ok := p.exchangeCache.Get(topic.Name()); !ok {
			if err := p.declareExchange(pub, topic.Name()); err != nil {
				p.l.Error().Msgf("error declaring exchange %s: %v", topic.Name(), err)
				return err
			}

			p.exchangeCache.Add(topic.Name(), true)
		}

		exchange = topic.Name()
		routingKey = ""
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// never persistent, expire immediately without an active consumer: this is
	// an at-most-once notification path
	err = pub.PublishWithContext(ctx, exchange, routingKey, false, false, amqp.Publishing{
		Body:       body,
		Expiration: "0",
	})

	if err != nil {
		p.l.Error().Msgf("error publishing pubsub message to topic %s: %v", topic.Name(), err)
		return err
	}

	return nil
}

// Sub subscribes to a topic. Delivery is at-most-once: messages are acked
// before the handler runs, and handler errors are logged, never redelivered.
func (p *PubSub) Sub(topic msgqueue.Topic, handler msgqueue.MsgHandler) (func() error, error) {
	ctx, cancel := context.WithCancel(p.ctx)

	p.l.Debug().Msgf("subscribing to topic: %s", topic.Name())

	sessionCount := 0
	wg := sync.WaitGroup{}

	innerFn := func() error {
		poolCh, err := p.subChannels.Acquire(ctx)

		if err != nil {
			return err
		}

		sub := poolCh.Value()

		if sub.IsClosed() {
			poolCh.Destroy()
			return fmt.Errorf("channel is closed, destroying and retrying")
		}

		defer poolCh.Release()

		// the queues are exclusive and bound to the session, so we declare them
		// on each new session
		queueName, err := p.declareSubQueue(sub, topic)

		if err != nil {
			return err
		}

		deliveries, err := sub.ConsumeWithContext(ctx, queueName, p.identity, false, true, false, false, nil)

		if err != nil {
			return err
		}

		for rabbitMsg := range deliveries {
			wg.Add(1)

			go func(rabbitMsg amqp.Delivery, session int) {
				defer wg.Done()

				// always ack: at-most-once, no redelivery
				if err := sub.Ack(rabbitMsg.DeliveryTag, false); err != nil {
					p.l.Error().Msgf("error acknowledging message: %v", err)
					return
				}

				if len(rabbitMsg.Body) == 0 {
					p.l.Error().Msgf("empty message body for message: %s", rabbitMsg.MessageId)
					return
				}

				msg := &msgqueue.Message{}

				if err := json.Unmarshal(rabbitMsg.Body, msg); err != nil {
					p.l.Error().Msgf("error unmarshalling message: %v", err)
					return
				}

				if msg.Compressed {
					decompressedPayloads, err := msgqueue.DecompressPayloads(msg.Payloads)

					if err != nil {
						p.l.Error().Msgf("error decompressing payloads: %v", err)
						return
					}

					msg.Payloads = decompressedPayloads
				}

				p.l.Debug().Msgf("(session: %d) got pubsub msg", session)

				if err := handler(msg); err != nil {
					p.l.Error().Msgf("error handling pubsub message %s: %v", msg.ID, err)
				}
			}(rabbitMsg, sessionCount)
		}

		p.l.Info().Msg("pubsub deliveries channel closed")

		return nil
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		retryCount := 0
		lastRetry := time.Now()

		for {
			if ctx.Err() != nil {
				return
			}

			sessionCount++

			if err := innerFn(); err != nil {
				if time.Since(lastRetry) > RETRY_RESET_INTERVAL {
					retryCount = 0
				}

				p.l.Error().Msgf("could not run pubsub subscriber loop (retry count %d): %v", retryCount, err)
				queueutils.SleepWithExponentialBackoff(10*time.Millisecond, 5*time.Second, retryCount)
				lastRetry = time.Now()
				retryCount++
				continue
			}
		}
	}()

	cleanup := func() error {
		p.l.Debug().Msgf("shutting down pubsub subscriber: %s", topic.Name())
		cancel()
		wg.Wait()
		p.l.Debug().Msgf("successfully shut down pubsub subscriber: %s", topic.Name())

		return nil
	}

	return cleanup, nil
}

// declareSubQueue declares the consumer queue for a topic and returns its
// name. Tenant topics get a random-suffix exclusive queue bound to the tenant
// fanout exchange; scheduler topics consume the well-known partition queue on
// the default exchange.
func (p *PubSub) declareSubQueue(ch *amqp.Channel, topic msgqueue.Topic) (string, error) {
	args := make(amqp.Table)
	args["x-consumer-timeout"] = 300000 // 5 minutes

	name := topic.Name()

	if topic.Kind() == msgqueue.TopicKindTenantStream {
		if err := p.declareExchange(ch, topic.Name()); err != nil {
			return "", err
		}

		suffix, err := random.Generate(8)

		if err != nil {
			p.l.Error().Msgf("error generating random bytes: %v", err)
			return "", err
		}

		name = fmt.Sprintf("%s-%s", topic.Name(), suffix)
	}

	if _, err := ch.QueueDeclare(name, false, true, true, false, args); err != nil {
		p.l.Error().Msgf("cannot declare queue: %q, %v", name, err)
		return "", err
	}

	if topic.Kind() == msgqueue.TopicKindTenantStream {
		p.l.Debug().Msgf("binding queue: %s to exchange: %s", name, topic.Name())

		if err := ch.QueueBind(name, "", topic.Name(), false, nil); err != nil {
			p.l.Error().Msgf("cannot bind queue: %q, %v", name, err)
			return "", err
		}
	}

	return name, nil
}

// declareExchange declares a tenant fanout exchange. The args are identical to
// the legacy declareTenantExchange so declares are conflict-free in
// mixed-version fleets.
func (p *PubSub) declareExchange(ch *amqp.Channel, name string) error {
	return ch.ExchangeDeclare(
		name,
		"fanout",
		true,  // durable
		false, // auto-deleted
		false, // not internal, accepts publishings
		false, // no-wait
		nil,   // arguments
	)
}
