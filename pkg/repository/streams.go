package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// defaultListStreamMessagesLimit bounds a single ListMessagesAfterCursor page
// when the caller doesn't specify one.
const defaultListStreamMessagesLimit = 500

// MaxStreamMessagePayloadBytes caps a single durable stream message. Enforced
// by the Publish gRPC handler before a message ever reaches the queue; the
// CreateOrderedStreamMessageOpts.Payload validate tag below is a second line
// of defense and must be kept numerically in sync with this constant.
const MaxStreamMessagePayloadBytes = 512 * 1024

// CreateOrderedStreamMessageOpts carries a published message plus the
// producer_id/producer_seq every publish is required to supply, needed to
// enforce that single producer's emission order (see
// InsertOrderedStreamMessage / internal/services/controllers/streams).
type CreateOrderedStreamMessageOpts struct {
	Namespace string `validate:"max=255"`

	Topic string `validate:"required,max=255"`

	Payload []byte `validate:"required,max=524288"`

	ProducerID string `validate:"required"`

	ProducerSeq int64 `validate:"min=0"`
}

// OrderedStreamMessageResult reports the outcome of InsertOrderedStreamMessage.
type OrderedStreamMessageResult struct {
	// Inserted is true iff the message was durably applied.
	Inserted bool

	// CurrentSeq is the producer's watermark as of this call, valid whenever
	// Inserted is false: CurrentSeq >= the attempted seq means it was a stale
	// duplicate (already applied, safe to drop); CurrentSeq < seq-1 means a
	// gap (the message arrived ahead of its predecessor, worth retrying).
	// -1 means this producer has no prior row at all (never published to
	// this topic before), which is itself always < seq-1 for any seq >= 0.
	CurrentSeq int64

	// CurrentSeqAdvancedAt is when CurrentSeq was last applied; zero when the
	// producer has no row.
	CurrentSeqAdvancedAt time.Time
}

type ListStreamMessagesOpts struct {
	// (optional) the namespace the topic belongs to; empty string is the default namespace
	Namespace string

	Topic string `validate:"required"`

	// (optional) resume from this cursor; the zero value starts from the
	// beginning of retained history
	Cursor StreamCursor

	// (optional) defaults to defaultListStreamMessagesLimit
	Limit int32 `validate:"omitempty,min=1,max=1000"`
}

type StreamsRepository interface {
	// EnsureTopic implicitly registers a (tenant, namespace, topic) the first
	// time it's published to, enforcing the tenant's topic-count limit on
	// genuinely new topics. It is a cheap no-op for a topic already seen
	// recently by this process (see sharedRepository.streamTopicSeenCache).
	EnsureTopic(ctx context.Context, tenantId uuid.UUID, namespace, topic string) error

	// InsertOrderedStreamMessage atomically applies a producer-sequenced
	// message only if ProducerSeq is exactly one past that producer's last
	// applied sequence, so a message that commits ahead of its predecessor is
	// reported as not-inserted instead of breaking emission order.
	InsertOrderedStreamMessage(ctx context.Context, tenantId uuid.UUID, opts CreateOrderedStreamMessageOpts) (OrderedStreamMessageResult, error)

	// ForceInsertOrderedStreamMessage inserts unconditionally and bumps the
	// producer's watermark forward, for a message InsertOrderedStreamMessage
	// has been unable to apply for too long.
	ForceInsertOrderedStreamMessage(ctx context.Context, tenantId uuid.UUID, opts CreateOrderedStreamMessageOpts) error

	// ListMessagesAfterCursor returns up to opts.Limit messages strictly after
	// opts.Cursor, ordered by id ascending.
	ListMessagesAfterCursor(ctx context.Context, tenantId uuid.UUID, opts ListStreamMessagesOpts) ([]*sqlcv1.V1StreamMessage, error)

	// CheckCursorRetained returns a *StreamCursorExpiredError if cursor points
	// past the tenant's retention or into a partition that has been dropped.
	CheckCursorRetained(ctx context.Context, tenantId uuid.UUID, cursor StreamCursor) error
}

// StreamCursorExpiredError reports a cursor whose position has been deleted by
// retention, so resuming from it would silently skip messages.
type StreamCursorExpiredError struct {
	CursorCreatedAt time.Time
	RetentionStart  time.Time
}

func (e *StreamCursorExpiredError) Error() string {
	return fmt.Sprintf(
		"stream cursor has expired: it points to a message from %s, but messages before %s have been deleted by retention",
		e.CursorCreatedAt.UTC().Format(time.RFC3339), e.RetentionStart.UTC().Format(time.RFC3339),
	)
}

type streamsRepositoryImpl struct {
	*sharedRepository
}

func newStreamsRepository(s *sharedRepository) StreamsRepository {
	return &streamsRepositoryImpl{
		sharedRepository: s,
	}
}

