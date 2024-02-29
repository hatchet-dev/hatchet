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
	"github.com/hatchet-dev/hatchet/internal/taskqueue"
)

// session composes an amqp.Connection with an amqp.Channel
type session struct {
	*amqp.Connection
	*amqp.Channel
}

type taskWithQueue struct {
	*taskqueue.Task

	q taskqueue.Queue
}

// TaskQueueImpl implements TaskQueue interface using AMQP.
type TaskQueueImpl struct {
	ctx      context.Context
	sessions chan chan session
	tasks    chan *taskWithQueue
	identity string

	l *zerolog.Logger

	ready bool

	// lru cache for tenant ids
	tenantIdCache *lru.Cache[string, bool]
}

func (t *TaskQueueImpl) IsReady() bool {
	return t.ready
}

type TaskQueueImplOpt func(*TaskQueueImplOpts)

type TaskQueueImplOpts struct {
	l   *zerolog.Logger
	url string
}

func defaultTaskQueueImplOpts() *TaskQueueImplOpts {
	l := logger.NewDefaultLogger("rabbitmq")

	return &TaskQueueImplOpts{
		l: &l,
	}
}

func WithLogger(l *zerolog.Logger) TaskQueueImplOpt {
	return func(opts *TaskQueueImplOpts) {
		opts.l = l
	}
}

func WithURL(url string) TaskQueueImplOpt {
	return func(opts *TaskQueueImplOpts) {
		opts.url = url
	}
}

