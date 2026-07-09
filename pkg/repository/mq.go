package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/telemetry"
)

type PubSubMessage struct {
	QueueName string          `json:"queue_name"`
	Payload   json.RawMessage `json:"payload"`
}

type MessageQueueRepository interface {
	// PubSub
	Listen(ctx context.Context, name string, f func(ctx context.Context, notification *PubSubMessage) error) error
	Notify(ctx context.Context, name string, payload string, durable, autoDeleted, exclusive bool) error

	// Queues
	BindQueue(ctx context.Context, queue string, durable, autoDeleted, exclusive bool, exclusiveConsumer *string) error
	UpdateQueueLastActive(ctx context.Context, queue string) error
	CleanupQueues(ctx context.Context) error

	// Messages
	AddMessage(ctx context.Context, queue string, payload []byte) error
	AddMessageEnsuringQueue(ctx context.Context, queue string, payload []byte, durable, autoDeleted, exclusive bool) error
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

func (m *messageQueueRepository) Notify(ctx context.Context, name string, payload string, durable, autoDeleted, exclusive bool) error {
	wrappedPayload, err := m.m.wrapMessage(name, payload)
	if err != nil {
		m.l.Error().Ctx(ctx).Err(err).Msg("error wrapping message")
		return err
	}

	// PostgreSQL's pg_notify has an 8000 byte limit
	// If the wrapped message exceeds this, fall back to database storage
	if len(wrappedPayload) > 8000 {
		// An auto-deleted queue can be reaped by CleanupMessageQueue in the window
		// between the producer's 15s existence-cache hit and this insert, so route
		// through the self-healing path. This covers EXCLUSIVE auto-deleted queues
		// too — the dispatcher queue (expirable ⇒ autoDeleted, exclusive) and
		// controller consumer queues — which the reaper deletes regardless of
		// exclusivity.
		if autoDeleted {
			return m.AddMessageEnsuringQueue(ctx, name, []byte(payload), durable, autoDeleted, exclusive)
		}

		return m.AddMessage(ctx, name, []byte(payload))
	}

	return m.m.notify(ctx, wrappedPayload)
}

func (m *messageQueueRepository) AddMessage(ctx context.Context, queue string, payload []byte) error {
	return m.queries.AddMessage(ctx, m.pool, sqlcv1.AddMessageParams{
		Queueid: queue,
		Payload: payload,
	})
}

func (m *messageQueueRepository) AddMessageEnsuringQueue(ctx context.Context, queue string, payload []byte, durable, autoDeleted, exclusive bool) error {
	// The plain insert's FK check takes only a KEY SHARE lock on the parent
	// MessageQueue row, so concurrent publishers to the same queue do not
	// serialize on it (an unconditional ON CONFLICT DO UPDATE here takes a
	// row-exclusive tuple lock per publish, convoying every publisher on one
	// row). It fails with a foreign-key violation exactly when the parent has
	// been reaped, which is the only case that needs the self-heal.
	err := m.AddMessage(ctx, queue, payload)

	if err == nil || !autoDeleted || !isForeignKeyViolation(err) {
		return err
	}

	// The parent queue was reaped by CleanupMessageQueue (auto-deleted queues
	// are reap-eligible after 1h idle). Healing is expected at most once per
	// queue per reap: the producer's queue-bind cache expires every 15s and
	// each re-bind refreshes lastActive, so an actively-published queue can
	// never accrue the 1h of staleness the reaper requires (15s ≪ 1h). A
	// sustained heal rate therefore means something else is deleting
	// MessageQueue rows out-of-band.
	m.l.Warn().Ctx(ctx).Str("queue", queue).Msg("parent queue row missing on publish; self-healing by recreating it")

	// Recreate the queue and insert the item in a single atomic statement.
	// The upsert sets lastActive = NOW(), so the healed queue is not
	// reap-eligible for another hour and subsequent publishes take the plain
	// insert above.
	return m.queries.AddMessageEnsuringQueue(ctx, m.pool, sqlcv1.AddMessageEnsuringQueueParams{
		Queueid:     queue,
		Payload:     payload,
		Durable:     durable,
		Autodeleted: autoDeleted,
		Exclusive:   exclusive,
	})
}

func isForeignKeyViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == pgerrcode.ForeignKeyViolation
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
