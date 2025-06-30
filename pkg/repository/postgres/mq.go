package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hatchet-dev/hatchet/internal/telemetry"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

type messageQueueRepository struct {
	*sharedRepository

	m *multiplexedListener
}

func NewMessageQueueRepository(shared *sharedRepository) *messageQueueRepository {
	m := newMultiplexedListener(shared.l, shared.pool)

	return &messageQueueRepository{
		sharedRepository: shared,
		m:                m,
	}
}

func (mq *messageQueueRepository) Listen(ctx context.Context, name string, f func(ctx context.Context, notification *repository.PubSubMessage) error) error {
	return mq.m.listen(ctx, name, f)
}

func (mq *messageQueueRepository) Notify(ctx context.Context, name string, payload string) error {
	return mq.m.notify(ctx, name, payload)
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
