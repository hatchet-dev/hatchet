package rabbitmq

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog"

	msgqueue "github.com/hatchet-dev/hatchet/internal/msgqueue/v1"
	"github.com/hatchet-dev/hatchet/internal/queueutils"
	"github.com/hatchet-dev/hatchet/internal/telemetry"
	"github.com/hatchet-dev/hatchet/pkg/logger"
	"github.com/hatchet-dev/hatchet/pkg/random"
	"github.com/hatchet-dev/hatchet/pkg/repository/metered"
)

const MAX_RETRY_COUNT = 15
const RETRY_INTERVAL = 2 * time.Second
const RETRY_RESET_INTERVAL = 30 * time.Second

// MessageQueueImpl implements MessageQueue interface using AMQP.
type MessageQueueImpl struct {
	ctx      context.Context
	identity string
	configFs []MessageQueueImplOpt

	qos int

	l *zerolog.Logger

	disableTenantExchangePubs bool

	// lru cache for tenant ids
	tenantIdCache *lru.Cache[string, bool]

	channels *channelPool

	deadLetterBackoff time.Duration
}

func (t *MessageQueueImpl) IsReady() bool {
	return t.channels.hasActiveConnection()
}

type MessageQueueImplOpt func(*MessageQueueImplOpts)

type MessageQueueImplOpts struct {
	l                         *zerolog.Logger
	url                       string
	qos                       int
	disableTenantExchangePubs bool
	deadLetterBackoff         time.Duration
}

