//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/validator"
)

func createStreamsRepository(t *testing.T, pool *pgxpool.Pool) *streamsRepositoryImpl {
	logger := zerolog.Nop()

	return &streamsRepositoryImpl{
		sharedRepository: &sharedRepository{
			pool:    pool,
			ddlPool: pool,
			v:       validator.NewDefaultValidator(),
			l:       &logger,
			queries: sqlcv1.New(),
			m:       createTenantLimitRepositoryForTest(t, pool, defaultLimitTestConfig()),
		},
	}
}

func streamOpts(seq int64, payload string) CreateOrderedStreamMessageOpts {
	return CreateOrderedStreamMessageOpts{Topic: "t", Payload: []byte(payload), ProducerID: "p1", ProducerSeq: seq}
}

func streamPayloads(t *testing.T, msgs []*sqlcv1.V1StreamMessage) []string {
	t.Helper()

	out := make([]string, len(msgs))
	for i, m := range msgs {
		out[i] = string(m.Payload)
	}

	return out
}

// Repository behavior that only touches its own tenants, so every case shares
// one database. Cases that depend on database-wide state (the partition job,
// query plans) have their own.
func TestStreamsRepository(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("sequence check: first, out of order, gap, duplicate", func(t *testing.T) {
		repo := createStreamsRepository(t, pool)
		tenantId := uuid.New()

		// seq 1 arriving before seq 0 is a gap, not the producer's first row,
		// or seq 0 would later be dropped as stale
		early, err := repo.InsertOrderedStreamMessage(ctx, tenantId, streamOpts(1, "second"))
		require.NoError(t, err)
		assert.False(t, early.Inserted)
		assert.Equal(t, int64(-1), early.CurrentSeq, "no watermark yet")

		first, err := repo.InsertOrderedStreamMessage(ctx, tenantId, streamOpts(0, "first"))
		require.NoError(t, err)
		require.True(t, first.Inserted)

		retried, err := repo.InsertOrderedStreamMessage(ctx, tenantId, streamOpts(1, "second"))
		require.NoError(t, err)
		require.True(t, retried.Inserted)

		gap, err := repo.InsertOrderedStreamMessage(ctx, tenantId, streamOpts(3, "fourth"))
		require.NoError(t, err)
		assert.False(t, gap.Inserted)
		assert.Equal(t, int64(1), gap.CurrentSeq, "a gap reports the real watermark")

		dup, err := repo.InsertOrderedStreamMessage(ctx, tenantId, streamOpts(0, "first"))
		require.NoError(t, err)
		assert.False(t, dup.Inserted)
		assert.Equal(t, int64(1), dup.CurrentSeq, "a duplicate reports the real watermark")

		msgs, err := repo.ListMessagesAfterCursor(ctx, tenantId, ListStreamMessagesOpts{Topic: "t"})
		require.NoError(t, err)
		assert.Equal(t, []string{"first", "second"}, streamPayloads(t, msgs))
	})

	t.Run("a batch applies in producer order with per-topic offsets", func(t *testing.T) {
		repo := createStreamsRepository(t, pool)
		tenantA, tenantB := uuid.New(), uuid.New()

		msg := func(tenant uuid.UUID, topic, producer string, seq int64) TenantStreamMessage {
			return TenantStreamMessage{TenantID: tenant, Opts: CreateOrderedStreamMessageOpts{
				Topic: topic, Payload: []byte(fmt.Sprintf("%s-%d", producer, seq)), ProducerID: producer, ProducerSeq: seq,
			}}
		}

		results, err := repo.InsertOrderedStreamMessages(ctx, []TenantStreamMessage{
			msg(tenantA, "a", "p1", 1), // arrives before its predecessor in the same batch
			msg(tenantB, "a", "p2", 0),
			msg(tenantA, "a", "p1", 0),
			msg(tenantA, "a", "p1", 3), // gap: no seq 2
			msg(tenantA, "a", "p1", 0), // duplicate
			msg(tenantA, "b", "p1", 0), // another topic: its own offsets
			msg(tenantA, "b", "p1", 1),
		})
		require.NoError(t, err)

		inserted := make([]bool, len(results))
		for i, r := range results {
			inserted[i] = r.Inserted
		}

		assert.Equal(t, []bool{true, true, true, false, false, true, true}, inserted)

		// one more publish to topic a continues its offsets
		_, err = repo.InsertOrderedStreamMessage(ctx, tenantA, CreateOrderedStreamMessageOpts{Topic: "a", Payload: []byte("p1-2"), ProducerID: "p1", ProducerSeq: 2})
		require.NoError(t, err)

		ids := func(topic string) ([]int64, []string) {
			msgs, err := repo.ListMessagesAfterCursor(ctx, tenantA, ListStreamMessagesOpts{Topic: topic})
			require.NoError(t, err)

			out := make([]int64, len(msgs))
			for i, m := range msgs {
				out[i] = m.ID
			}

			return out, streamPayloads(t, msgs)
		}

		aIDs, aPayloads := ids("a")
		assert.Equal(t, []string{"p1-0", "p1-1", "p1-2"}, aPayloads)
		assert.IsIncreasing(t, aIDs, "rejected messages may leave holes, never reorder")

		bIDs, _ := ids("b")
		assert.Equal(t, []int64{1, 2}, bIDs, "each topic numbers its own offsets")
	})

	t.Run("cursors expire past the tenant's retention or the oldest partition", func(t *testing.T) {
		repo := createStreamsRepository(t, pool)
		shortTenant := createLimitTestTenant(t, pool)
		defaultTenant := createLimitTestTenant(t, pool)
		setStreamRetentionHours(t, pool, shortTenant, 1)

		// the migration only seeds partitions from the current hour on
		require.NoError(t, repo.queries.CreateStreamMessagePartitions(ctx, pool, sqlcv1.CreateStreamMessagePartitionsParams{
			Fromtime: pgtype.Timestamptz{Time: time.Now().Add(-5 * time.Hour), Valid: true},
			Totime:   pgtype.Timestamptz{Time: time.Now(), Valid: true},
		}))

		hoursAgo := func(h int) StreamCursor {
			return StreamCursor{ID: 5, CreatedAt: time.Now().Add(-time.Duration(h) * time.Hour)}
		}

		var expired *StreamCursorExpiredError

		assert.NoError(t, repo.CheckCursorRetained(ctx, shortTenant, StreamCursor{}), "the zero cursor can't expire")
		assert.NoError(t, repo.CheckCursorRetained(ctx, shortTenant, hoursAgo(0)))

		require.ErrorAs(t, repo.CheckCursorRetained(ctx, shortTenant, hoursAgo(3)), &expired, "past the tenant's own retention")
		assert.WithinDuration(t, time.Now().Add(-time.Hour), expired.RetentionStart, time.Minute)

		assert.NoError(t, repo.CheckCursorRetained(ctx, defaultTenant, hoursAgo(3)), "within the default retention and the retained partitions")

		require.ErrorAs(t, repo.CheckCursorRetained(ctx, defaultTenant, hoursAgo(10)), &expired, "older than the oldest partition")
		assert.WithinDuration(t, time.Now().Add(-5*time.Hour), expired.RetentionStart, time.Hour)
	})

	t.Run("topic metadata reports a topic's newest message and count", func(t *testing.T) {
		repo := createStreamsRepository(t, pool)
		tenantId := uuid.New()

		for seq := range int64(3) {
			_, err := repo.InsertOrderedStreamMessage(ctx, tenantId, streamOpts(seq, "m"))
			require.NoError(t, err)
		}

		_, err := repo.InsertOrderedStreamMessage(ctx, tenantId, CreateOrderedStreamMessageOpts{Namespace: "ns", Topic: "t", Payload: []byte("m"), ProducerID: "p1"})
		require.NoError(t, err)

		msgs, err := repo.ListMessagesAfterCursor(ctx, tenantId, ListStreamMessagesOpts{Topic: "t"})
		require.NoError(t, err)
		last := msgs[len(msgs)-1]

		md, err := repo.GetTopicMetadata(ctx, tenantId, "", "t")
		require.NoError(t, err)
		assert.Equal(t, int64(3), md.MessageCount)
		require.NotNil(t, md.LatestCursor)
		assert.Equal(t, last.ID, md.LatestCursor.ID)
		assert.Equal(t, "t", md.LatestCursor.Topic)
		assert.True(t, last.InsertedAt.Time.Equal(md.LatestCursor.CreatedAt))

		other, err := repo.GetTopicMetadata(ctx, tenantId, "ns", "t")
		require.NoError(t, err)
		assert.Equal(t, int64(1), other.MessageCount, "the same topic name in another namespace is its own topic")

		_, err = repo.GetTopicMetadata(ctx, tenantId, "", "missing")
		assert.ErrorIs(t, err, ErrStreamTopicNotFound)
	})

	t.Run("pages stop at the byte budget", func(t *testing.T) {
		repo := createStreamsRepository(t, pool)
		tenantId := uuid.New()
		payload := make([]byte, 3*1024*1024)

		for seq := range int64(5) {
			res, err := repo.InsertOrderedStreamMessage(ctx, tenantId, CreateOrderedStreamMessageOpts{
				Topic: "t", Payload: payload, ProducerID: "p1", ProducerSeq: seq,
			})
			require.NoError(t, err)
			require.True(t, res.Inserted)
		}

		// 3MB each against an 8MB budget: the first three start under it
		first, err := repo.ListMessagesAfterCursor(ctx, tenantId, ListStreamMessagesOpts{Topic: "t"})
		require.NoError(t, err)
		require.Len(t, first, 3)

		last := first[len(first)-1]
		rest, err := repo.ListMessagesAfterCursor(ctx, tenantId, ListStreamMessagesOpts{Topic: "t", Cursor: StreamCursor{ID: last.ID, CreatedAt: last.InsertedAt.Time}})
		require.NoError(t, err)
		require.Len(t, rest, 2)
		assert.Greater(t, rest[0].ID, last.ID)
	})

	// A reader paging forward while many producers publish concurrently must end
	// up with every message exactly once, in order: offsets become visible in
	// order, so nothing may appear behind its cursor after it has moved past.
	t.Run("a concurrent reader never skips a message", func(t *testing.T) {
		repo := createStreamsRepository(t, pool)
		tenantId := uuid.New()

		const producers, perProducer = 8, 40

		var wg sync.WaitGroup
		for p := range producers {
			wg.Go(func() {
				for seq := range int64(perProducer) {
					res, err := repo.InsertOrderedStreamMessage(ctx, tenantId, CreateOrderedStreamMessageOpts{
						Topic: "t", Payload: []byte(fmt.Sprintf("p%d-%d", p, seq)), ProducerID: fmt.Sprintf("p%d", p), ProducerSeq: seq,
					})
					assert.NoError(t, err)
					assert.True(t, res.Inserted)
				}
			})
		}

		done := make(chan struct{})
		go func() {
			wg.Wait()
			close(done)
		}()

		var cursor StreamCursor
		var seen []int64

		read := func() {
			msgs, err := repo.ListMessagesAfterCursor(ctx, tenantId, ListStreamMessagesOpts{Topic: "t", Cursor: cursor})
			require.NoError(t, err)

			for _, m := range msgs {
				seen = append(seen, m.ID)
				cursor = StreamCursor{ID: m.ID, CreatedAt: m.InsertedAt.Time}
			}
		}

		for finished := false; !finished; {
			select {
			case <-done:
				finished = true
			default:
			}

			read()
		}

		read()

		// every publish succeeded on one topic, so the stored offsets are exactly 1..n
		stored := make([]int64, producers*perProducer)
		for i := range stored {
			stored[i] = int64(i + 1)
		}

		assert.Equal(t, stored, seen, "the reader must see every stored message, in order")
	})

	// The expired-message delete runs in batches until a tenant has none left,
	// touching nothing newer and no other tenant.
	t.Run("expired messages are deleted in batches", func(t *testing.T) {
		repo := createTaskRepository(pool)
		streams := createStreamsRepository(t, pool)
		tenant, other := uuid.New(), uuid.New()

		publish := func(tenantId uuid.UUID, producer string, n int) {
			msgs := make([]TenantStreamMessage, n)
			for i := range msgs {
				msgs[i] = TenantStreamMessage{TenantID: tenantId, Opts: CreateOrderedStreamMessageOpts{Topic: "t", Payload: []byte("m"), ProducerID: producer, ProducerSeq: int64(i)}}
			}

			_, err := streams.InsertOrderedStreamMessages(ctx, msgs)
			require.NoError(t, err)
		}

		count := func(tenantId uuid.UUID) []*sqlcv1.V1StreamMessage {
			msgs, err := streams.ListMessagesAfterCursor(ctx, tenantId, ListStreamMessagesOpts{Topic: "t", Limit: 1000})
			require.NoError(t, err)
			return msgs
		}

		expired := deleteExpiredStreamMessagesBatchSize*2 + 7
		publish(tenant, "old", expired)
		publish(other, "old", expired)

		// inserted_at is the publish transaction's start, shared by the whole batch
		cutoff := count(tenant)[0].InsertedAt.Time.Add(time.Microsecond)
		time.Sleep(10 * time.Millisecond)
		publish(tenant, "new", 1)

		require.NoError(t, repo.deleteExpiredStreamMessages(ctx, tenant, cutoff))

		assert.Len(t, count(tenant), 1, "only the newer message is left")
		assert.Len(t, count(other), expired, "other tenants are untouched")
	})

	t.Run("expired messages take their uploaded payloads with them", func(t *testing.T) {
		repo := createTaskRepository(pool)
		streams := createStreamsRepository(t, pool)
		tenant, other := uuid.New(), uuid.New()

		upload := func(tenantId uuid.UUID) StreamPayloadRef {
			ref, err := streams.InsertStreamPayload(ctx, tenantId, []byte("large"))
			require.NoError(t, err)
			return ref
		}

		publish := func(tenantId uuid.UUID, seq int64, ref StreamPayloadRef) {
			res, err := streams.InsertOrderedStreamMessage(ctx, tenantId, CreateOrderedStreamMessageOpts{Topic: "t", ProducerID: "p", ProducerSeq: seq, PayloadRef: &ref})
			require.NoError(t, err)
			require.True(t, res.Inserted)
		}

		payloadExists := func(tenantId uuid.UUID, ref StreamPayloadRef) bool {
			_, err := streams.GetStreamPayload(ctx, tenantId, ref)
			if errors.Is(err, ErrStreamPayloadNotFound) {
				return false
			}
			require.NoError(t, err)
			return true
		}

		expiredRef, orphanRef, otherRef := upload(tenant), upload(tenant), upload(other)
		publish(tenant, 0, expiredRef)
		publish(other, 0, otherRef)

		msgs, err := streams.ListMessagesAfterCursor(ctx, tenant, ListStreamMessagesOpts{Topic: "t"})
		require.NoError(t, err)
		cutoff := msgs[0].InsertedAt.Time.Add(time.Microsecond)

		time.Sleep(10 * time.Millisecond)
		keptRef := upload(tenant)
		publish(tenant, 1, keptRef)

		require.NoError(t, repo.deleteExpiredStreamMessages(ctx, tenant, cutoff))

		assert.False(t, payloadExists(tenant, expiredRef), "deleted with its expired message")
		assert.True(t, payloadExists(tenant, keptRef), "its message is still retained")
		assert.True(t, payloadExists(tenant, orphanRef), "never published, so it's left for its partition to be dropped")
		assert.True(t, payloadExists(other, otherRef), "another tenant's retention isn't applied")
	})

	t.Run("an uploaded payload is published and read by ref", func(t *testing.T) {
		repo := createStreamsRepository(t, pool)
		tenantId, other := uuid.New(), uuid.New()

		ref, err := repo.InsertStreamPayload(ctx, tenantId, []byte("large"))
		require.NoError(t, err)

		retained, err := repo.RetainStreamPayloadForPublish(ctx, tenantId, ref)
		require.NoError(t, err)
		assert.Equal(t, ref, retained, "a fresh upload is referenced as is")

		_, err = repo.RetainStreamPayloadForPublish(ctx, other, ref)
		assert.ErrorIs(t, err, ErrStreamPayloadNotFound, "another tenant's payload")

		res, err := repo.InsertOrderedStreamMessage(ctx, tenantId, CreateOrderedStreamMessageOpts{Topic: "t", ProducerID: "p1", ProducerSeq: 0, PayloadRef: &ref})
		require.NoError(t, err)
		require.True(t, res.Inserted)

		msgs, err := repo.ListMessagesAfterCursor(ctx, tenantId, ListStreamMessagesOpts{Topic: "t"})
		require.NoError(t, err)
		require.Len(t, msgs, 1)
		assert.Empty(t, msgs[0].Payload)
		require.NotNil(t, msgs[0].PayloadID)
		assert.Equal(t, ref.ID, *msgs[0].PayloadID)
		assert.True(t, ref.CreatedAt.Equal(msgs[0].PayloadInsertedAt.Time))

		encoded, err := EncodeStreamPayloadRef(ref)
		require.NoError(t, err)
		decoded, err := DecodeStreamPayloadRef(encoded)
		require.NoError(t, err)

		payload, err := repo.GetStreamPayload(ctx, tenantId, decoded)
		require.NoError(t, err)
		assert.Equal(t, "large", string(payload))

		_, err = repo.GetStreamPayload(ctx, other, decoded)
		assert.ErrorIs(t, err, ErrStreamPayloadNotFound, "another tenant's payload")

		_, err = repo.InsertOrderedStreamMessage(ctx, tenantId, CreateOrderedStreamMessageOpts{Topic: "t", ProducerID: "p1", ProducerSeq: 1})
		assert.Error(t, err, "a message needs a payload or a ref")
	})

	t.Run("an upload too old to outlive its message is copied on publish", func(t *testing.T) {
		repo := createStreamsRepository(t, pool)
		tenantId := uuid.New()

		ref, err := repo.InsertStreamPayload(ctx, tenantId, []byte("large"))
		require.NoError(t, err)

		// a cutoff in the future treats the fresh upload as too old
		tooOld := time.Now().Add(time.Hour)

		copied, err := repo.retainStreamPayloadForPublish(ctx, tenantId, ref, tooOld)
		require.NoError(t, err)
		assert.NotEqual(t, ref.ID, copied.ID)
		assert.False(t, copied.CreatedAt.Before(ref.CreatedAt), "the copy lands in the current partition")

		payload, err := repo.GetStreamPayload(ctx, tenantId, copied)
		require.NoError(t, err)
		assert.Equal(t, "large", string(payload))

		_, err = repo.retainStreamPayloadForPublish(ctx, uuid.New(), ref, tooOld)
		assert.ErrorIs(t, err, ErrStreamPayloadNotFound, "another tenant's payload isn't copied")
	})
}
