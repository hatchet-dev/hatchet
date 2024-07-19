package pgqueue

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/lib/pq"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/pkg/logger"
	"github.com/hatchet-dev/hatchet/pkg/random"
)

type msgWithQueue struct {
	*msgqueue.Message

	q msgqueue.Queue
}

// MessageQueueImpl implements MessageQueue interface using AMQP.
type MessageQueueImpl struct {
	ctx      context.Context
	listener *pq.Listener
	db       *sql.DB
	msgs     chan *msgWithQueue
	identity string

	l *zerolog.Logger

	ready bool

	// lru cache for tenant ids
	tenantIdCache *lru.Cache[string, bool]

	channels map[string]chan *pq.Notification
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
	l := logger.NewDefaultLogger("pgqueue")

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

	var conninfo = "user=hatchet password=hatchet dbname=hatchet sslmode=disable host=localhost port=5431"

	db, err := sql.Open("postgres", conninfo)
	if err != nil {
		panic(err)
	}

	reportProblem := func(ev pq.ListenerEventType, err error) {
		if err != nil {
			fmt.Println(err.Error())
		}
	}

	minReconn := 10 * time.Second
	maxReconn := time.Minute

	t.channels = make(map[string]chan *pq.Notification)

	t.listener = pq.NewListener(conninfo, minReconn, maxReconn, reportProblem)
	t.db = db

	t.msgs = make(chan *msgWithQueue)

	// create a new lru cache for tenant ids
	t.tenantIdCache, _ = lru.New[string, bool](2000) // nolint: errcheck - this only returns an error if the size is less than 0

	// init the queues in a blocking fashion
	if _, err := t.initQueue(msgqueue.EVENT_PROCESSING_QUEUE); err != nil {
		t.l.Debug().Msgf("error initializing queue: %v", err)
		cancel()
		return nil, nil
	}

	if _, err := t.initQueue(msgqueue.JOB_PROCESSING_QUEUE); err != nil {
		t.l.Debug().Msgf("error initializing queue: %v", err)
		cancel()
		return nil, nil
	}

	if _, err := t.initQueue(msgqueue.WORKFLOW_PROCESSING_QUEUE); err != nil {
		t.l.Debug().Msgf("error initializing queue: %v", err)
		cancel()
		return nil, nil
	}

	if _, err := t.initQueue(msgqueue.SCHEDULING_QUEUE); err != nil {
		t.l.Debug().Msgf("error initializing queue: %v", err)
		cancel()
		return nil, nil
	}

	// create publisher go func
	cleanup1 := t.startPublishing()
	cleanup2 := t.startListening()

	cleanup := func() error {
		cancel()
		if err := cleanup1(); err != nil {
			return fmt.Errorf("error cleaning up pg publisher: %w", err)
		}
		if err := cleanup2(); err != nil {
			return fmt.Errorf("error cleaning up pg listener: %w", err)
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

	cleanup := t.subscribe(t.identity, q, preAck, postAck)
	return cleanup, nil
}

func (t *MessageQueueImpl) RegisterTenant(ctx context.Context, tenantId string) error {
	// create a new fanout exchange for the tenant

	tID := "queue_" + strings.ReplaceAll(tenantId, "-", "_")

	if _, ok := t.channels[tID]; !ok {
		t.channels[tID] = make(chan *pq.Notification, 9999999)
	}

	if err := t.listener.Listen(tID); err != nil && !errors.Is(err, pq.ErrChannelAlreadyOpen) {
		return fmt.Errorf("error listening to queue: %w", err)
	}

	t.tenantIdCache.Add(tenantId, true)

	return nil
}

func (t *MessageQueueImpl) initQueue(q msgqueue.Queue) (string, error) {
	name := "queue_" + strings.ReplaceAll(q.Name(), "-", "_")

	if q.FanoutExchangeKey() != "" {
		suffix, err := random.Generate(8)

		if err != nil {
			t.l.Error().Msgf("error generating random bytes: %v", err)
			return "", err
		}

		name = fmt.Sprintf("%s-%s", name, suffix)
	}

	if q.DLX() != "" {
		// TODO
		t.l.Debug().Msgf("binding DLX queue: %s to exchange: %s", name, q.DLX())
	}

	// if the queue has a subscriber key, bind it to the fanout exchange
	if q.FanoutExchangeKey() != "" {
		t.l.Debug().Msgf("binding queue: %s to exchange: %s", name, q.FanoutExchangeKey())
		panic("unimplemented")
	}

	if _, ok := t.channels[name]; !ok {
		t.channels[name] = make(chan *pq.Notification, 9999999)
	}

	if err := t.listener.Listen(name); err != nil && !errors.Is(err, pq.ErrChannelAlreadyOpen) {
		return "", fmt.Errorf("error listening to queue: %w", err)
	}

	return name, nil
}

func (t *MessageQueueImpl) startListening() func() error {
	ctx, cancel := context.WithCancel(t.ctx)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case msg := <-t.listener.Notify:
				var sent bool

				for i, ch := range t.channels {
					if msg.Channel == i {
						ch <- msg
						sent = true
						break
					}
				}

				if !sent {
					t.l.Warn().Msgf("message not sent to any channel: %s with payload %+v", msg.Channel, msg)
				}
			}
		}
	}()

	cleanup := func() error {
		cancel()

		for _, ch := range t.channels {
			close(ch)
		}

		return nil
	}

	return cleanup
}

