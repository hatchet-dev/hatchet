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
	"golang.org/x/exp/rand"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/internal/telemetry"
	"github.com/hatchet-dev/hatchet/pkg/logger"
	"github.com/hatchet-dev/hatchet/pkg/random"
)

const MAX_RETRY_COUNT = 15
const RETRY_INTERVAL = 2 * time.Second

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

	connections *connectionPool
	channels    *channelPool
}

func (t *MessageQueueImpl) IsReady() bool {
	return t.connections.hasActiveConnection()
}

type MessageQueueImplOpt func(*MessageQueueImplOpts)

type MessageQueueImplOpts struct {
	l                         *zerolog.Logger
	url                       string
	qos                       int
	disableTenantExchangePubs bool
}

func defaultMessageQueueImplOpts() *MessageQueueImplOpts {
	l := logger.NewDefaultLogger("rabbitmq")

	return &MessageQueueImplOpts{
		l:                         &l,
		disableTenantExchangePubs: false,
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

// New creates a new MessageQueueImpl.
func New(fs ...MessageQueueImplOpt) (func() error, *MessageQueueImpl) {
	ctx, cancel := context.WithCancel(context.Background())

	opts := defaultMessageQueueImplOpts()

	for _, f := range fs {
		f(opts)
	}

	newLogger := opts.l.With().Str("service", "rabbitmq").Logger()
	opts.l = &newLogger

	// initialize the connection and channel pools
	connectionPool, err := newConnectionPool(ctx, opts.l, opts.url)

	if err != nil {
		cancel()
		return nil, nil
	}

	channelPool, err := newChannelPool(ctx, opts.l, connectionPool)

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
		connections:               connectionPool,
		channels:                  channelPool,
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

	if _, err := t.initQueue(ch, msgqueue.EVENT_PROCESSING_QUEUE); err != nil {
		t.l.Debug().Msgf("error initializing queue: %v", err)
		cancel()
		return nil, nil
	}

	if _, err := t.initQueue(ch, msgqueue.JOB_PROCESSING_QUEUE); err != nil {
		t.l.Debug().Msgf("error initializing queue: %v", err)
		cancel()
		return nil, nil
	}

	if _, err := t.initQueue(ch, msgqueue.WORKFLOW_PROCESSING_QUEUE); err != nil {
		t.l.Debug().Msgf("error initializing queue: %v", err)
		cancel()
		return nil, nil
	}

	if _, err := t.initQueue(ch, msgqueue.TASK_PROCESSING_QUEUE); err != nil {
		t.l.Debug().Msgf("error initializing queue: %v", err)
		cancel()
		return nil, nil
	}

	if _, err := t.initQueue(ch, msgqueue.TRIGGER_QUEUE); err != nil {
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

		err = pub.PublishWithContext(ctx, msg.TenantID, "", false, false, amqp.Publishing{
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

	defer poolCh.Release()

	t.l.Debug().Msgf("registering tenant exchange: %s", tenantId)

	// create a fanout exchange for the tenant. each consumer of the fanout exchange will get notified
	// with the tenant events.
	err = sub.ExchangeDeclare(
		tenantId,
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

func (t *MessageQueueImpl) initQueue(ch *channelWithId, q msgqueue.Queue) (string, error) {
	args := make(amqp.Table)
	name := q.Name()

	if q.FanoutExchangeKey() != "" {
		suffix, err := random.Generate(8)

		if err != nil {
			t.l.Error().Msgf("error generating random bytes: %v", err)
			return "", err
		}

		name = fmt.Sprintf("%s-%s", q.Name(), suffix)
	}

	if q.DLX() != "" {
		args["x-dead-letter-exchange"] = ""
		args["x-dead-letter-routing-key"] = q.DLX()
		args["x-consumer-timeout"] = 5000 // 5 seconds

		dlqArgs := make(amqp.Table)

		dlqArgs["x-dead-letter-exchange"] = ""
		dlqArgs["x-dead-letter-routing-key"] = name
		dlqArgs["x-message-ttl"] = 5000 // 5 seconds

		// declare the dead letter queue
		if _, err := ch.QueueDeclare(q.DLX(), true, false, false, false, dlqArgs); err != nil {
			t.l.Error().Msgf("cannot declare dead letter exchange/queue: %q, %v", q.DLX(), err)
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

	defer poolCh.Release()

	_, err = ch.QueueDelete(q.Name(), true, true, true)

	if err != nil {
		t.l.Error().Msgf("cannot delete queue: %q, %v", q.Name(), err)
		return err
	}

	if q.DLX() != "" {
		_, err = ch.QueueDelete(q.DLX(), true, true, true)

		if err != nil {
			t.l.Error().Msgf("cannot delete dead letter queue: %q, %v", q.DLX(), err)
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
	retryCount := 0

	wg := sync.WaitGroup{}
	var queueName string

	if !q.Exclusive() {
		poolCh, err := t.channels.Acquire(ctx)

		if err != nil {
			return nil, fmt.Errorf("cannot acquire channel for initializing queue: %v", err)
		}

		sub := poolCh.Value()

		// we initialize the queue here because exclusive queues are bound to the session/connection. however, it's not clear
		// if the exclusive queue will be available to the next session.
		queueName, err = t.initQueue(sub, q)

		if err != nil {
			poolCh.Release()
			return nil, fmt.Errorf("error initializing queue: %v", err)
		}

		poolCh.Release()
	}

	innerFn := func() error {
		poolCh, err := t.channels.Acquire(ctx)

		if err != nil {
			return err
		}

		sub := poolCh.Value()

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

	inner:
		for {
			select {
			case <-ctx.Done():
				return nil
			case rabbitMsg, ok := <-deliveries:
				if !ok {
					t.l.Info().Msg("deliveries channel closed")
					break inner
				}

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
						t.l.Error().Msgf("error in pre-ack: %v", err)

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
		}

		return nil
	}

	go func() {
		for {
			if ctx.Err() != nil {
				return
			}

			sessionCount++

			if err := innerFn(); err != nil {
				t.l.Error().Msgf("could not run inner loop: %v", err)
				sleepWithExponentialBackoff(10*time.Millisecond, 5*time.Second, retryCount)
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

// sleepWithExponentialBackoff sleeps for a duration calculated using exponential backoff and jitter,
// based on the retry count. The base sleep time and maximum sleep time are provided as inputs.
// retryCount determines the exponential backoff multiplier.
func sleepWithExponentialBackoff(base, max time.Duration, retryCount int) { // nolint: revive
	if retryCount < 0 {
		retryCount = 0
	}

	// Calculate exponential backoff
	backoff := base * (1 << retryCount)
	if backoff > max {
		backoff = max
	}

	// Apply jitter
	jitter := time.Duration(rand.Int63n(int64(backoff / 2))) // nolint: gosec
	sleepDuration := backoff/2 + jitter

	time.Sleep(sleepDuration)
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
