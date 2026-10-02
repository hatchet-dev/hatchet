package repository

import (
	"bytes"
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// defaultListStreamMessagesLimit bounds a single ListMessagesAfterCursor page
// when the caller doesn't specify one.
const defaultListStreamMessagesLimit = 500

// MaxStreamMessagePayloadBytes caps a single durable stream message. Enforced
// by the Publish gRPC handler before a message ever reaches the queue; the
// CreateOrderedStreamMessageOpts.Payload validate tag below is a second line
// of defense and must be kept numerically in sync with this constant.
//
// It sits just under gRPC's 4 MiB message limit, leaving room for the rest of
// a publish request and of a delivered entry (worst case, its cursor); see
// TestMaxPayloadFitsInGRPCMessages.
const MaxStreamMessagePayloadBytes = 4*1024*1024 - 5*1024

// streamProducerCursorRetention is how long a producer's watermark survives
// without a publish. Past it, the producer's next publish is rejected as a
// sequence gap and the SDK resumes under a new producer ID.
const streamProducerCursorRetention = 3 * 24 * time.Hour

// streamProducerCursorMinBucket is the oldest producer cursor bucket still
// within retention as of now: lookups read from it, and older partitions are
// dropped.
func streamProducerCursorMinBucket(now time.Time) pgtype.Date {
	return pgtype.Date{Time: now.UTC().Add(-streamProducerCursorRetention).Truncate(24 * time.Hour), Valid: true}
}

// MaxStreamNameLength caps a stream topic or namespace, in characters. The
// validate tags below must stay in sync with it.
const MaxStreamNameLength = 255

// ValidateStreamAddress rejects a namespace or topic that couldn't be stored,
// so a publish fails up front rather than after it has been accepted.
func ValidateStreamAddress(namespace, topic string) error {
	if topic == "" {
		return errors.New("topic is required")
	}

	if err := validateStreamName("topic", topic); err != nil {
		return err
	}

	return validateStreamName("namespace", namespace)
}

func validateStreamName(field, name string) error {
	if !utf8.ValidString(name) {
		return fmt.Errorf("%s must be valid UTF-8", field)
	}

	if n := utf8.RuneCountInString(name); n > MaxStreamNameLength {
		return fmt.Errorf("%s is %d characters long, exceeding the maximum of %d", field, n, MaxStreamNameLength)
	}

	// Postgres text columns can't store NUL
	if strings.ContainsRune(name, 0) {
		return fmt.Errorf("%s must not contain NUL characters", field)
	}

	return nil
}

// MaxListStreamMessagesBytes bounds the payload bytes one ListMessagesAfterCursor
// page loads, so catching up on large messages can't load the whole row limit's
// worth into memory at once.
const MaxListStreamMessagesBytes = 8 * 1024 * 1024

// CreateOrderedStreamMessageOpts carries a published message plus the
// producer_id/producer_seq every publish is required to supply, needed to
// enforce that single producer's emission order (see
// InsertOrderedStreamMessages).
type CreateOrderedStreamMessageOpts struct {
	Namespace string `validate:"max=255"`

	Topic string `validate:"required,max=255"`

	Payload []byte `validate:"required,max=4189184"`

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
}

type ListStreamMessagesOpts struct {
	// (optional) the namespace the topic belongs to; empty string is the default namespace
	Namespace string

	Topic string `validate:"required,max=255"`

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

	// InsertOrderedStreamMessages applies InsertOrderedStreamMessage to each
	// message in one transaction and one round trip, returning results in the
	// order given. A sequence gap is a result, not an error; an error means
	// nothing was applied.
	InsertOrderedStreamMessages(ctx context.Context, msgs []TenantStreamMessage) ([]OrderedStreamMessageResult, error)

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
			if delErr := r.queries.DeleteStreamTopic(ctx, r.pool, sqlcv1.DeleteStreamTopicParams{Tenantid: tenantId, Namespace: namespace, Topic: topic}); delErr != nil {
				r.l.Error().Ctx(ctx).Err(delErr).Msg("failed to roll back stream topic after topic limit exceeded")
			}

			return ErrResourceExhausted
		}
	}

	r.streamTopicSeenCache.Add(key, struct{}{})

	return nil
}

