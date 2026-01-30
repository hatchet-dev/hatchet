package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/telemetry"
)

type PubSubMessage struct {
	QueueName string `json:"queue_name"`
	Payload   []byte `json:"payload"`
}

type MessageQueueRepository interface {
	// PubSub
	Listen(ctx context.Context, name string, f func(ctx context.Context, notification *PubSubMessage) error) error
	Notify(ctx context.Context, name string, payload string) error

	// Queues
	BindQueue(ctx context.Context, queue string, durable, autoDeleted, exclusive bool, exclusiveConsumer *string) error
	UpdateQueueLastActive(ctx context.Context, queue string) error
	CleanupQueues(ctx context.Context) error

	// Messages
	AddMessage(ctx context.Context, queue string, payload []byte) error
	ReadMessages(ctx context.Context, queue string, qos int) ([]*sqlcv1.ReadMessagesRow, error)
	AckMessage(ctx context.Context, id int64) error
	CleanupMessageQueueItems(ctx context.Context) error
}

type messageQueueRepository struct {
	*sharedRepository

	m *multiplexedListener
}

func newMessageQueueRepository(shared *sharedRepository) (*messageQueueRepository, func() error) {
	m := newMultiplexedListener(shared.l, shared.pool)

	return &messageQueueRepository{
			sharedRepository: shared,
			m:                m,
		}, func() error {
			m.cancel()
			return nil
		}
}

func (m *messageQueueRepository) Listen(ctx context.Context, name string, f func(ctx context.Context, notification *PubSubMessage) error) error {
	return m.m.listen(ctx, name, f)
}

func (m *messageQueueRepository) Notify(ctx context.Context, name string, payload string) error {
	return m.m.notify(ctx, name, payload)
}

func (m *messageQueueRepository) AddMessage(ctx context.Context, queue string, payload []byte) error {
	p := []sqlcv1.BulkAddMessageParams{}

	p = append(p, sqlcv1.BulkAddMessageParams{
		QueueId: pgtype.Text{
			String: queue,
			Valid:  true,
		},
		Payload:   payload,
		ExpiresAt: sqlchelpers.TimestampFromTime(time.Now().UTC().Add(5 * time.Minute)),
		ReadAfter: sqlchelpers.TimestampFromTime(time.Now().UTC()),
	})

	_, err := m.queries.BulkAddMessage(ctx, m.pool, p)

	return err
}

func (m *messageQueueRepository) BindQueue(ctx context.Context, queue string, durable, autoDeleted, exclusive bool, exclusiveConsumer *string) error {
	// if exclusive, but no consumer, return error
	if exclusive && exclusiveConsumer == nil {
		return errors.New("exclusive queue must have exclusive consumer")
	}

	params := sqlcv1.UpsertMessageQueueParams{
		Name:        queue,
		Durable:     durable,
		Autodeleted: autoDeleted,
		Exclusive:   exclusive,
	}

	if exclusiveConsumer != nil {
		parsedUuid := uuid.MustParse(*exclusiveConsumer)
		params.ExclusiveConsumerId = &parsedUuid
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

func (m *messageQueueRepository) ReadMessages(ctx context.Context, queue string, qos int) ([]*sqlcv1.ReadMessagesRow, error) {
	ctx, span := telemetry.NewSpan(ctx, "pgmq-read-messages")
	defer span.End()

	return m.queries.ReadMessages(ctx, m.pool, sqlcv1.ReadMessagesParams{
		Queueid: queue,
		Limit:   pgtype.Int4{Int32: int32(qos), Valid: true}, // nolint: gosec
	})
}

func (m *messageQueueRepository) AckMessage(ctx context.Context, id int64) error {
	return m.queries.BulkAckMessages(ctx, m.pool, []int64{id})
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
		err := m.queries.CleanupMessageQueueItems(ctx, m.pool, sqlcv1.CleanupMessageQueueItemsParams{
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
