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

	"github.com/hatchet-dev/hatchet/internal/encryption"
	"github.com/hatchet-dev/hatchet/internal/logger"
	"github.com/hatchet-dev/hatchet/internal/msgqueue"
)

const MAX_RETRY_COUNT = 15
const RETRY_INTERVAL = 2 * time.Second

// session composes an amqp.Connection with an amqp.Channel
type session struct {
	*amqp.Connection
	*amqp.Channel
}

type msgWithQueue struct {
	*msgqueue.Message

	q msgqueue.Queue
}

// MessageQueueImpl implements MessageQueue interface using AMQP.
type MessageQueueImpl struct {
	ctx      context.Context
	sessions chan chan session
	msgs     chan *msgWithQueue
	identity string

	l *zerolog.Logger

	ready bool

	// lru cache for tenant ids
	tenantIdCache *lru.Cache[string, bool]
}

func (t *MessageQueueImpl) IsReady() bool {
	return t.ready
}

type MessageQueueImplOpt func(*MessageQueueImplOpts)

type MessageQueueImplOpts struct {
	l   *zerolog.Logger
	url string
}

func defaultMessageQueueImplOpts() *MessageQueueImplOpts {
	l := logger.NewDefaultLogger("rabbitmq")

	return &MessageQueueImplOpts{
		l: &l,
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

// New creates a new MessageQueueImpl.
func New(fs ...MessageQueueImplOpt) (func() error, *MessageQueueImpl) {
	ctx, cancel := context.WithCancel(context.Background())

	opts := defaultMessageQueueImplOpts()

	for _, f := range fs {
		f(opts)
	}

	newLogger := opts.l.With().Str("service", "events-controller").Logger()
	opts.l = &newLogger

	t := &MessageQueueImpl{
		ctx:      ctx,
		identity: identity(),
		l:        opts.l,
	}

	t.sessions = t.redial(ctx, opts.l, opts.url)
	t.msgs = make(chan *msgWithQueue)

	// create a new lru cache for tenant ids
	t.tenantIdCache, _ = lru.New[string, bool](2000) // nolint: errcheck - this only returns an error if the size is less than 0

	// init the queues in a blocking fashion
	sub := <-<-t.sessions
	if _, err := t.initQueue(sub, msgqueue.EVENT_PROCESSING_QUEUE); err != nil {
		t.l.Debug().Msgf("error initializing queue: %v", err)
		cancel()
		return nil, nil
	}

	if _, err := t.initQueue(sub, msgqueue.JOB_PROCESSING_QUEUE); err != nil {
		t.l.Debug().Msgf("error initializing queue: %v", err)
		cancel()
		return nil, nil
	}

	if _, err := t.initQueue(sub, msgqueue.WORKFLOW_PROCESSING_QUEUE); err != nil {
		t.l.Debug().Msgf("error initializing queue: %v", err)
		cancel()
		return nil, nil
	}

	if _, err := t.initQueue(sub, msgqueue.SCHEDULING_QUEUE); err != nil {
		t.l.Debug().Msgf("error initializing queue: %v", err)
		cancel()
		return nil, nil
	}

	// create publisher go func
	cleanup1 := t.startPublishing()

	cleanup := func() error {
		cancel()
		if err := cleanup1(); err != nil {
			return fmt.Errorf("error cleaning up rabbitmq publisher: %w", err)
		}
		return nil
	}

	return cleanup, t
}

// AddMessage adds a msg to the queue.
func (t *MessageQueueImpl) AddMessage(ctx context.Context, q msgqueue.Queue, msg *msgqueue.Message) error {
	t.msgs <- &msgWithQueue{
		Message: msg,
		q:       q,
	}

	return nil
}

// Subscribe subscribes to the msg queue.
func (t *MessageQueueImpl) Subscribe(
	q msgqueue.Queue,
	preAck msgqueue.AckHook,
	postAck msgqueue.AckHook,
) (func() error, error) {
	t.l.Debug().Msgf("subscribing to queue: %s", q.Name())

	cleanup := t.subscribe(t.identity, q, t.sessions, preAck, postAck)
	return cleanup, nil
}

func (t *MessageQueueImpl) RegisterTenant(ctx context.Context, tenantId string) error {
	// create a new fanout exchange for the tenant
	sub := <-<-t.sessions

	t.l.Debug().Msgf("registering tenant exchange: %s", tenantId)

	// create a fanout exchange for the tenant. each consumer of the fanout exchange will get notified
	// with the tenant events.
	err := sub.ExchangeDeclare(
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

func (t *MessageQueueImpl) initQueue(sub session, q msgqueue.Queue) (string, error) {
	args := make(amqp.Table)
	name := q.Name()

	if q.FanoutExchangeKey() != "" {
		suffix, err := encryption.GenerateRandomBytes(4)

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
		if _, err := sub.QueueDeclare(q.DLX(), true, false, false, false, dlqArgs); err != nil {
			t.l.Error().Msgf("cannot declare dead letter exchange/queue: %q, %v", q.DLX(), err)
			return "", err
		}
	}

	if _, err := sub.QueueDeclare(name, q.Durable(), q.AutoDeleted(), q.Exclusive(), false, args); err != nil {
		t.l.Error().Msgf("cannot declare queue: %q, %v", name, err)
		return "", err
	}

	// if the queue has a subscriber key, bind it to the fanout exchange
	if q.FanoutExchangeKey() != "" {
		t.l.Debug().Msgf("binding queue: %s to exchange: %s", name, q.FanoutExchangeKey())

		if err := sub.QueueBind(name, "", q.FanoutExchangeKey(), false, nil); err != nil {
			t.l.Error().Msgf("cannot bind queue: %q, %v", name, err)
			return "", err
		}
	}

	return name, nil
}

func (t *MessageQueueImpl) startPublishing() func() error {
	ctx, cancel := context.WithCancel(t.ctx)

	cleanup := func() error {
		cancel()
		return nil
	}

	go func() {
		for session := range t.sessions {
			pub := <-session

			for {
				if pub.Channel.IsClosed() || pub.Connection.IsClosed() {
					break
				}

				select {
				case <-ctx.Done():
					return
				case msg := <-t.msgs:
					go func(msg *msgWithQueue) {
						body, err := json.Marshal(msg)

						if err != nil {
							t.l.Error().Msgf("error marshaling msg queue: %v", err)
							return
						}

						ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
						defer cancel()

						t.l.Debug().Msgf("publishing msg %s to queue %s", msg.ID, msg.q.Name())

						err = pub.PublishWithContext(ctx, "", msg.q.Name(), false, false, amqp.Publishing{
							Body: body,
						})

						// retry failed delivery on the next session
						if err != nil {
							t.msgs <- msg
							return
						}

						// if this is a tenant msg, publish to the tenant exchange
						if msg.TenantID() != "" {
							// determine if the tenant exchange exists
							if _, ok := t.tenantIdCache.Get(msg.TenantID()); !ok {
								// register the tenant exchange
								err = t.RegisterTenant(ctx, msg.TenantID())

								if err != nil {
									t.l.Error().Msgf("error registering tenant exchange: %v", err)
									return
								}
							}

							t.l.Debug().Msgf("publishing tenant msg %s to exchange %s", msg.ID, msg.TenantID())

							err = pub.PublishWithContext(ctx, msg.TenantID(), "", false, false, amqp.Publishing{
								Body: body,
							})

							if err != nil {
								t.l.Error().Msgf("error publishing tenant msg: %v", err)
								return
							}
						}

						t.l.Debug().Msgf("published msg %s to queue %s", msg.ID, msg.q.Name())
					}(msg)
				}
			}
		}
	}()

	return cleanup
}

func (t *MessageQueueImpl) subscribe(
	subId string,
	q msgqueue.Queue,
	sessions chan chan session,
	preAck msgqueue.AckHook,
	postAck msgqueue.AckHook,
) func() error {
	ctx, cancel := context.WithCancel(context.Background())

	sessionCount := 0

	wg := sync.WaitGroup{}

	go func() {
		for session := range sessions {
			sessionCount++
			sub := <-session

			// we initialize the queue here because exclusive queues are bound to the session/connection. however, it's not clear
			// if the exclusive queue will be available to the next session.
			queueName, err := t.initQueue(sub, q)

			if err != nil {
				return
			}

			// We'd like to limit to 1k TPS per engine. The max channels on an instance is 10.
			err = sub.Qos(100, 0, false)

			if err != nil {
				t.l.Error().Msgf("cannot set qos: %v", err)
				return
			}

			deliveries, err := sub.Consume(queueName, subId, false, q.Exclusive(), false, false, nil)

			if err != nil {
				t.l.Error().Msgf("cannot consume from: %s, %v", queueName, err)
				return
			}

		inner:
			for {
				select {
				case <-ctx.Done():
					return
				case rabbitMsg, ok := <-deliveries:
					if !ok {
						t.l.Info().Msg("deliveries channel closed")
						break inner
					}

					wg.Add(1)

					go func(rabbitMsg amqp.Delivery) {
						defer wg.Done()
						msg := &msgWithQueue{}

						if len(rabbitMsg.Body) == 0 {
							t.l.Error().Msgf("empty message body for message: %s", rabbitMsg.MessageId)

							// reject this message
							if err := rabbitMsg.Reject(false); err != nil {
								t.l.Error().Msgf("error rejecting message: %v", err)
							}

							return
						}

						if err := json.Unmarshal(rabbitMsg.Body, msg); err != nil {
							t.l.Error().Msgf("error unmarshaling message: %v", err)

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

							t.l.Debug().Msgf("message %s has been rejected %d times", msg.ID, deathCount)

							if deathCount > int64(msg.Retries) {
								t.l.Debug().Msgf("message %s has been rejected %d times, not requeuing", msg.ID, deathCount)

								// acknowledge so it's removed from the queue
								if err := rabbitMsg.Ack(false); err != nil {
									t.l.Error().Msgf("error acknowledging message: %v", err)
								}

								return
							}
						}

						t.l.Debug().Msgf("(session: %d) got msg: %v", sessionCount, msg.ID)

						if err := preAck(msg.Message); err != nil {
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

						if err := postAck(msg.Message); err != nil {
							t.l.Error().Msgf("error in post-ack: %v", err)
							return
						}
					}(rabbitMsg)
				}
			}

			err = sub.CloseDeadline(time.Now())

			if err != nil {
				t.l.Error().Msgf("cannot close session: %s, %v", sub.LocalAddr().String(), err)
			}
		}
	}()

	cleanup := func() error {
		cancel()

		t.l.Debug().Msgf("shutting down subscriber: %s", subId)
		wg.Wait()
		t.l.Debug().Msgf("successfully shut down subscriber: %s", subId)
		return nil
	}

	return cleanup
}

// redial continually connects to the URL, exiting the program when no longer possible
func (t *MessageQueueImpl) redial(ctx context.Context, l *zerolog.Logger, url string) chan chan session {
	sessions := make(chan chan session)

	go func() {
		sess := make(chan session)
		defer close(sessions)

		for {
			select {
			case sessions <- sess:
			case <-ctx.Done():
				l.Info().Msgf("shutting down session factory")
				return
			}

			var newSession session
			var err error

			for i := 0; i < MAX_RETRY_COUNT; i++ {
				newSession, err = getSession(ctx, l, url)
				if err == nil {
					if i > 0 {
						l.Info().Msgf("re-established session after %d attempts", i)
					}

					break
				}

				l.Error().Msgf("error getting session (attempt %d): %v", i+1, err)
				time.Sleep(RETRY_INTERVAL)
			}

			if err != nil {
				l.Error().Msgf("failed to get session after %d attempts", MAX_RETRY_COUNT)
				return
			}

			t.ready = true

			ch := newSession.Connection.NotifyClose(make(chan *amqp.Error, 1))

			go func() {
				select {
				case <-ctx.Done():
					return
				case <-ch:
					t.ready = false
				}
			}()

			select {
			case sess <- newSession:
			case <-ctx.Done():
				l.Info().Msgf("shutting down new session")
				return
			}
		}
	}()

	return sessions
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

func getSession(ctx context.Context, l *zerolog.Logger, url string) (session, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		l.Error().Msgf("cannot (re)dial: %v: %q", err, url)
		return session{}, err
	}

	ch, err := conn.Channel()
	if err != nil {
		l.Error().Msgf("cannot create channel: %v", err)
		return session{}, err
	}

	return session{conn, ch}, nil
}
