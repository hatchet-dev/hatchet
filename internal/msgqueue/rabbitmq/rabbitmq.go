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
	"github.com/jackc/puddle/v2"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/internal/telemetry"
	"github.com/hatchet-dev/hatchet/pkg/logger"
	"github.com/hatchet-dev/hatchet/pkg/random"
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

	// we use this to acknowledge the message
	ackChan chan<- ack

	// we use failed so we don't keep retrying a message that we have failed to send
	// this prevents phantom messages from being sent but also prevents us from trying to ack a message we have already marked as failed.
	failed      bool
	failedMutex sync.Mutex
}

type ack struct {
	e *error
}

// MessageQueueImpl implements MessageQueue interface using AMQP.
type MessageQueueImpl struct {
	ctx      context.Context
	sessions chan chan session
	msgs     chan *msgWithQueue
	identity string
	configFs []MessageQueueImplOpt

	qos int

	l *zerolog.Logger

	readyMux sync.Mutex
	ready    bool

	disableTenantExchangePubs bool

	// lru cache for tenant ids
	tenantIdCache *lru.Cache[string, bool]
}

func (t *MessageQueueImpl) setNotReady() {
	t.safeSetReady(false)
}

func (t *MessageQueueImpl) setReady() {
	t.safeSetReady(true)
}

func (t *MessageQueueImpl) safeCheckReady() bool {
	t.readyMux.Lock()
	defer t.readyMux.Unlock()
	return t.ready
}
func (t *MessageQueueImpl) safeSetReady(ready bool) {
	t.readyMux.Lock()
	defer t.readyMux.Unlock()
	t.ready = ready
}

func (t *MessageQueueImpl) IsReady() bool {
	return t.safeCheckReady()
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

	newLogger := opts.l.With().Str("service", "events-controller").Logger()
	opts.l = &newLogger

	t := &MessageQueueImpl{
		ctx:                       ctx,
		identity:                  identity(),
		l:                         opts.l,
		qos:                       opts.qos,
		configFs:                  fs,
		disableTenantExchangePubs: opts.disableTenantExchangePubs,
	}

	constructor := func(context.Context) (*amqp.Connection, error) {
		conn, err := amqp.Dial(opts.url)

		if err != nil {
			opts.l.Error().Msgf("cannot (re)dial: %v: %q", err, opts.url)
			return nil, err
		}

		return conn, nil
	}

	destructor := func(conn *amqp.Connection) {
		if !conn.IsClosed() {
			err := conn.Close()

			if err != nil {
				opts.l.Error().Msgf("error closing connection: %v", err)
			}
		}
	}

	maxPoolSize := int32(10)

	pool, err := puddle.NewPool(&puddle.Config[*amqp.Connection]{Constructor: constructor, Destructor: destructor, MaxSize: maxPoolSize})

	if err != nil {
		t.l.Error().Err(err).Msg("cannot create connection pool")
		cancel()
		return nil, nil
	}

	t.sessions = t.redial(ctx, opts.l, pool)
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

	// create publisher go func
	cleanupPub := t.startPublishing()

	cleanup := func() error {
		cancel()
		if err := cleanupPub(); err != nil {
			return fmt.Errorf("error cleaning up rabbitmq publisher: %w", err)
		}

		pool.Close()

		return nil
	}

	return cleanup, t
}

func (t *MessageQueueImpl) Clone() (func() error, msgqueue.MessageQueue) {
	return New(t.configFs...)
}

func (t *MessageQueueImpl) SetQOS(prefetchCount int) {
	t.qos = prefetchCount
}