func (t *MessageQueueImpl) startPublishing() func() error {
	ctx, cancel := context.WithCancel(t.ctx)

	cleanup := func() error {
		cancel()
		return nil
	}

	go func() {
		for {
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

					if err := t.notify(msg.q.Name(), string(body)); err != nil {
						t.l.Error().Msgf("error publishing msg: %v", err)
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

						if err := t.notify(msg.TenantID(), string(body)); err != nil {
							t.l.Error().Msgf("error publishing msg: %v", err)
							return
						}

						if err != nil {
							t.l.Error().Msgf("error publishing tenant msg: %v", err)
							return
						}
					}

					t.l.Debug().Msgf("published msg %s to queue %s", msg.ID, msg.q.Name())
				}(msg)
			}
		}
	}()

	return cleanup
}

func (t *MessageQueueImpl) subscribe(
	subId string,
	q msgqueue.Queue,
	preAck msgqueue.AckHook,
	postAck msgqueue.AckHook,
) func() error {
	ctx, cancel := context.WithCancel(context.Background())

	name, err := t.initQueue(q)
	if err != nil {
		panic(err) // TODO
	}

	sessionCount := 0

	wg := sync.WaitGroup{}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-t.channels[name]:
				if !ok {
					t.l.Warn().Msgf("channel %s not found", name)
					continue
				}
				if msg.Channel != name {
					t.l.Warn().Msgf("received unamtched message! %s vs name %s with payload %s", msg.Channel, name, msg.Extra)
					continue
				}

				wg.Add(1)

				go func(orig *pq.Notification) {
					defer wg.Done()

					msg := &msgWithQueue{}

					if len(orig.Extra) == 0 {
						t.l.Error().Msgf("empty message body for message: %s", orig.Channel)

						return
					}

					if err := json.Unmarshal([]byte(orig.Extra), msg); err != nil {
						t.l.Error().Msgf("error unmarshaling message: %v", err)

						return
					}

					t.l.Debug().Msgf("(session: %d) got msg: %v", sessionCount, msg.ID)

					if err := preAck(msg.Message); err != nil {
						t.l.Error().Err(err).Msgf("error in pre-ack: %v", err)
						return
					}

					// TODO pre-ack action?

					if err := postAck(msg.Message); err != nil {
						t.l.Error().Err(err).Msgf("error in post-ack: %v", err)
						return
					}
				}(msg)
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

func (t *MessageQueueImpl) notify(channel string, message string) error {
	ch := "queue_" + strings.ReplaceAll(channel, "-", "_")

	msg := strings.ReplaceAll(message, "'", "\\'")
	cmd := fmt.Sprintf("NOTIFY %s, '%s'", ch, msg)
	log.Println("CMD")
	log.Println(cmd)
	if _, err := t.db.Exec(cmd); err != nil {
		return fmt.Errorf("error publishing msg using NOTIFY: %v", err)
	}
	return nil
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