func (r *streamsRepositoryImpl) EnsureTopic(ctx context.Context, tenantId uuid.UUID, namespace, topic string) error {
	key := streamTopicKey{tenantId: tenantId, namespace: namespace, topic: topic}

	if _, ok := r.streamTopicSeenCache.Get(key); ok {
		return nil
	}

	row, err := r.queries.UpsertStreamTopic(ctx, r.pool, sqlcv1.UpsertStreamTopicParams{
		Tenantid:  tenantId,
		Namespace: namespace,
		Topic:     topic,
	})

	if err != nil {
		return err
	}

	if row.Inserted {
		// 0, not 1: the topic was just inserted, so the live count already includes it
		canCreate, _, err := r.m.CanCreate(ctx, sqlcv1.LimitResourceSTREAMTOPIC, tenantId, 0)

		if err != nil {
			return err
		}

		if !canCreate {
			// roll back the just-created row so the tenant's topic count stays accurate
			if delErr := r.queries.DeleteStreamTopic(ctx, r.pool, row.V1StreamTopic.ID); delErr != nil {
				r.l.Error().Ctx(ctx).Err(delErr).Msg("failed to roll back stream topic after topic limit exceeded")
			}

			return ErrResourceExhausted
		}
	}

	r.streamTopicSeenCache.Add(key, struct{}{})

	return nil
}

func (r *streamsRepositoryImpl) InsertOrderedStreamMessage(ctx context.Context, tenantId uuid.UUID, opts CreateOrderedStreamMessageOpts) (OrderedStreamMessageResult, error) {
	if err := r.v.Validate(&opts); err != nil {
		return OrderedStreamMessageResult{}, err
	}

	row, err := r.queries.InsertOrderedStreamMessage(ctx, r.pool, sqlcv1.InsertOrderedStreamMessageParams{
		Tenantid:        tenantId,
		Namespace:       opts.Namespace,
		Topic:           opts.Topic,
		Payload:         opts.Payload,
		Producerid:      opts.ProducerID,
		Producerseq:     opts.ProducerSeq,
		Expectedprevseq: opts.ProducerSeq - 1,
	})

	if err != nil {
		return OrderedStreamMessageResult{}, err
	}

	currentSeq := int64(-1)

	if row.CurrentLastSeq.Valid {
		currentSeq = row.CurrentLastSeq.Int64
	}

	return OrderedStreamMessageResult{Inserted: row.Inserted, CurrentSeq: currentSeq, CurrentSeqAdvancedAt: row.CurrentUpdatedAt.Time}, nil
}

func (r *streamsRepositoryImpl) ForceInsertOrderedStreamMessage(ctx context.Context, tenantId uuid.UUID, opts CreateOrderedStreamMessageOpts) error {
	if err := r.v.Validate(&opts); err != nil {
		return err
	}

	return r.queries.ForceInsertOrderedStreamMessage(ctx, r.pool, sqlcv1.ForceInsertOrderedStreamMessageParams{
		Tenantid:    tenantId,
		Namespace:   opts.Namespace,
		Topic:       opts.Topic,
		Payload:     opts.Payload,
		Producerid:  opts.ProducerID,
		Producerseq: opts.ProducerSeq,
	})
}

func (r *streamsRepositoryImpl) ListMessagesAfterCursor(ctx context.Context, tenantId uuid.UUID, opts ListStreamMessagesOpts) ([]*sqlcv1.V1StreamMessage, error) {
	if err := r.v.Validate(&opts); err != nil {
		return nil, err
	}

	limit := opts.Limit
	if limit == 0 {
		limit = defaultListStreamMessagesLimit
	}

	retention, err := r.m.StreamRetention(ctx, tenantId)

	if err != nil {
		return nil, err
	}

	return r.queries.ListStreamMessagesAfterCursor(ctx, r.pool, sqlcv1.ListStreamMessagesAfterCursorParams{
		Tenantid:  tenantId,
		Namespace: opts.Namespace,
		Topic:     opts.Topic,
		Afterid:   opts.Cursor.ID,
		Retainedsince: pgtype.Timestamptz{
			Time:  time.Now().Add(-retention),
			Valid: true,
		},
		Limit: limit,
	})
}

func (r *streamsRepositoryImpl) CheckCursorRetained(ctx context.Context, tenantId uuid.UUID, cursor StreamCursor) error {
	// the zero cursor means "from the oldest retained message", which can't expire
	if cursor.ID == 0 {
		return nil
	}

	retention, err := r.m.StreamRetention(ctx, tenantId)

	if err != nil {
		return err
	}

	retainedSince := time.Now().Add(-retention)

	partitionStart, err := r.queries.GetStreamMessageRetentionStart(ctx, r.pool)

	if err != nil {
		return err
	}

	if partitionStart.Valid && partitionStart.Time.After(retainedSince) {
		retainedSince = partitionStart.Time
	}

	if cursor.CreatedAt.Before(retainedSince) {
		return &StreamCursorExpiredError{CursorCreatedAt: cursor.CreatedAt, RetentionStart: retainedSince}
	}

	return nil
}
