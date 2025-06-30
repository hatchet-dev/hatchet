package postgres

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgxlisten"
	"github.com/rs/zerolog"

	"github.com/kelindar/event"

	"github.com/hatchet-dev/hatchet/pkg/repository"
)

// multiplexChannel is a single channel used for all multiplexed messages.
const multiplexChannel = "hatchet_listener"

// multiplexedListener listens for messages on a single Postgres channel and
// dispatches them to the appropriate handlers based on the queue name.
type multiplexedListener struct {
	isListening   bool
	isListeningMu sync.Mutex

	pool *pgxpool.Pool

	bus *event.Dispatcher

	l *zerolog.Logger

	eventTypes   map[string]uint32
	eventTypesMu sync.Mutex

	listenerCtx context.Context
	cancel      context.CancelFunc
}

type busEvent struct {
	*repository.PubSubMessage

	eventType uint32
}

func (b busEvent) Type() uint32 {
	return b.eventType
}

func newMultiplexedListener(l *zerolog.Logger, pool *pgxpool.Pool) *multiplexedListener {
	bus := event.NewDispatcher()

	listenerCtx, cancel := context.WithCancel(context.Background())

	return &multiplexedListener{
		pool:        pool,
		bus:         bus,
		eventTypes:  make(map[string]uint32),
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

	// acquire an exclusive connection
	pgxpoolConn, _ := m.pool.Acquire(m.listenerCtx)

	// listen for multiplexed messages
	listener := &pgxlisten.Listener{
		Connect: func(ctx context.Context) (*pgx.Conn, error) {
			return pgxpoolConn.Conn(), nil
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

		pubSubMsg := &repository.PubSubMessage{}

		err := json.Unmarshal([]byte(notification.Payload), pubSubMsg)

		if err != nil {
			m.l.Error().Err(err).Msg("error unmarshalling notification payload")
			return err
		}

		event.Publish(m.bus, busEvent{
			PubSubMessage: pubSubMsg,
			eventType:     m.upsertEventType(pubSubMsg.QueueName),
		})

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

func (m *multiplexedListener) upsertEventType(name string) uint32 {
	m.eventTypesMu.Lock()
	defer m.eventTypesMu.Unlock()

	if _, exists := m.eventTypes[name]; !exists {
		eType := uint32(len(m.eventTypes) + 1) // nolint: gosec
		m.eventTypes[name] = eType
		return eType
	}

	return m.eventTypes[name]
}

// NOTE: name is the target channel, not the global multiplex channel
func (m *multiplexedListener) listen(ctx context.Context, name string, f func(ctx context.Context, notification *repository.PubSubMessage) error) error {
	m.startListening()

	// Create a subscription to the bus for the specific event type
	cancel := event.SubscribeTo(m.bus, m.upsertEventType(name), func(ev busEvent) {
		err := f(ctx, ev.PubSubMessage)

		if err != nil {
			m.l.Error().Err(err).Msg("error processing notification")
		}
	})

	<-ctx.Done()
	cancel()

	return nil
}

// notify sends a notification through the Postgres channel.
func (m *multiplexedListener) notify(ctx context.Context, name string, payload string) error {
	pubSubMsg := &repository.PubSubMessage{
		QueueName: name,
		Payload:   []byte(payload),
	}

	payloadBytes, err := json.Marshal(pubSubMsg)

	if err != nil {
		m.l.Error().Err(err).Msg("error marshalling notification payload")
		return err
	}

	_, err = m.pool.Exec(ctx, "select pg_notify($1,$2)", multiplexChannel, string(payloadBytes))

	return err
}
