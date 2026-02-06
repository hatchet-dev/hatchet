package rabbitmq

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/internal/queueutils"
	"github.com/hatchet-dev/hatchet/pkg/logger"
	"github.com/hatchet-dev/hatchet/pkg/random"
	"github.com/hatchet-dev/hatchet/pkg/telemetry"
)

const MAX_RETRY_COUNT = 15
const RETRY_INTERVAL = 2 * time.Second
const RETRY_RESET_INTERVAL = 30 * time.Second

// MessageQueueImpl implements MessageQueue interface using AMQP.
type MessageQueueImpl struct {
	ctx                       context.Context
	pubChannels               *channelPool
	l                         *zerolog.Logger
	subChannels               *channelPool
	tenantIdCache             *lru.Cache[uuid.UUID, bool]
	identity                  string
	configFs                  []MessageQueueImplOpt
	qos                       int
	deadLetterBackoff         time.Duration
	maxPayloadSize            int
	compressionThreshold      int
	maxDeathCount             int
	disableTenantExchangePubs bool
	compressionEnabled        bool
	enableMessageRejection    bool
}

func (t *MessageQueueImpl) IsReady() bool {
	return t.pubChannels.hasActiveConnection() && t.subChannels.hasActiveConnection()
}

type MessageQueueImplOpt func(*MessageQueueImplOpts)

type MessageQueueImplOpts struct {
	l                         *zerolog.Logger
	url                       string
	qos                       int
	deadLetterBackoff         time.Duration
	compressionThreshold      int
	maxDeathCount             int
	maxPubChannels            int32
	maxSubChannels            int32
	disableTenantExchangePubs bool
	compressionEnabled        bool
	enableMessageRejection    bool
}

func defaultMessageQueueImplOpts() *MessageQueueImplOpts {
	l := logger.NewDefaultLogger("rabbitmq")

	return &MessageQueueImplOpts{
		l:                         &l,
		disableTenantExchangePubs: false,
		deadLetterBackoff:         5 * time.Second,
		enableMessageRejection:    false,
		maxDeathCount:             5,
	}
}

func WithLogger(l *zerolog.Logger) MessageQueueImplOpt {
	return func(opts *MessageQueueImplOpts) {
		opts.l = l
	}
}

func WithURL(url string) MessageQueueImplOpt {
	return func(opts *MessageQueueImplOpts) {
		opts.url = url
	}
}

func WithMaxPubChannels(maxConns int32) MessageQueueImplOpt {
	return func(opts *MessageQueueImplOpts) {
		opts.maxPubChannels = maxConns
	}
}

func WithMaxSubChannels(maxConns int32) MessageQueueImplOpt {
	return func(opts *MessageQueueImplOpts) {
		opts.maxSubChannels = maxConns
	}
}

func WithQos(qos int) MessageQueueImplOpt {
	return func(opts *MessageQueueImplOpts) {
		opts.qos = qos
	}
}

func WithDisableTenantExchangePubs(disable bool) MessageQueueImplOpt {
	return func(opts *MessageQueueImplOpts) {
		opts.disableTenantExchangePubs = disable
	}
}

func WithDeadLetterBackoff(backoff time.Duration) MessageQueueImplOpt {
	return func(opts *MessageQueueImplOpts) {
		opts.deadLetterBackoff = backoff
	}
}

func WithGzipCompression(enabled bool, threshold int) MessageQueueImplOpt {
	return func(opts *MessageQueueImplOpts) {
		opts.compressionEnabled = enabled

		if threshold <= 0 {
			threshold = 5 * 1024 // default to 5KB
		}

		opts.compressionThreshold = threshold
	}
}

func WithMessageRejection(enabled bool, maxDeathCount int) MessageQueueImplOpt {
	return func(opts *MessageQueueImplOpts) {
		opts.enableMessageRejection = enabled

		if maxDeathCount <= 0 {
			maxDeathCount = 5
		}

		opts.maxDeathCount = maxDeathCount
	}
}