// New creates a new TaskQueueImpl.
func New(fs ...TaskQueueImplOpt) (func() error, *TaskQueueImpl) {
	ctx, cancel := context.WithCancel(context.Background())

	opts := defaultTaskQueueImplOpts()

	for _, f := range fs {
		f(opts)
	}

	newLogger := opts.l.With().Str("service", "events-controller").Logger()
	opts.l = &newLogger

	t := &TaskQueueImpl{
		ctx:      ctx,
		identity: identity(),
		l:        opts.l,
	}

	t.sessions = t.redial(ctx, opts.l, opts.url)
	t.tasks = make(chan *taskWithQueue)

	// create a new lru cache for tenant ids
	t.tenantIdCache, _ = lru.New[string, bool](2000) // nolint: errcheck - this only returns an error if the size is less than 0

	// init the queues in a blocking fashion
	sub := <-<-t.sessions
	if _, err := t.initQueue(sub, taskqueue.EVENT_PROCESSING_QUEUE); err != nil {
		t.l.Debug().Msgf("error initializing queue: %v", err)
		cancel()
		return nil, nil
	}

	if _, err := t.initQueue(sub, taskqueue.JOB_PROCESSING_QUEUE); err != nil {
		t.l.Debug().Msgf("error initializing queue: %v", err)
		cancel()
		return nil, nil
	}

	if _, err := t.initQueue(sub, taskqueue.WORKFLOW_PROCESSING_QUEUE); err != nil {
		t.l.Debug().Msgf("error initializing queue: %v", err)
		cancel()
		return nil, nil
	}

	if _, err := t.initQueue(sub, taskqueue.SCHEDULING_QUEUE); err != nil {
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

// AddTask adds a task to the queue.
func (t *TaskQueueImpl) AddTask(ctx context.Context, q taskqueue.Queue, task *taskqueue.Task) error {
	t.tasks <- &taskWithQueue{
		Task: task,
		q:    q,
	}

	return nil
}

// Subscribe subscribes to the task queue.
func (t *TaskQueueImpl) Subscribe(q taskqueue.Queue) (func() error, <-chan *taskqueue.Task, error) {
	t.l.Debug().Msgf("subscribing to queue: %s", q.Name())

	tasks := make(chan *taskqueue.Task)
	cleanup := t.subscribe(t.identity, q, t.sessions, t.tasks, tasks)
	return cleanup, tasks, nil
}

func (t *TaskQueueImpl) RegisterTenant(ctx context.Context, tenantId string) error {
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

func (t *TaskQueueImpl) initQueue(sub session, q taskqueue.Queue) (string, error) {
	name := q.Name()

	if q.FanoutExchangeKey() != "" {
		suffix, err := encryption.GenerateRandomBytes(4)

		if err != nil {
			t.l.Error().Msgf("error generating random bytes: %v", err)
			return "", err
		}

		name = fmt.Sprintf("%s-%s", q.Name(), suffix)
	}

	if _, err := sub.QueueDeclare(name, q.Durable(), q.AutoDeleted(), q.Exclusive(), false, nil); err != nil {
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

func (t *TaskQueueImpl) startPublishing() func() error {
	ctx, cancel := context.WithCancel(t.ctx)

	cleanup := func() error {
		cancel()
		return nil
	}

	go func() {
		for session := range t.sessions {
			pub := <-session

			for {
				select {
				case <-ctx.Done():
					return
				case task := <-t.tasks:
					go func(task *taskWithQueue) {
						body, err := json.Marshal(task)

						if err != nil {
							t.l.Error().Msgf("error marshaling task queue: %v", err)
							return
						}

						ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
						defer cancel()

						t.l.Debug().Msgf("publishing task %s to queue %s", task.ID, task.q.Name())

						err = pub.PublishWithContext(ctx, "", task.q.Name(), false, false, amqp.Publishing{
							Body: body,
						})

						// TODO: retry failed delivery on the next session
						if err != nil {
							t.l.Error().Msgf("error publishing task: %v", err)
							return
						}

						// if this is a tenant task, publish to the tenant exchange
						if task.TenantID() != "" {
							// determine if the tenant exchange exists
							if _, ok := t.tenantIdCache.Get(task.TenantID()); !ok {
								// register the tenant exchange
								err = t.RegisterTenant(ctx, task.TenantID())

								if err != nil {
									t.l.Error().Msgf("error registering tenant exchange: %v", err)
									return
								}
							}

							t.l.Debug().Msgf("publishing tenant task %s to exchange %s", task.ID, task.TenantID())

							err = pub.PublishWithContext(ctx, task.TenantID(), "", false, false, amqp.Publishing{
								Body: body,
							})

							if err != nil {
								t.l.Error().Msgf("error publishing tenant task: %v", err)
								return
							}
						}

						t.l.Debug().Msgf("published task %s to queue %s", task.ID, task.q.Name())
					}(task)
				}
			}
		}
	}()

	return cleanup
}

func (t *TaskQueueImpl) subscribe(subId string, q taskqueue.Queue, sessions chan chan session, messages chan *taskWithQueue, tasks chan<- *taskqueue.Task) func() error {
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

			deliveries, err := sub.Consume(queueName, subId, false, q.Exclusive(), false, false, nil)

			if err != nil {
				t.l.Error().Msgf("cannot consume from: %s, %v", queueName, err)
				return
			}

			for {
				select {
				case msg := <-deliveries:
					wg.Add(1)
					go func(msg amqp.Delivery) {
						defer wg.Done()
						task := &taskWithQueue{}

						if err := json.Unmarshal(msg.Body, task); err != nil {
							t.l.Error().Msgf("error unmarshaling message: %v", err)
							return
						}

						t.l.Debug().Msgf("(session: %d) got task: %v", sessionCount, task.ID)

						tasks <- task.Task

						if err := sub.Ack(msg.DeliveryTag, false); err != nil {
							t.l.Error().Msgf("error acknowledging message: %v", err)
							return
						}
					}(msg)
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	cleanup := func() error {
		cancel()

		t.l.Debug().Msgf("shutting down subscriber: %s", subId)
		wg.Wait()
		close(tasks)
		t.l.Debug().Msgf("successfully shut down subscriber: %s", subId)
		return nil
	}

	return cleanup
}

// redial continually connects to the URL, exiting the program when no longer possible
func (t *TaskQueueImpl) redial(ctx context.Context, l *zerolog.Logger, url string) chan chan session {
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

			newSession, err := getSession(ctx, l, url)
			if err != nil {
				l.Error().Msgf("error getting session: %v", err)
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
