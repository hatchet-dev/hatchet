package rabbitmq

import (
	"context"
	"net/url"
	"sync"
	"time"

	"github.com/hatchet-dev/hatchet/internal/queueutils"

	"github.com/jackc/puddle/v2"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

const (
	channelPoolQueueDurable = "durable"
	channelPoolQueuePubSub  = "pubsub"
	channelPoolRolePub      = "pub"
	channelPoolRoleSub      = "sub"

	channelPoolAcquiredMetric = "hatchet.msgqueue.rabbitmq.channel_pool.acquired"
	channelPoolMaxMetric      = "hatchet.msgqueue.rabbitmq.channel_pool.max"
)

// reconnectCheckInterval is how often a pool checks whether its connection is
// still open. It is a variable so tests can shorten it.
var reconnectCheckInterval = time.Second

type channelPool struct {
	*puddle.Pool[*amqp.Channel]

	l *zerolog.Logger

	url  string
	dial func(url string) (*amqp.Connection, error)

	conn   *amqp.Connection
	connMu sync.Mutex

	metricReg metric.Registration
}

// redactURL masks the password in a connection URL (as url.URL.Redacted does).
// If the URL cannot be parsed, a placeholder is returned instead.
func redactURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || (raw != "" && u.Host == "") {
		return "<unparseable url>"
	}

	return u.Redacted()
}

func (p *channelPool) newConnection() error {
	conn, err := p.dial(p.url)

	if err != nil {
		p.l.Error().Msgf("cannot (re)dial: %v: %q", err, redactURL(p.url))
		return err
	}

	p.connMu.Lock()
	p.conn = conn
	p.connMu.Unlock()
	return nil
}

func (p *channelPool) getConnection() *amqp.Connection {
	p.connMu.Lock()
	defer p.connMu.Unlock()

	return p.conn
}

// reconnect replaces a closed connection and drops every pooled channel that was
// opened on the old one. Those channels are closed with their connection, and
// puddle hands out idle resources before constructing new ones, so without the
// reset a publisher could acquire nothing but dead channels until each had
// been discovered and destroyed one at a time. Reset also marks channels that
// are acquired right now, so they are destroyed on release instead of returning
// to the idle set.
func (p *channelPool) reconnect() error {
	if err := p.newConnection(); err != nil {
		return err
	}

	p.Reset()

	return nil
}

func (p *channelPool) hasActiveConnection() bool {
	p.connMu.Lock()
	defer p.connMu.Unlock()

	return p.conn != nil && !p.conn.IsClosed()
}

func (p *channelPool) Close() {
	if p.metricReg != nil {
		_ = p.metricReg.Unregister()
		p.metricReg = nil
	}

	if p.Pool != nil {
		p.Pool.Close()
	}
}

func newChannelPool(ctx context.Context, l *zerolog.Logger, url string, maxChannels int32, queue, role string) (*channelPool, error) {
	return newChannelPoolWithDial(ctx, l, url, maxChannels, queue, role, amqp.Dial)
}

func newChannelPoolWithDial(ctx context.Context, l *zerolog.Logger, url string, maxChannels int32, queue, role string, dial func(string) (*amqp.Connection, error)) (*channelPool, error) {
	p := &channelPool{
		l:    l,
		url:  url,
		dial: dial,
	}

	err := p.newConnection()

	if err != nil {
		return nil, err
	}

	constructor := func(context.Context) (*amqp.Channel, error) {
		conn := p.getConnection()

		ch, err := conn.Channel()

		if err != nil {
			l.Error().Msgf("cannot create channel: %v", err)
			return nil, err
		}

		return ch, nil
	}

	destructor := func(ch *amqp.Channel) {
		if !ch.IsClosed() {
			err := ch.Close()

			if err != nil {
				l.Error().Msgf("error closing channel: %v", err)
			}
		}
	}

	// FIXME: this is probably too many channels
	maxPoolSize := maxChannels

	pool, err := puddle.NewPool(&puddle.Config[*amqp.Channel]{
		Constructor: constructor,
		Destructor:  destructor,
		MaxSize:     maxPoolSize,
	})

	if err != nil {
		l.Error().Err(err).Msg("cannot create connection pool")
		return nil, nil
	}

	p.Pool = pool
	p.registerMetrics(queue, role)

	// started last: reconnect resets the pool, so the pool must be assigned
	// before the watcher can observe a closed connection
	go p.watchConnection(ctx, role)

	return p, nil
}

// watchConnection periodically checks whether the connection is still open and
// redials it when it is not.
func (p *channelPool) watchConnection(ctx context.Context, role string) {
	retries := 0
	ticker := time.NewTicker(reconnectCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}

		conn := p.getConnection()

		if conn.IsClosed() {
			p.l.Warn().Str("pool", role).Msg("rabbitmq connection is closed, redialing")

			err := p.reconnect()

			if err != nil {
				p.l.Error().Msgf("cannot (re)dial: %v: %q", err, redactURL(p.url))
				queueutils.SleepWithExponentialBackoff(10*time.Millisecond, 5*time.Second, retries)
				retries++
				continue
			}

			retries = 0
		}
	}
}

func (p *channelPool) registerMetrics(queue, role string) {
	meter := otel.Meter("hatchet.run/metrics")

	acquired, err := meter.Int64ObservableGauge(
		channelPoolAcquiredMetric,
		metric.WithDescription("AMQP channels currently acquired from the RabbitMQ channel pool"),
		metric.WithUnit("{channel}"),
	)
	if err != nil {
		p.l.Warn().Err(err).Msg("cannot create channel pool acquired gauge")
		return
	}

	maxChans, err := meter.Int64ObservableGauge(
		channelPoolMaxMetric,
		metric.WithDescription("Configured maximum AMQP channels in the RabbitMQ channel pool"),
		metric.WithUnit("{channel}"),
	)
	if err != nil {
		p.l.Warn().Err(err).Msg("cannot create channel pool max gauge")
		return
	}

	attrs := metric.WithAttributes(
		attribute.String("queue", queue),
		attribute.String("pool", role),
	)

	reg, err := meter.RegisterCallback(func(_ context.Context, o metric.Observer) error {
		if p.Pool == nil {
			return nil
		}

		st := p.Stat()
		o.ObserveInt64(acquired, int64(st.AcquiredResources()), attrs)
		o.ObserveInt64(maxChans, int64(st.MaxResources()), attrs)

		return nil
	}, acquired, maxChans)
	if err != nil {
		p.l.Warn().Err(err).Msg("cannot register channel pool metric callback")
		return
	}

	p.metricReg = reg
}