// New creates a new MessageQueueImpl.
func New(fs ...MessageQueueImplOpt) (func() error, *MessageQueueImpl, error) {
	ctx, cancel := context.WithCancel(context.Background())

	opts := defaultMessageQueueImplOpts()

	for _, f := range fs {
		f(opts)
	}

	newLogger := opts.l.With().Str("service", "rabbitmq").Logger()
	opts.l = &newLogger

	pubMaxChans := opts.maxPubChannels

	if pubMaxChans <= 0 {
		pubMaxChans = 20
	}

	pubChannelPool, err := newChannelPool(ctx, opts.l, opts.url, pubMaxChans)

	if err != nil {
		cancel()
		return nil, nil, err
	}

	subMaxChans := opts.maxSubChannels

	if subMaxChans <= 0 {
		subMaxChans = 100
	}

	subChannelPool, err := newChannelPool(ctx, opts.l, opts.url, subMaxChans)

	if err != nil {
		pubChannelPool.Close()
		cancel()
		return nil, nil, err
	}

	t := &MessageQueueImpl{
		ctx:                       ctx,
		identity:                  identity(),
		l:                         opts.l,
		qos:                       opts.qos,
		configFs:                  fs,
		disableTenantExchangePubs: opts.disableTenantExchangePubs,
		pubChannels:               pubChannelPool,
		subChannels:               subChannelPool,
		deadLetterBackoff:         opts.deadLetterBackoff,
		compressionEnabled:        opts.compressionEnabled,
		compressionThreshold:      opts.compressionThreshold,
		enableMessageRejection:    opts.enableMessageRejection,
		maxDeathCount:             opts.maxDeathCount,
		maxPayloadSize:            16 * 1024 * 1024, // 16 MB
	}

	// create a new lru cache for tenant ids
	t.tenantIdCache, _ = lru.New[uuid.UUID, bool](2000) // nolint: errcheck - this only returns an error if the size is less than 0

	// init the queues in a blocking fashion
	poolCh, err := subChannelPool.Acquire(ctx)

	if err != nil {
		t.l.Error().Msgf("[New] cannot acquire channel: %v", err)
		cancel()
		return nil, nil, err
	}

	ch := poolCh.Value()

	defer poolCh.Release()

	if _, err := t.initQueue(ch, msgqueue.TASK_PROCESSING_QUEUE); err != nil {
		cancel()
		return nil, nil, fmt.Errorf("failed to initialize queue: %w", err)
	}

	if _, err := t.initQueue(ch, msgqueue.OLAP_QUEUE); err != nil {
		cancel()
		return nil, nil, fmt.Errorf("failed to initialize queue: %w", err)
	}

	if _, err := t.initQueue(ch, msgqueue.DISPATCHER_DEAD_LETTER_QUEUE); err != nil {
		cancel()
		return nil, nil, fmt.Errorf("failed to initialize queue: %w", err)
	}

	return func() error {
		cancel()
		return nil
	}, t, nil
}

func (t *MessageQueueImpl) Clone() (func() error, msgqueue.MessageQueue, error) {
	return New(t.configFs...)
}

func (t *MessageQueueImpl) SetQOS(prefetchCount int) {
	t.qos = prefetchCount
}

const (
	mb                       = 1024 * 1024 // 1 MB in bytes
	maxSizeErrorLogThreshold = 10 * mb
)

func (t *MessageQueueImpl) SendMessage(ctx context.Context, q msgqueue.Queue, msg *msgqueue.Message) error {
	ctx, span := telemetry.NewSpan(ctx, "MessageQueueImpl.SendMessage")
	defer span.End()

	span.SetAttributes(
		attribute.String("MessageQueueImpl.SendMessage.queue_name", q.Name()),
		attribute.String("MessageQueueImpl.SendMessage.tenant_id", msg.TenantID.String()),
		attribute.String("MessageQueueImpl.SendMessage.message_id", msg.ID),
		attribute.Int("MessageQueueImpl.SendMessage.num_payloads", len(msg.Payloads)),
	)

	err := t.pubMessage(ctx, q, msg)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "error publishing message")
		return err
	}

	return nil
}

