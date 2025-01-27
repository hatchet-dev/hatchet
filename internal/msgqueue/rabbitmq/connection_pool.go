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

type connectionWithId struct {
	*amqp.Connection

	id string
}

type connectionPool struct {
	*puddle.Pool[*connectionWithId]

	// we keep a list of connection IDs to status. if connections are closed we store them as
	// false
	connections   map[string]bool
	connectionsMu sync.Mutex
}

func newConnectionPool(ctx context.Context, l *zerolog.Logger, url string) (*connectionPool, error) {
	p := &connectionPool{
		connections: make(map[string]bool),
	}

	constructor := func(context.Context) (*connectionWithId, error) {
		id := uuid.New().String()

		conn, err := amqp.Dial(url)

		if err != nil {
			l.Error().Msgf("cannot (re)dial: %v: %q", err, url)
			return nil, err
		}

		closeCh := conn.NotifyClose(make(chan *amqp.Error, 1))

		go func() {
			select {
			case <-ctx.Done():
				return
			case <-closeCh:
				l.Info().Msgf("connection %s closed", id)
				p.setConnectionClosed(id)
			}
		}()

		p.setConnectionOpen(id)

		return &connectionWithId{
			Connection: conn,
			id:         id,
		}, nil
	}

	destructor := func(conn *connectionWithId) {
		p.removeConnection(conn.id)

		if !conn.IsClosed() {
			err := conn.Close()

			if err != nil {
				l.Error().Msgf("error closing connection: %v", err)
			}
		}
	}

	maxPoolSize := int32(10)

	pool, err := puddle.NewPool(&puddle.Config[*connectionWithId]{
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
				conns := pool.AcquireAllIdle()

				p.connectionsMu.Lock()

				for _, conn := range conns {
					v := conn.Value()

					if val, ok := p.connections[v.id]; !ok || !val {
						conn.Destroy()
					}

					if v.IsClosed() {
						conn.Destroy()
					}

					conn.Release()
				}

				p.connectionsMu.Unlock()
			}
		}
	}()

	p.Pool = pool

	return p, nil
}

func (p *connectionPool) hasActiveConnection() bool {
	return p.Pool.Stat().TotalResources() > 0
}

func (p *connectionPool) setConnectionOpen(id string) {
	p.connectionsMu.Lock()
	defer p.connectionsMu.Unlock()

	p.connections[id] = true
}

func (p *connectionPool) setConnectionClosed(id string) {
	p.connectionsMu.Lock()
	defer p.connectionsMu.Unlock()

	p.connections[id] = false
}

func (p *connectionPool) removeConnection(id string) {
	p.connectionsMu.Lock()
	defer p.connectionsMu.Unlock()

	delete(p.connections, id)
}
