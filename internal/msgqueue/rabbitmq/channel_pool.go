package rabbitmq

import (
	"context"
	"fmt"
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

type channelPool struct {
	*puddle.Pool[*amqp.Channel]

	l *zerolog.Logger

	url string

	conn   *amqp.Connection
	connMu sync.Mutex

	metricReg metric.Registration

	// confirmMode, when true, puts every channel this pool constructs into
	// RabbitMQ publisher-confirm mode and tracks a waiter per channel (see
	// pubConfirm) -- confirmations are always drained in the background
	// regardless of whether any caller is waiting, since an undrained
	// NotifyPublish channel can stall the whole connection's frame dispatch.
	confirmMode bool
	confirms    sync.Map // *amqp.Channel -> *pubConfirm
}

// pubConfirm tracks the in-flight publisher-confirm waiter for one pooled
// channel. A channel is only ever checked out to one goroutine at a time, so
// there is at most one waiter -- registering it before publishing (see
// publishWithConfirm) is enough to correlate it correctly, no delivery-tag
// bookkeeping needed.
type pubConfirm struct {
	mu     sync.Mutex
	waiter chan amqp.Confirmation
}

// publishWithConfirm runs publish with a confirm waiter registered first, then
// blocks for the broker's ack/nack or ctx cancellation. If ch isn't in confirm
// mode it just runs publish, so callers that don't need the guarantee pay
// nothing extra.
func (p *channelPool) publishWithConfirm(ctx context.Context, ch *amqp.Channel, publish func() error) error {
	v, ok := p.confirms.Load(ch)

	if !ok {
		return publish()
	}

	pc := v.(*pubConfirm)

	waitCh := make(chan amqp.Confirmation, 1)

	pc.mu.Lock()
	pc.waiter = waitCh
	pc.mu.Unlock()

	clearWaiter := func() {
		pc.mu.Lock()
		if pc.waiter == waitCh {
			pc.waiter = nil
		}
		pc.mu.Unlock()
	}

	if err := publish(); err != nil {
		clearWaiter()
		return err
	}

	select {
	case confirmation := <-waitCh:
		if !confirmation.Ack {
			return fmt.Errorf("broker nacked publish")
		}

		return nil
	case <-ctx.Done():
		clearWaiter()
		return ctx.Err()
	}
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
	conn, err := amqp.Dial(p.url)

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

func newChannelPool(ctx context.Context, l *zerolog.Logger, url string, maxChannels int32, queue, role string, confirmMode bool) (*channelPool, error) {
	p := &channelPool{
		l:           l,
		url:         url,
		confirmMode: confirmMode,
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

		if p.confirmMode {
			if err := ch.Confirm(false); err != nil {
				l.Error().Msgf("cannot enable publisher confirms: %v", err)
				_ = ch.Close()
				return nil, err
			}

			pc := &pubConfirm{}
			confirmations := ch.NotifyPublish(make(chan amqp.Confirmation, 1))
			p.confirms.Store(ch, pc)

			// always drained, whether or not a publish is currently waiting on
			// it -- an unread NotifyPublish channel can stall dispatch for the
			// whole connection, not just this channel. Exits when ch closes.
			go func() {
				for confirmation := range confirmations {
					pc.mu.Lock()
					w := pc.waiter
					pc.waiter = nil
					pc.mu.Unlock()

					if w != nil {
						w <- confirmation
					}
				}
			}()
		}

		return ch, nil
	}

	destructor := func(ch *amqp.Channel) {
		if p.confirmMode {
			p.confirms.Delete(ch)
		}

		if !ch.IsClosed() {
			err := ch.Close()

			if err != nil {
				l.Error().Msgf("error closing channel: %v", err)
			}
		}
	}

	// periodically check if the connection is still open
	go func() {
		retries := 0
		ticker := time.NewTicker(1 * time.Second)

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}

			conn := p.getConnection()

			if conn.IsClosed() {
				err := p.newConnection()

				if err != nil {
					l.Error().Msgf("cannot (re)dial: %v: %q", err, redactURL(p.url))
					queueutils.SleepWithExponentialBackoff(10*time.Millisecond, 5*time.Second, retries)
					retries++
					continue
				}

				retries = 0
			}
		}
	}()

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

	return p, nil
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