const PUB_ACQUIRE_CHANNEL_RETRIES = 3

func (t *MessageQueueImpl) pubMessage(ctx context.Context, q msgqueue.Queue, msg *msgqueue.Message) error {
	otelCarrier := telemetry.GetCarrier(ctx)

	ctx, span := telemetry.NewSpanWithCarrier(ctx, "publish-message", otelCarrier)
	defer span.End()

	telemetry.WithAttributes(span, telemetry.AttributeKV{Key: "tenant.id", Value: msg.TenantID})

	msg.SetOtelCarrier(otelCarrier)

	var compressionResult *CompressionResult

	// don't re-compress if the message was already compressed
	if len(msg.Payloads) > 0 && !msg.Compressed {
		var err error
		compressionResult, err = t.compressPayloads(msg.Payloads)
		if err != nil {
			t.l.Error().Msgf("error compressing payloads: %v", err)
			return fmt.Errorf("failed to compress payloads: %w", err)
		}

		if compressionResult.WasCompressed {
			msg.Payloads = compressionResult.Payloads
			msg.Compressed = true

			t.l.Debug().Msgf("compressed payloads for message %s: original=%d bytes, compressed=%d bytes, ratio=%.2f%%",
				msg.ID, compressionResult.OriginalSize, compressionResult.CompressedSize, compressionResult.CompressionRatio*100)
		}
	}

	var pub *amqp.Channel
	var acquireErr error

	for range PUB_ACQUIRE_CHANNEL_RETRIES {
		acquireCtx, acquireSpan := telemetry.NewSpan(ctx, "acquire_publish_channel")

		poolCh, err := t.pubChannels.Acquire(acquireCtx)

		if err != nil {
			acquireSpan.RecordError(err)
			acquireSpan.SetStatus(codes.Error, "error acquiring publish channel")
			acquireSpan.End()
			t.l.Error().Msgf("[pubMessage] cannot acquire channel: %v", err)
			// we don't retry this error, because it's always a timeout on acquiring the channel/connection, and is
			// unlikely to succeed on retry
			return err
		}

		acquireSpan.End()

		pub = poolCh.Value()

		// we need to case on the channel being closed here, because the channel exception may be async (after a previous pub has been
		// sent), so we might have acquired a closed channel from the pool
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
		t.l.Error().Msgf("error marshaling msg queue: %v", err)
		return err
	}

	bodySize := len(body)

	if bodySize > t.maxPayloadSize {
		if len(msg.Payloads) == 1 {
			err := fmt.Errorf("message size %d bytes exceeds maximum allowed size of %d bytes", bodySize, t.maxPayloadSize)
			span.RecordError(err)
			span.SetStatus(codes.Error, "message size exceeds maximum allowed size")
			return err
		}

		// split the payload in half each time
		// can change this value to configure the number of chunks we split the payload
		// into. more chunks means more messages published, but a smaller likelihood of needing
		// to recurse multiple times, and vice versa.
		numChunks := 2
		payloadsPerChunk := len(msg.Payloads) / numChunks

		if payloadsPerChunk < 1 {
			payloadsPerChunk = 1
		}

		// publish chunks sequentially to avoid channel pool exhaustion
		// parallel publishing at the chunk level causes too many concurrent channel acquisitions
		for chunk := range slices.Chunk(msg.Payloads, payloadsPerChunk) {
			// recursively call pubMessage with the chunked payloads
			// if the payload chunks are still too large, this will continue to split them
			// until they are under the max size.
			err := t.pubMessage(ctx, q, &msgqueue.Message{
				ID:                msg.ID,
				Payloads:          chunk,
				TenantID:          msg.TenantID,
				ImmediatelyExpire: msg.ImmediatelyExpire,
				Persistent:        msg.Persistent,
				OtelCarrier:       msg.OtelCarrier,
				Retries:           msg.Retries,
				Compressed:        msg.Compressed,
			})

			if err != nil {
				return err
			}
		}

		return nil
	}

	if bodySize > maxSizeErrorLogThreshold {
		t.l.Error().
			Int("message_size_bytes", bodySize).
			Int("num_messages", len(msg.Payloads)).
			Str("tenant_id", msg.TenantID.String()).
			Str("queue_name", q.Name()).
			Str("message_id", msg.ID).
			Msg("sending a very large message, this may impact performance")
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	t.l.Debug().Msgf("publishing msg to queue %s", q.Name())

	pubMsg := amqp.Publishing{
		Body: body,
	}

	if msg.Persistent {
		pubMsg.DeliveryMode = amqp.Persistent
	}

	if msg.ImmediatelyExpire {
		pubMsg.Expiration = "0"
	}

	ctx, pubSpan := telemetry.NewSpan(ctx, "publish_message")

	spanAttrs := []attribute.KeyValue{
		attribute.String("MessageQueueImpl.publish_message.queue_name", q.Name()),
		attribute.String("MessageQueueImpl.publish_message.tenant_id", msg.TenantID.String()),
		attribute.String("MessageQueueImpl.publish_message.message_id", msg.ID),
	}

	// Add compression metrics if payloads were present
	if compressionResult != nil && compressionResult.WasCompressed {
		spanAttrs = append(spanAttrs,
			attribute.Bool("MessageQueueImpl.publish_message.compressed", compressionResult.WasCompressed),
			attribute.Int("MessageQueueImpl.publish_message.original_size", compressionResult.OriginalSize),
			attribute.Int("MessageQueueImpl.publish_message.compressed_size", compressionResult.CompressedSize),
			attribute.Float64("MessageQueueImpl.publish_message.compression_ratio", compressionResult.CompressionRatio),
		)
	}

	pubSpan.SetAttributes(spanAttrs...)

	err = pub.PublishWithContext(ctx, "", q.Name(), false, false, pubMsg)

	// retry failed delivery on the next session
	if err != nil {
		pubSpan.RecordError(err)
		pubSpan.SetStatus(codes.Error, "error publishing message")
		pubSpan.End()
		return err
	}

	pubSpan.End()

	// if this is a tenant msg, publish to the tenant exchange
	if (!t.disableTenantExchangePubs || msg.ID == "task-stream-event") && msg.TenantID != uuid.Nil {
		// determine if the tenant exchange exists
		if _, ok := t.tenantIdCache.Get(msg.TenantID); !ok {
			// register the tenant exchange
			err = t.RegisterTenant(ctx, msg.TenantID)

			if err != nil {
				t.l.Error().Msgf("error registering tenant exchange: %v", err)
				return err
			}
		}

		t.l.Debug().Msgf("publishing tenant msg to exchange %s", msg.TenantID)

		err = pub.PublishWithContext(ctx, msgqueue.GetTenantExchangeName(msg.TenantID), "", false, false, amqp.Publishing{
			Body: body,
		})

		if err != nil {
			t.l.Error().Msgf("error publishing tenant msg: %v", err)
			return err
		}
	}

	t.l.Debug().Msgf("published msg to queue %s", q.Name())

	return nil
}

// Subscribe subscribes to the msg queue.
func (t *MessageQueueImpl) Subscribe(
	q msgqueue.Queue,
	preAck msgqueue.AckHook,
	postAck msgqueue.AckHook,
) (func() error, error) {
	ctx, cancel := context.WithCancel(context.Background())

	t.l.Debug().Msgf("subscribing to queue: %s", q.Name())

	cleanupSub, err := t.subscribe(ctx, t.identity, q, preAck, postAck)

	if err != nil {
		cancel()
		return nil, err
	}

	// only automatic DLQs get subscribed to, static DLQs require a separate subscription
	if q.DLQ() != nil && q.DLQ().IsAutoDLQ() {
		cleanupSubDLQ, err := t.subscribe(ctx, t.identity, q.DLQ(), preAck, postAck)

		if err != nil {
			cancel()
			return nil, err
		}

		f1 := cleanupSub
		f2 := cleanupSubDLQ

		cleanupSub = func() error {
			if err := f1(); err != nil {
				t.l.Error().Msgf("error cleaning up subscriber: %v", err)
			}

			if err := f2(); err != nil {
				t.l.Error().Msgf("error cleaning up subscriber: %v", err)
			}

			return nil
		}
	}

	return func() error {
		cancel()

		if err := cleanupSub(); err != nil {
			t.l.Error().Msgf("error cleaning up subscriber: %v", err)
			return nil
		}

		return nil
	}, nil
}

func (t *MessageQueueImpl) RegisterTenant(ctx context.Context, tenantId uuid.UUID) error {
	// create a new fanout exchange for the tenant
	poolCh, err := t.pubChannels.Acquire(ctx)

	if err != nil {
		t.l.Error().Msgf("[RegisterTenant] cannot acquire channel: %v", err)
		return err
	}

	sub := poolCh.Value()

	if sub.IsClosed() {
		poolCh.Destroy()
		return fmt.Errorf("channel is closed")
	}

	defer poolCh.Release()

	t.l.Debug().Msgf("registering tenant exchange: %s", tenantId)

	// create a fanout exchange for the tenant. each consumer of the fanout exchange will get notified
	// with the tenant events.
	err = sub.ExchangeDeclare(
		msgqueue.GetTenantExchangeName(tenantId),
		"fanout",
		true,  // durable
		false, // auto-deleted
		false, // not internal, accepts publishings
		false, // no-wait
		nil,   // arguments
	)

	if err != nil {
		t.l.Error().Msgf("cannot declare exchange: %q, %v", tenantId, err)
		return err
	}

	t.tenantIdCache.Add(tenantId, true)

	return nil
}

func (t *MessageQueueImpl) initQueue(ch *amqp.Channel, q msgqueue.Queue) (string, error) {
	if q.IsDLQ() {
		return q.Name(), nil
	}

	args := make(amqp.Table)
	args["x-consumer-timeout"] = 300000 // 5 minutes
	name := q.Name()

	if q.FanoutExchangeKey() != "" {
		suffix, err := random.Generate(8)

		if err != nil {
			t.l.Error().Msgf("error generating random bytes: %v", err)
			return "", err
		}

		name = fmt.Sprintf("%s-%s", q.Name(), suffix)
	}

	if !q.IsDLQ() && q.DLQ() != nil && q.DLQ().IsAutoDLQ() {
		dlx1 := getTmpDLQName(q.DLQ().Name())
		dlx2 := getProcDLQName(q.DLQ().Name())

		args["x-dead-letter-exchange"] = ""
		args["x-dead-letter-routing-key"] = dlx1

		dlq1Args := make(amqp.Table)

		dlq1Args["x-dead-letter-exchange"] = ""
		dlq1Args["x-dead-letter-routing-key"] = dlx2
		dlq1Args["x-message-ttl"] = t.deadLetterBackoff.Milliseconds()

		dlq2Args := make(amqp.Table)

		dlq2Args["x-dead-letter-exchange"] = ""
		dlq2Args["x-dead-letter-routing-key"] = dlx1

		// declare the dead letter queue
		if _, err := ch.QueueDeclare(dlx1, true, false, false, false, dlq1Args); err != nil {
			t.l.Error().Msgf("cannot declare dead letter exchange/queue: %s, %s", dlx1, err.Error())
			return "", err
		}

		if _, err := ch.QueueDeclare(dlx2, true, false, false, false, dlq2Args); err != nil {
			t.l.Error().Msgf("cannot declare dead letter exchange/queue: %s, %s", dlx2, err.Error())
			return "", err
		}
	}

	if !q.IsDLQ() && q.DLQ() != nil && !q.DLQ().IsAutoDLQ() {
		args["x-dead-letter-exchange"] = ""
		args["x-dead-letter-routing-key"] = q.DLQ().Name()
	}

	if q.IsExpirable() {
		args["x-message-ttl"] = int32(20000) // 20 seconds
		args["x-expires"] = int32(600000)    // 10 minutes
	}

	if _, err := ch.QueueDeclare(name, q.Durable(), q.AutoDeleted(), q.Exclusive(), false, args); err != nil {
		t.l.Error().Msgf("cannot declare queue: %q, %v", name, err)
		return "", err
	}

	// if the queue has a subscriber key, bind it to the fanout exchange
	if q.FanoutExchangeKey() != "" {
		t.l.Debug().Msgf("binding queue: %s to exchange: %s", name, q.FanoutExchangeKey())

		if err := ch.QueueBind(name, "", q.FanoutExchangeKey(), false, nil); err != nil {
			t.l.Error().Msgf("cannot bind queue: %q, %v", name, err)
			return "", err
		}
	}

	return name, nil
}

func (t *MessageQueueImpl) subscribe(
	ctx context.Context,
	subId string,
	q msgqueue.Queue,
	preAck msgqueue.AckHook,
	postAck msgqueue.AckHook,
) (func() error, error) {
	sessionCount := 0

	wg := sync.WaitGroup{}
	var queueName string

	if !q.Exclusive() {
		poolCh, err := t.subChannels.Acquire(ctx)

		if err != nil {
			return nil, fmt.Errorf("cannot acquire channel for initializing queue: %v", err)
		}

		sub := poolCh.Value()

		if sub.IsClosed() {
			poolCh.Destroy()
			return nil, fmt.Errorf("channel is closed")
		}

		// we initialize the queue here because exclusive queues are bound to the session/connection. however, it's not clear
		// if the exclusive queue will be available to the next session.
		queueName, err = t.initQueue(sub, q)

		if err != nil {
			poolCh.Release()
			return nil, fmt.Errorf("error initializing queue: %v", err)
		}

		poolCh.Release()
	}

	if q.IsDLQ() && q.IsAutoDLQ() {
		queueName = getProcDLQName(q.Name())
	}

	innerFn := func() error {
		poolCh, err := t.subChannels.Acquire(ctx)

		if err != nil {
			return err
		}

		sub := poolCh.Value()

		if sub.IsClosed() {
			poolCh.Destroy()
			return fmt.Errorf("channel is closed, destroying and retrying")
		}

		defer poolCh.Release()

		// we initialize the queue here because exclusive queues are bound to the session/connection. however, it's not clear
		// if the exclusive queue will be available to the next session.
		if q.Exclusive() {
			queueName, err = t.initQueue(sub, q)

			if err != nil {
				return err
			}
		}

		// We'd like to limit to 1k TPS per engine. The max channels on an instance is 10.
		err = sub.Qos(t.qos, 0, false)

		if err != nil {
			return err
		}

		deliveries, err := sub.ConsumeWithContext(ctx, queueName, subId, false, q.Exclusive(), false, false, nil)

		if err != nil {
			return err
		}

		wg.Add(1) // we add an extra delta for the deliveries channel to be closed
		defer wg.Done()

		for rabbitMsg := range deliveries {
			wg.Add(1)

			go func(rabbitMsg amqp.Delivery) {
				defer wg.Done()

				msg := &msgqueue.Message{}

				if len(rabbitMsg.Body) == 0 {
					t.l.Error().Msgf("empty message body for message: %s", rabbitMsg.MessageId)

					// reject this message
					if err := rabbitMsg.Reject(false); err != nil {
						t.l.Error().Msgf("error rejecting message: %v", err)
					}

					return
				}

				if err := json.Unmarshal(rabbitMsg.Body, msg); err != nil {
					t.l.Error().Msgf("error unmarshalling message: %v", err)

					// reject this message
					if err := rabbitMsg.Reject(false); err != nil {
						t.l.Error().Msgf("error rejecting message: %v", err)
					}

					return
				}

				if msg.Compressed {
					decompressedPayloads, err := t.decompressPayloads(msg.Payloads)
					if err != nil {
						t.l.Error().Msgf("error decompressing payloads: %v", err)
						// reject this message
						if err := rabbitMsg.Reject(false); err != nil {
							t.l.Error().Msgf("error rejecting message: %v", err)
						}
						return
					}
					msg.Payloads = decompressedPayloads
				}

				// determine if we've hit the max number of retries
				xDeath, exists := rabbitMsg.Headers["x-death"].([]interface{})

				if exists {
					// message was rejected before
					deathCount := xDeath[0].(amqp.Table)["count"].(int64)

					if deathCount > 5 {
						t.l.Error().
							Int64("death_count", deathCount).
							Str("message_id", msg.ID).
							Str("tenant_id", msg.TenantID.String()).
							Int("num_payloads", len(msg.Payloads)).
							Msgf("message has been retried for %d times", deathCount)
					}

					if t.enableMessageRejection && deathCount > int64(t.maxDeathCount) {
						t.l.Error().
							Int64("death_count", deathCount).
							Str("message_id", msg.ID).
							Str("tenant_id", msg.TenantID.String()).
							Int("max_death_count", t.maxDeathCount).
							Msg("permanently rejecting message due to exceeding max death count")

						if err := rabbitMsg.Ack(false); err != nil {
							t.l.Error().Err(err).Msg("error permanently rejecting message")
						}
						return
					}
				}

				t.l.Debug().Msgf("(session: %d) got msg", sessionCount)

				if err := preAck(msg); err != nil {
					if isPermanentPreAckError(err) {
						t.l.Error().
							Err(err).
							Str("message_id", msg.ID).
							Str("tenant_id", msg.TenantID.String()).
							Int("num_payloads", len(msg.Payloads)).
							Msg("dropping message due to permanent pre-ack error")

						if ackErr := rabbitMsg.Ack(false); ackErr != nil {
							t.l.Error().Err(ackErr).Msg("error acknowledging message after permanent pre-ack error")
						}

						return
					}

					t.l.Error().Msgf("error in pre-ack on msg %s: %v", msg.ID, err)

					// nack the message
					if err := rabbitMsg.Reject(false); err != nil {
						t.l.Error().Msgf("error rejecting message: %v", err)
					}

					return
				}

				if err := sub.Ack(rabbitMsg.DeliveryTag, false); err != nil {
					t.l.Error().Msgf("error acknowledging message: %v", err)
					return
				}

				if err := postAck(msg); err != nil {
					t.l.Error().Msgf("error in post-ack: %v", err)
					return
				}
			}(rabbitMsg)
		}

		t.l.Info().Msg("deliveries channel closed")

		return nil
	}

	go func() {
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

				t.l.Error().Msgf("could not run inner loop (retry count %d): %v", retryCount, err)
				queueutils.SleepWithExponentialBackoff(10*time.Millisecond, 5*time.Second, retryCount)
				lastRetry = time.Now()
				retryCount++
				continue
			}
		}
	}()

	cleanup := func() error {
		t.l.Debug().Msgf("shutting down subscriber: %s", subId)
		wg.Wait()
		t.l.Debug().Msgf("successfully shut down subscriber: %s", subId)

		return nil
	}

	return cleanup, nil
}

func isPermanentPreAckError(err error) bool {
	if err == nil {
		return false
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		// invalid input syntax for type json / jsonb
		if pgErr.Code == pgerrcode.InvalidTextRepresentation {
			return true
		}
	}

	// Fallback: some error paths may lose pg error type info.
	errStr := err.Error()
	if strings.Contains(errStr, fmt.Sprintf("SQLSTATE %s", pgerrcode.InvalidTextRepresentation)) {
		return true
	}
	if strings.Contains(errStr, "invalid input syntax for type json") {
		return true
	}

	return false
}

// identity returns the same host/process unique string for the lifetime of
// this process so that subscriber reconnections reuse the same queue name.
func identity() string {
	hostname, err := os.Hostname()
	h := sha256.New()
	_, _ = fmt.Fprint(h, hostname)
	_, _ = fmt.Fprint(h, err)
	_, _ = fmt.Fprint(h, os.Getpid())
	return fmt.Sprintf("%x", h.Sum(nil))
}

func getTmpDLQName(dlxName string) string {
	return fmt.Sprintf("%s_tmp", dlxName)
}

func getProcDLQName(dlxName string) string {
	return fmt.Sprintf("%s_proc", dlxName)
}
