package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgxlisten"

	"github.com/hatchet-dev/hatchet/internal/telemetry"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

type messageQueueRepository struct {
	*sharedRepository
}

func NewMessageQueueRepository(shared *sharedRepository) *messageQueueRepository {
	return &messageQueueRepository{
		sharedRepository: shared,
	}
}

func (m *messageQueueRepository) Listen(ctx context.Context, name string, f func(ctx context.Context, notification *repository.PubMessage) error) error {
	l := &pgxlisten.Listener{
		Connect: func(ctx context.Context) (*pgx.Conn, error) {
			pgxpoolConn, err := m.pool.Acquire(ctx)

			if err != nil {
				return nil, err
			}

			return pgxpoolConn.Conn(), nil
		},
		LogError: func(innerCtx context.Context, err error) {
			if ctx.Err() != nil {
				m.l.Warn().Err(err).Msg("error in listener")
			}
		},
		ReconnectDelay: 10 * time.Second,
	}

	var handler pgxlisten.HandlerFunc = func(ctx context.Context, notification *pgconn.Notification, conn *pgx.Conn) error {
		return f(ctx, &repository.PubMessage{
			Channel: notification.Channel,
			Payload: notification.Payload,
		})
	}

	l.Handle(name, handler)

	return l.Listen(ctx)
}

func (m *messageQueueRepository) Notify(ctx context.Context, name string, payload string) error {
	_, err := m.pool.Exec(ctx, "select pg_notify($1,$2)", name, payload)

	return err
}

func (m *messageQueueRepository) AddMessage(ctx context.Context, queue string, payload []byte) error {
	// NOTE: hack for tenant, just passing in an empty string for now
	_, err := m.bulkAddMQBuffer.FireAndWait(ctx, "", addMessage{
		queue:   queue,
		payload: payload,
	})

	return err
}

func (m *messageQueueRepository) BindQueue(ctx context.Context, queue string, durable, autoDeleted, exclusive bool, exclusiveConsumer *string) error {
	// if exclusive, but no consumer, return error
	if exclusive && exclusiveConsumer == nil {
		return errors.New("exclusive queue must have exclusive consumer")
	}

	params := dbsqlc.UpsertMessageQueueParams{
		Name:        queue,
		Durable:     durable,
		Autodeleted: autoDeleted,
		Exclusive:   exclusive,
	}

	if exclusiveConsumer != nil {
		params.ExclusiveConsumerId = sqlchelpers.UUIDFromStr(*exclusiveConsumer)
	}

	_, err := m.queries.UpsertMessageQueue(ctx, m.pool, params)

	return err
}

func (m *messageQueueRepository) UpdateQueueLastActive(ctx context.Context, queue string) error {
	return m.queries.UpdateMessageQueueActive(ctx, m.pool, queue)
}

func (m *messageQueueRepository) CleanupQueues(ctx context.Context) error {
	return m.queries.CleanupMessageQueue(ctx, m.pool)
}

func (m *messageQueueRepository) ReadMessages(ctx context.Context, queue string, qos int) ([]*dbsqlc.ReadMessagesRow, error) {
	ctx, span := telemetry.NewSpan(ctx, "pgmq-read-messages")
	defer span.End()

	return m.queries.ReadMessages(ctx, m.pool, dbsqlc.ReadMessagesParams{
		Queueid: queue,
		Limit:   pgtype.Int4{Int32: int32(qos), Valid: true}, // nolint: gosec
	})
}

func (m *messageQueueRepository) AckMessage(ctx context.Context, id int64) error {
	// NOTE: hack for tenant, just passing in an empty string for now
	return m.bulkAckMQBuffer.FireForget("", id)
}

func (m *messageQueueRepository) CleanupMessageQueueItems(ctx context.Context) error {
	// setup telemetry
	ctx, span := telemetry.NewSpan(ctx, "cleanup-message-queues-database")
	defer span.End()

	// get the min and max queue items
	minMax, err := m.queries.GetMinMaxExpiredMessageQueueItems(ctx, m.pool)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}

		return fmt.Errorf("could not get min max processed queue items: %w", err)
	}

	if minMax == nil {
		return nil
	}

	minId := minMax.MinId
	maxId := minMax.MaxId

	if maxId == 0 {
		return nil
	}

	// iterate until we have no more queue items to process
	var batchSize int64 = 10000
	var currBatch int64

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		currBatch++

		currMax := minId + batchSize*currBatch

		if currMax > maxId {
			currMax = maxId
		}

		// get the next batch of queue items
		err := m.queries.CleanupMessageQueueItems(ctx, m.pool, dbsqlc.CleanupMessageQueueItemsParams{
			Minid: minId,
			Maxid: minId + batchSize*currBatch,
		})

		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil
			}

			return fmt.Errorf("could not cleanup queue items: %w", err)
		}

		if currMax == maxId {
			break
		}
	}

	return nil
}
