package repository

import (
	"context"
	"encoding/json"
	"slices"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgxlisten"
	"github.com/rs/zerolog"
)

// multiplexChannel is a single channel used for all multiplexed messages.
const multiplexChannel = "hatchet_listener"

// multiplexedListener listens for messages on a single Postgres channel and
// dispatches them to the appropriate handlers based on the queue name.
type multiplexedListener struct {
	isListening   bool
	isListeningMu sync.Mutex

	pool *pgxpool.Pool

	l *zerolog.Logger

	subscribers   map[string][]chan *PubSubMessage
	subscribersMu sync.RWMutex

	listenerCtx context.Context
	cancel      context.CancelFunc
}

func newMultiplexedListener(l *zerolog.Logger, pool *pgxpool.Pool) *multiplexedListener {
	listenerCtx, cancel := context.WithCancel(context.Background())

	return &multiplexedListener{
		pool:        pool,
		subscribers: make(map[string][]chan *PubSubMessage),
		cancel:      cancel,
		listenerCtx: listenerCtx,
		l:           l,
	}
}

func (m *multiplexedListener) startListening() {
	m.isListeningMu.Lock()
	defer m.isListeningMu.Unlock()

	if m.isListening {
		return
	}

	// listen for multiplexed messages
	listener := &pgxlisten.Listener{
		Connect: func(ctx context.Context) (*pgx.Conn, error) {
			// Acquire a new connection each time
			poolConn, err := m.pool.Acquire(ctx)
			if err != nil {
				return nil, err
			}
			return poolConn.Conn(), nil
		},
		LogError: func(innerCtx context.Context, err error) {
			m.l.Warn().Err(err).Msg("error in listener")
		},
		ReconnectDelay: 10 * time.Second,
	}

	var handler pgxlisten.HandlerFunc = func(ctx context.Context, notification *pgconn.Notification, conn *pgx.Conn) error {
		// unmarshal the payload
		if notification.Payload == "" {
			return nil
		}

		pubSubMsg := &PubSubMessage{}

		err := json.Unmarshal([]byte(notification.Payload), pubSubMsg)

		if err != nil {
			m.l.Error().Err(err).Msg("error unmarshalling notification payload")
			return err
		}

		m.publishToSubscribers(pubSubMsg)

		return nil
	}

	listener.Handle(multiplexChannel, handler)

	go func() {
		err := listener.Listen(m.listenerCtx)

		if err != nil {
			m.isListeningMu.Lock()
			m.isListening = false
			m.isListeningMu.Unlock()

			m.l.Error().Err(err).Msg("error listening for multiplexed messages")
			return
		}
	}()

	m.isListening = true
}

func (m *multiplexedListener) publishToSubscribers(msg *PubSubMessage) {
	m.subscribersMu.RLock()
	defer m.subscribersMu.RUnlock()

	if subscribers, exists := m.subscribers[msg.QueueName]; exists {
		for _, ch := range subscribers {
			select {
			case ch <- msg:
			default:
				// Channel is full or closed, skip this subscriber
				m.l.Warn().Str("queue", msg.QueueName).Msg("failed to send message to subscriber channel")
			}
		}
	}
}

func (m *multiplexedListener) subscribe(queueName string) chan *PubSubMessage {
	m.subscribersMu.Lock()
	defer m.subscribersMu.Unlock()

	ch := make(chan *PubSubMessage, 100) // Buffered channel
	m.subscribers[queueName] = append(m.subscribers[queueName], ch)
	return ch
}

func (m *multiplexedListener) unsubscribe(queueName string, ch chan *PubSubMessage) {
	m.subscribersMu.Lock()
	defer m.subscribersMu.Unlock()

	if subscribers, exists := m.subscribers[queueName]; exists {
		for i, subscriber := range subscribers {
			if subscriber == ch {
				close(ch)
				m.subscribers[queueName] = slices.Delete(subscribers, i, i+1)
				if len(m.subscribers[queueName]) == 0 {
					delete(m.subscribers, queueName)
				}
				break
			}
		}
	}
}

// NOTE: name is the target channel, not the global multiplex channel
func (m *multiplexedListener) listen(ctx context.Context, name string, f func(ctx context.Context, notification *PubSubMessage) error) error {
	m.startListening()

	// Subscribe to the channel for the specific queue
	ch := m.subscribe(name)
	defer m.unsubscribe(name, ch)

	for {
		select {
		case msg, ok := <-ch:
			if !ok {
				// Channel was closed
				return nil
			}
			// Spawn handler as goroutine to avoid blocking message processing
			go func(msg *PubSubMessage) {
				err := f(ctx, msg)
				if err != nil {
					m.l.Error().Err(err).Msg("error processing notification")
				}
			}(msg)
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// wrapMessage wraps a payload in a PubSubMessage and marshals it.
// Returns the marshaled bytes or an error.
func (m *multiplexedListener) wrapMessage(name string, payload string) ([]byte, error) {
	var jsonPayload json.RawMessage

	// Handle empty payload - use null as valid JSON
	if payload == "" {
		jsonPayload = json.RawMessage("null")
	} else {
		jsonPayload = json.RawMessage(payload)
	}

	pubSubMsg := &PubSubMessage{
		QueueName: name,
		Payload:   jsonPayload,
	}

	return json.Marshal(pubSubMsg)
}

// notify sends a notification through the Postgres channel.
// wrappedPayload should be the already-marshaled PubSubMessage.
func (m *multiplexedListener) notify(ctx context.Context, wrappedPayload []byte) error {
	_, err := m.pool.Exec(ctx, "select pg_notify($1,$2)", multiplexChannel, string(wrappedPayload))
	return err
}
