package rabbitmq

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/puddle/v2"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog"
)

type channelWithId struct {
	*amqp.Channel

	id string
}

type channelPool struct {
	*puddle.Pool[*channelWithId]

	// we keep a list of channel IDs to status. if channels are closed we store them as
	// false
	channels   map[string]bool
	channelsMu sync.Mutex
}

func newChannelPool(ctx context.Context, l *zerolog.Logger, connectionPool *connectionPool) (*channelPool, error) {
	p := &channelPool{
		channels: make(map[string]bool),
	}

	constructor := func(context.Context) (*channelWithId, error) {
		id := uuid.New().String()

		conn, err := connectionPool.Acquire(ctx)

		if err != nil {
			l.Error().Msgf("cannot acquire connection: %v", err)
			return nil, err
		}

		ch, err := conn.Value().Channel()

		if err != nil {
			conn.Destroy()
			l.Error().Msgf("cannot create channel: %v", err)
			return nil, err
		}

		conn.Release()

		closeCh := ch.NotifyClose(make(chan *amqp.Error, 1))

		go func() {
			select {
			case <-ctx.Done():
				return
			case <-closeCh:
				l.Info().Msgf("channel %s closed", id)
				p.setChannelClosed(id)
			}
		}()

		p.setChannelOpen(id)

		return &channelWithId{
			Channel: ch,
			id:      id,
		}, nil
	}

	destructor := func(ch *channelWithId) {
		p.removeChannel(ch.id)

		if !ch.IsClosed() {
			err := ch.Close()

			if err != nil {
				l.Error().Msgf("error closing channel: %v", err)
			}
		}
	}

	// FIXME: this is probably too many channels
	maxPoolSize := int32(100)

	pool, err := puddle.NewPool(&puddle.Config[*channelWithId]{
		Constructor: constructor,
		Destructor:  destructor,
		MaxSize:     maxPoolSize,
	})

	if err != nil {
		l.Error().Err(err).Msg("cannot create connection pool")
		return nil, nil
	}

	// start a goroutine which periodically removes closed connections from the pool
	go func() {
		ticker := time.NewTicker(5 * time.Second)

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				chs := pool.AcquireAllIdle()

				p.channelsMu.Lock()

				for _, ch := range chs {
					v := ch.Value()

					if val, ok := p.channels[v.id]; !ok || !val {
						ch.Destroy()
					}

					if v.IsClosed() {
						ch.Destroy()
					}

					ch.Release()
				}

				p.channelsMu.Unlock()
			}
		}
	}()

	p.Pool = pool

	return p, nil
}

func (p *channelPool) setChannelOpen(id string) {
	p.channelsMu.Lock()
	defer p.channelsMu.Unlock()

	p.channels[id] = true
}

func (p *channelPool) setChannelClosed(id string) {
	p.channelsMu.Lock()
	defer p.channelsMu.Unlock()

	p.channels[id] = false
}

func (p *channelPool) removeChannel(id string) {
	p.channelsMu.Lock()
	defer p.channelsMu.Unlock()

	delete(p.channels, id)
}
