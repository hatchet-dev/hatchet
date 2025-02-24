package rabbitmq

import (
	"context"
	"sync"
	"time"

	"github.com/jackc/puddle/v2"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog"
)

type channelPool struct {
	*puddle.Pool[*amqp.Channel]

	l *zerolog.Logger

	url string

	conn   *amqp.Connection
	connMu sync.Mutex
}

func (p *channelPool) newConnection() error {
	conn, err := amqp.Dial(p.url)

	if err != nil {
		p.l.Error().Msgf("cannot (re)dial: %v: %q", err, p.url)
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

func newChannelPool(ctx context.Context, l *zerolog.Logger, url string) (*channelPool, error) {
	p := &channelPool{
		l:   l,
		url: url,
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

	// periodically check if the connection is still open
	go func() {
		retries := 0

		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			conn := p.getConnection()

			if conn.IsClosed() {
				err := p.newConnection()

				if err != nil {
					l.Error().Msgf("cannot (re)dial: %v: %q", err, p.url)
					sleepWithExponentialBackoff(10*time.Millisecond, 5*time.Second, retries)
					retries++
					continue
				}

				retries = 0
			}
		}
	}()

	// FIXME: this is probably too many channels
	maxPoolSize := int32(100)

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

	return p, nil
}