func defaultMessageQueueImplOpts() *MessageQueueImplOpts {
	l := logger.NewDefaultLogger("rabbitmq")

	return &MessageQueueImplOpts{
		l:                         &l,
		disableTenantExchangePubs: false,
		deadLetterBackoff:         5 * time.Second,
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

// New creates a new MessageQueueImpl.
func New(fs ...MessageQueueImplOpt) (func() error, *MessageQueueImpl) {
	ctx, cancel := context.WithCancel(context.Background())

	opts := defaultMessageQueueImplOpts()

	for _, f := range fs {
		f(opts)
	}

	newLogger := opts.l.With().Str("service", "rabbitmq").Logger()
	opts.l = &newLogger

	channelPool, err := newChannelPool(ctx, opts.l, opts.url)

	if err != nil {
		cancel()
		return nil, nil
	}

	t := &MessageQueueImpl{
		ctx:                       ctx,
		identity:                  identity(),
		l:                         opts.l,
		qos:                       opts.qos,
		configFs:                  fs,
		disableTenantExchangePubs: opts.disableTenantExchangePubs,
		channels:                  channelPool,
		deadLetterBackoff:         opts.deadLetterBackoff,
	}

	// create a new lru cache for tenant ids
	t.tenantIdCache, _ = lru.New[string, bool](2000) // nolint: errcheck - this only returns an error if the size is less than 0

	// init the queues in a blocking fashion
	poolCh, err := channelPool.Acquire(ctx)

	if err != nil {
		t.l.Error().Msgf("cannot acquire channel: %v", err)
		cancel()
		return nil, nil
	}

	ch := poolCh.Value()

	defer poolCh.Release()

	if _, err := t.initQueue(ch, msgqueue.TASK_PROCESSING_QUEUE); err != nil {
		t.l.Debug().Msgf("error initializing queue: %v", err)
		cancel()
		return nil, nil
	}

	if _, err := t.initQueue(ch, msgqueue.OLAP_QUEUE); err != nil {
		t.l.Debug().Msgf("error initializing queue: %v", err)
		cancel()
		return nil, nil
	}

	return func() error {
		cancel()
		return nil
	}, t
}

func (t *MessageQueueImpl) Clone() (func() error, msgqueue.MessageQueue) {
	return New(t.configFs...)
}

func (t *MessageQueueImpl) SetQOS(prefetchCount int) {
	t.qos = prefetchCount
}

func (t *MessageQueueImpl) SendMessage(ctx context.Context, q msgqueue.Queue, msg *msgqueue.Message) error {
	return t.pubMessage(ctx, q, msg)
}

func (t *MessageQueueImpl) pubMessage(ctx context.Context, q msgqueue.Queue, msg *msgqueue.Message) error {
	otelCarrier := telemetry.GetCarrier(ctx)

	ctx, span := telemetry.NewSpanWithCarrier(ctx, "publish-message", otelCarrier)
	defer span.End()

	msg.SetOtelCarrier(otelCarrier)

	poolCh, err := t.channels.Acquire(ctx)

	if err != nil {
		t.l.Error().Msgf("cannot acquire channel: %v", err)
		return err
	}

	pub := poolCh.Value()

	if pub.IsClosed() {
		poolCh.Destroy()
		return fmt.Errorf("channel is closed")
	}

	defer poolCh.Release()

	body, err := json.Marshal(msg)

	if err != nil {
		t.l.Error().Msgf("error marshaling msg queue: %v", err)
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
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

	err = pub.PublishWithContext(ctx, "", q.Name(), false, false, pubMsg)

	// retry failed delivery on the next session
	if err != nil {
		return err
	}

	// if this is a tenant msg, publish to the tenant exchange
	if !t.disableTenantExchangePubs && msg.TenantID != "" {
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

	if q.DLQ() != nil {
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

func (t *MessageQueueImpl) RegisterTenant(ctx context.Context, tenantId string) error {
	// create a new fanout exchange for the tenant
	poolCh, err := t.channels.Acquire(ctx)

	if err != nil {
		t.l.Error().Msgf("cannot acquire channel: %v", err)
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

	if !q.IsDLQ() && q.DLQ() != nil {
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

// deleteQueue is a helper function for removing durable queues which are used for tests.
func (t *MessageQueueImpl) deleteQueue(q msgqueue.Queue) error {
	poolCh, err := t.channels.Acquire(context.Background())

	if err != nil {
		t.l.Error().Msgf("cannot acquire channel for deleting queue: %v", err)
		return err
	}

	ch := poolCh.Value()

	if ch.IsClosed() {
		poolCh.Destroy()
		return fmt.Errorf("channel is closed")
	}

	defer poolCh.Release()

	_, err = ch.QueueDelete(q.Name(), true, true, false)

	if err != nil {
		t.l.Error().Msgf("cannot delete queue: %q, %v", q.Name(), err)
		return err
	}

	if q.DLQ().Name() != "" {
		dlq1 := getTmpDLQName(q.DLQ().Name())
		dlq2 := getProcDLQName(q.DLQ().Name())

		_, err = ch.QueueDelete(dlq1, true, true, false)

		if err != nil {
			t.l.Error().Msgf("cannot delete dead letter queue: %q, %v", dlq1, err)
			return err
		}

		_, err = ch.QueueDelete(dlq2, true, true, false)

		if err != nil {
			t.l.Error().Msgf("cannot delete dead letter queue: %q, %v", dlq2, err)
			return err
		}
	}

	return nil
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
		poolCh, err := t.channels.Acquire(ctx)

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

	if q.IsDLQ() {
		queueName = getProcDLQName(q.Name())
	}

	innerFn := func() error {
		poolCh, err := t.channels.Acquire(ctx)

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

				// determine if we've hit the max number of retries
				xDeath, exists := rabbitMsg.Headers["x-death"].([]interface{})

				if exists {
					// message was rejected before
					deathCount := xDeath[0].(amqp.Table)["count"].(int64)

					t.l.Debug().Msgf("message has been rejected %d times", deathCount)

					if deathCount > int64(msg.Retries) {
						t.l.Debug().Msgf("message has been rejected %d times, not requeuing", deathCount)

						// acknowledge so it's removed from the queue
						if err := rabbitMsg.Ack(false); err != nil {
							t.l.Error().Msgf("error acknowledging message: %v", err)
						}

						return
					}
				}

				t.l.Debug().Msgf("(session: %d) got msg", sessionCount)

				if err := preAck(msg); err != nil {
					t.l.Error().Msgf("error in pre-ack on msg %s: %v", msg.ID, err)

					if err == metered.ErrResourceExhausted {
						return
					}

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