// AddMessage adds a msg to the queue.
func (t *MessageQueueImpl) AddMessage(ctx context.Context, q msgqueue.Queue, msg *msgqueue.Message) error {
	ctx, span := telemetry.NewSpan(ctx, "add-message")
	defer span.End()

	// inject otel carrier into the message
	if msg.OtelCarrier == nil {
		msg.OtelCarrier = telemetry.GetCarrier(ctx)
	}

	// we can have multiple error acks
	ackC := make(chan ack, 5)

	msgWithQueue := msgWithQueue{
		Message: msg,
		q:       q,
		ackChan: ackC,
		failed:  false,
	}

	select {
	case t.msgs <- &msgWithQueue:
	case <-time.After(5 * time.Second):
		return fmt.Errorf("timeout sending message to %s queue", q.Name())
	case <-ctx.Done():
		return fmt.Errorf("failed to send message to queue %s: %w", q.Name(), ctx.Err())
	}

	var ackErr *error
	for {
		select {
		case ack := <-ackC:
			ackErr = ack.e
			if ackErr != nil {
				t.l.Err(*ackErr).Msg("error adding message")
			} else {
				return nil // success
			}
		case <-time.After(5 * time.Second):
			msgWithQueue.failedMutex.Lock()
			defer msgWithQueue.failedMutex.Unlock()
			msgWithQueue.failed = true

			if ackErr != nil {
				// we have additional context so lets send it
				t.l.Err(*ackErr).Msg("timeout adding message")
				return *ackErr
			}

			return fmt.Errorf("timeout adding message to %s queue", q.Name())
		case <-ctx.Done():
			msgWithQueue.failedMutex.Lock()
			defer msgWithQueue.failedMutex.Unlock()
			msgWithQueue.failed = true
			return fmt.Errorf("failed to add message to queue %s: %w", q.Name(), ctx.Err())

		}
	}
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
		args["x-consumer-timeout"] = 300000 // 5 minutes

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

			conn := pub.Connection

			t.l.Debug().Msgf("starting publisher: %s", conn.LocalAddr().String())

			for {
				if pub.Channel.IsClosed() {
					break
				} else if conn.IsClosed() {
					t.l.Error().Msgf("connection is closed, reconnecting")
					break
				}

				select {
				case <-ctx.Done():
					return
				case msg := <-t.msgs:
					go func(msg *msgWithQueue) {

						// we don't allow the message to be failed when we are processing it
						msg.failedMutex.Lock()
						defer msg.failedMutex.Unlock()

						if msg.failed {
							t.l.Warn().Msgf("message %s has already failed, not publishing", msg.ID)
							return
						}

						body, err := json.Marshal(msg)

						if err != nil {
							t.l.Error().Msgf("error marshaling msg queue: %v", err)
							return
						}

						ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
						defer cancel()

						ctx, span := telemetry.NewSpanWithCarrier(ctx, "publish-message", msg.OtelCarrier)
						defer span.End()

						t.l.Debug().Msgf("publishing msg %s to queue %s", msg.ID, msg.q.Name())

						pubMsg := amqp.Publishing{
							Body: body,
						}

						if msg.ImmediatelyExpire {
							pubMsg.Expiration = "0"
						}

						err = pub.PublishWithContext(ctx, "", msg.q.Name(), false, false, pubMsg)

						if err != nil {
							select {
							case msg.ackChan <- ack{e: &err}:
								t.msgs <- msg
							case <-time.After(100 * time.Millisecond):
								t.l.Error().Msgf("ack channel blocked for %s", msg.ID)
							}
							return
						}

						// if this is a tenant msg, publish to the tenant exchange
						if !t.disableTenantExchangePubs && msg.TenantID() != "" {
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
						select {
						case msg.ackChan <- ack{e: nil}:
							return
						case <-ctx.Done():
							return
						case <-time.After(5 * time.Second):
							t.l.Error().Msgf("timeout sending ack for message %s", msg.ID)
							return
						}

					}(msg)
				}
			}

			if !pub.Channel.IsClosed() {
				err := pub.Channel.Close()

				if err != nil {
					t.l.Error().Msgf("cannot close channel: %s, %v", conn.LocalAddr().String(), err)
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

			sessionWg := sync.WaitGroup{}

			conn := sub.Connection

			t.l.Debug().Msgf("starting subscriber %s on: %s", subId, conn.LocalAddr().String())

			// we initialize the queue here because exclusive queues are bound to the session/connection. however, it's not clear
			// if the exclusive queue will be available to the next session.
			queueName, err := t.initQueue(sub, q)

			if err != nil {
				return
			}

			// We'd like to limit to 1k TPS per engine. The max channels on an instance is 10.
			err = sub.Qos(t.qos, 0, false)

			if err != nil {
				t.l.Error().Msgf("cannot set qos: %v", err)
				return
			}

			deliveries, err := sub.Consume(queueName, subId, false, q.Exclusive(), false, false, nil)

			if err != nil {
				t.l.Error().Msgf("cannot consume from: %s, %v", queueName, err)
				return
			}

			closeChannel := func() {
				sessionWg.Wait()

				if !sub.Channel.IsClosed() {
					err = sub.Channel.Close()

					if err != nil {
						t.l.Error().Msgf("cannot close channel: %s, %v", conn.LocalAddr().String(), err)
					}
				}
			}

		inner:
			for {
				select {
				case <-ctx.Done():
					closeChannel()
					return
				case rabbitMsg, ok := <-deliveries:
					if !ok {
						t.l.Info().Msg("deliveries channel closed")
						break inner
					}

					wg.Add(1)
					sessionWg.Add(1)

					go func(rabbitMsg amqp.Delivery) {
						defer wg.Done()
						defer sessionWg.Done()

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

			go closeChannel()
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

func (t *MessageQueueImpl) redial(ctx context.Context, l *zerolog.Logger, pool *puddle.Pool[*amqp.Connection]) chan chan session {
	sessions := make(chan chan session)

	go func() {
		defer close(sessions)

		for {
			sess := make(chan session)

			select {
			case sessions <- sess:
			case <-ctx.Done():
				l.Info().Msg("shutting down session factory")
				return
			}

			newSession, err := t.establishSessionWithRetry(ctx, l, pool)
			if err != nil {
				l.Error().Msg("failed to establish session after retries")
				t.setNotReady()

				return
			}

			t.setReady()
			t.monitorSession(ctx, l, newSession)

			select {
			case sess <- newSession:
			case <-ctx.Done():
				l.Info().Msg("shutting down new session")
				return
			}
		}
	}()

	return sessions
}

func (t *MessageQueueImpl) establishSessionWithRetry(ctx context.Context, l *zerolog.Logger, pool *puddle.Pool[*amqp.Connection]) (session, error) {
	var newSession session
	var err error

	for i := 0; i < MAX_RETRY_COUNT; i++ {
		newSession, err = getSession(ctx, l, pool)
		if err == nil {
			if i > 0 {
				l.Info().Msgf("re-established session after %d attempts", i)
			}
			return newSession, nil
		}

		l.Error().Msgf("error getting session (attempt %d): %v", i+1, err)
		time.Sleep(RETRY_INTERVAL)
	}

	return session{}, fmt.Errorf("failed to establish session after %d retries", MAX_RETRY_COUNT)
}

func (t *MessageQueueImpl) monitorSession(ctx context.Context, l *zerolog.Logger, newSession session) {
	closeCh := newSession.Connection.NotifyClose(make(chan *amqp.Error, 1))

	go func() {
		select {
		case <-ctx.Done():
		case <-closeCh:
			l.Warn().Msg("session closed, marking as not ready")
			t.setNotReady()
		}
	}()
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

func getSession(ctx context.Context, l *zerolog.Logger, pool *puddle.Pool[*amqp.Connection]) (session, error) {
	connFromPool, err := pool.Acquire(ctx)

	if err != nil {
		l.Error().Msgf("cannot acquire connection: %v", err)
		return session{}, err
	}

	conn := connFromPool.Value()

	ch, err := conn.Channel()

	if err != nil {
		connFromPool.Destroy()
		l.Error().Msgf("cannot create channel: %v", err)
		return session{}, err
	}

	connFromPool.Release()

	return session{
		Channel:    ch,
		Connection: conn,
	}, nil
}
