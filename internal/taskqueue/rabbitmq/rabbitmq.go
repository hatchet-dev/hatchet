package rabbitmq

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/internal/logger"
	"github.com/hatchet-dev/hatchet/internal/taskqueue"
)

// session composes an amqp.Connection with an amqp.Channel
type session struct {
	*amqp.Connection
	*amqp.Channel
}

// TaskQueueImpl implements TaskQueue interface using AMQP.
type TaskQueueImpl struct {
	ctx      context.Context
	sessions chan chan session
	tasks    chan *taskqueue.Task
	identity string

	l *zerolog.Logger
}

type TaskQueueImplOpt func(*TaskQueueImplOpts)

type TaskQueueImplOpts struct {
	l   *zerolog.Logger
	url string
}

func defaultTaskQueueImplOpts() *TaskQueueImplOpts {
	logger := logger.NewDefaultLogger("rabbitmq")

	return &TaskQueueImplOpts{
		l: &logger,
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
func New(ctx context.Context, fs ...TaskQueueImplOpt) *TaskQueueImpl {
	opts := defaultTaskQueueImplOpts()

	for _, f := range fs {
		f(opts)
	}

	newLogger := opts.l.With().Str("service", "events-controller").Logger()
	opts.l = &newLogger

	sessions := redial(ctx, opts.l, opts.url)
	tasks := make(chan *taskqueue.Task)

	t := &TaskQueueImpl{
		ctx:      ctx,
		sessions: sessions,
		tasks:    tasks,
		identity: identity(),
		l:        opts.l,
	}

	// init the queues in a blocking fashion
	for session := range sessions {
		sub := <-session

		t.initQueue(sub, string(taskqueue.EVENT_PROCESSING_QUEUE))
		t.initQueue(sub, string(taskqueue.JOB_PROCESSING_QUEUE))
		t.initQueue(sub, string(taskqueue.SCHEDULING_QUEUE))
		break
	}

	// create publisher go func
	go func() {
		t.publish()
	}()

	return t
}

// AddTask adds a task to the queue.
func (t *TaskQueueImpl) AddTask(ctx context.Context, queue taskqueue.QueueType, task *taskqueue.Task) error {
	t.tasks <- task
	return nil
}

// Subscribe subscribes to the task queue.
func (t *TaskQueueImpl) Subscribe(ctx context.Context, queueType taskqueue.QueueType) (<-chan *taskqueue.Task, error) {
	t.l.Debug().Msgf("subscribed to queue: %s", string(queueType))

	// init the queues in a blocking fashion
	for session := range t.sessions {
		sub := <-session

		t.initQueue(sub, string(queueType))
		break
	}

	tasks := make(chan *taskqueue.Task)
	go t.subscribe(ctx, t.identity, string(queueType), t.sessions, t.tasks, tasks)
	return tasks, nil
}

func (t *TaskQueueImpl) initQueue(sub session, name string) {
	// amqp.Table(map[string]interface{}{
	// 	"x-dead-letter-exchange": name,
	// }

	if _, err := sub.QueueDeclare(name, true, false, false, false, nil); err != nil {
		t.l.Error().Msgf("cannot declare queue: %q, %v", name, err)
		return
	}
}

func (t *TaskQueueImpl) publish() {
	for session := range t.sessions {
		pub := <-session

		for task := range t.tasks {
			go func(task *taskqueue.Task) {
				body, err := json.Marshal(task)

				if err != nil {
					t.l.Error().Msgf("error marshaling task queue: %v", err)
					return
				}

				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				t.l.Debug().Msgf("publishing task %s to queue %s", task.ID, string(task.Queue))

				err = pub.PublishWithContext(ctx, "", string(task.Queue), false, false, amqp.Publishing{
					Body: body,
				})

				// TODO: retry failed delivery on the next session
				if err != nil {
					t.l.Error().Msgf("error publishing task: %v", err)
					return
				}
			}(task)
		}
	}
}

// func (t *TaskQueueImpl) publish() {
// 	for session := range t.sessions {
// 		var (
// 			running bool
// 			reading = t.tasks
// 			pending = make(chan []byte, 1)
// 			confirm = make(chan amqp.Confirmation, 1)
// 		)

// 		pub := <-session

// 		// publisher confirms for this channel/connection
// 		if err := pub.Channel.Confirm(false); err != nil {
// 			t.l.Info().Msgf("publisher confirms not supported")
// 			close(confirm) // confirms not supported, simulate by always nacking
// 		} else {
// 			pub.NotifyPublish(confirm)
// 		}

// 		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
// 		defer cancel()

// 	Publish:
// 		for {
// 			var (
// 				body []byte
// 				task *taskqueue.Task
// 			)

// 			select {
// 			case confirmed, ok := <-confirm:
// 				if !ok {
// 					break Publish
// 				}
// 				if !confirmed.Ack {
// 					t.l.Info().Msgf("nack message %d", confirmed.DeliveryTag)
// 				}
// 				reading = t.tasks

// 			case body = <-pending:
// 				err := pub.PublishWithContext(ctx, "", string(task.Queue), false, false, amqp.Publishing{
// 					Body: body,
// 				})
// 				// Retry failed delivery on the next session
// 				if err != nil {
// 					pending <- body
// 					pub.Channel.Close()
// 					break Publish
// 				}
// 			case task, running = <-reading:
// 				body, err := json.Marshal(task)

// 				if err != nil {
// 					t.l.Error().Msgf("error marshaling task queue: %v", err)
// 					return
// 				}

// 				// all messages consumed
// 				if !running {
// 					return
// 				}

// 				// work on pending delivery until ack'd
// 				pending <- body
// 				reading = nil
// 			}
// 		}
// 	}
// }

func (t *TaskQueueImpl) subscribe(ctx context.Context, subId, queue string, sessions chan chan session, messages chan *taskqueue.Task, tasks chan<- *taskqueue.Task) {
	sessionCount := 0

	for session := range sessions {
		sessionCount++
		sub := <-session

		deliveries, err := sub.Consume(queue, subId, false, false, false, false, nil)

		if err != nil {
			t.l.Error().Msgf("cannot consume from: %q, %v", queue, err)
			return
		}

		for msg := range deliveries {
			go func(msg amqp.Delivery) {
				task := &taskqueue.Task{}

				if err := json.Unmarshal(msg.Body, task); err != nil {
					t.l.Error().Msgf("error unmarshaling message: %v", err)
					return
				}

				t.l.Debug().Msgf("(session: %d) got task: %v", sessionCount, task.ID)

				tasks <- task

				if err := sub.Ack(msg.DeliveryTag, false); err != nil {
					t.l.Error().Msgf("error acknowledging message: %v", err)
					return
				}
			}(msg)
		}
	}
}

// redial continually connects to the URL, exiting the program when no longer possible
func redial(ctx context.Context, l *zerolog.Logger, url string) chan chan session {
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
	fmt.Fprint(h, hostname)
	fmt.Fprint(h, err)
	fmt.Fprint(h, os.Getpid())
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