func (r *streamsRepositoryImpl) InsertOrderedStreamMessage(ctx context.Context, tenantId uuid.UUID, opts CreateOrderedStreamMessageOpts) (OrderedStreamMessageResult, error) {
	results, err := r.InsertOrderedStreamMessages(ctx, []TenantStreamMessage{{TenantID: tenantId, Opts: opts}})

	if err != nil {
		return OrderedStreamMessageResult{}, err
	}

	return results[0], nil
}

// TenantStreamMessage is one message of an InsertOrderedStreamMessages batch.
type TenantStreamMessage struct {
	TenantID uuid.UUID
	Opts     CreateOrderedStreamMessageOpts
}

func (r *streamsRepositoryImpl) InsertOrderedStreamMessages(ctx context.Context, msgs []TenantStreamMessage) ([]OrderedStreamMessageResult, error) {
	for i := range msgs {
		if err := r.v.Validate(&msgs[i].Opts); err != nil {
			return nil, err
		}
	}

	// apply in a fixed (producer, seq) order: concurrent batches then always lock
	// cursor rows in the same order and can't deadlock, and a producer's
	// messages within a batch apply in sequence
	order := make([]int, len(msgs))
	for i := range order {
		order[i] = i
	}

	slices.SortStableFunc(order, func(a, b int) int {
		ma, mb := msgs[a], msgs[b]

		return cmp.Or(
			bytes.Compare(ma.TenantID[:], mb.TenantID[:]),
			cmp.Compare(ma.Opts.Namespace, mb.Opts.Namespace),
			cmp.Compare(ma.Opts.Topic, mb.Opts.Topic),
			cmp.Compare(ma.Opts.ProducerID, mb.Opts.ProducerID),
			cmp.Compare(ma.Opts.ProducerSeq, mb.Opts.ProducerSeq),
		)
	})

	params := make([]sqlcv1.InsertOrderedStreamMessageParams, len(order))
	minBucket := streamProducerCursorMinBucket(time.Now())

	for i, idx := range order {
		m := msgs[idx]

		params[i] = sqlcv1.InsertOrderedStreamMessageParams{
			Tenantid:        m.TenantID,
			Namespace:       m.Opts.Namespace,
			Topic:           m.Opts.Topic,
			Payload:         m.Opts.Payload,
			Producerid:      m.Opts.ProducerID,
			Producerseq:     m.Opts.ProducerSeq,
			Expectedprevseq: m.Opts.ProducerSeq - 1,
			Minbucket:       minBucket,
		}

	}

	// one offset reservation per topic, in the same sorted order, so concurrent
	// batches also take topic row locks (always before cursor rows) in one order
	var reservations []sqlcv1.ReserveStreamTopicOffsetsParams
	var groupStart []int

	for i, p := range params {
		if i == 0 || p.Tenantid != params[i-1].Tenantid || p.Namespace != params[i-1].Namespace || p.Topic != params[i-1].Topic {
			reservations = append(reservations, sqlcv1.ReserveStreamTopicOffsetsParams{Tenantid: p.Tenantid, Namespace: p.Namespace, Topic: p.Topic})
			groupStart = append(groupStart, i)
		}

		reservations[len(reservations)-1].Count++
	}

	results := make([]OrderedStreamMessageResult, len(msgs))

	err := func() error {
		tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, r.pool, r.l)

		if err != nil {
			return err
		}

		defer rollback()

		var batchErr error

		r.queries.ReserveStreamTopicOffsets(ctx, tx, reservations).QueryRow(func(g int, lastOffset int64, err error) {
			if err != nil {
				batchErr = cmp.Or(batchErr, err)
				return
			}

			first := lastOffset - reservations[g].Count + 1

			for j := range reservations[g].Count {
				params[groupStart[g]+int(j)].Messageoffset = first + j
			}
		})

		if batchErr != nil {
			return batchErr
		}

		r.queries.InsertOrderedStreamMessage(ctx, tx, params).QueryRow(func(i int, row *sqlcv1.InsertOrderedStreamMessageRow, err error) {
			if err != nil {
				batchErr = cmp.Or(batchErr, err)
				return
			}

			currentSeq := int64(-1)

			if row.CurrentLastSeq.Valid {
				currentSeq = row.CurrentLastSeq.Int64
			}

			results[order[i]] = OrderedStreamMessageResult{Inserted: row.Inserted, CurrentSeq: currentSeq}
		})

		if batchErr != nil {
			return batchErr
		}

		return commit(ctx)
	}()

	if err != nil {
		return nil, err
	}

	return results, nil
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
		Limit:    limit,
		Maxbytes: MaxListStreamMessagesBytes,
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
