//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
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

	t.Run("cursor buckets: watermark carried forward, expired past retention", func(t *testing.T) {
		repo := createStreamsRepository(t, pool)
		tenantId := uuid.New()

		_, err := repo.InsertOrderedStreamMessage(ctx, tenantId, streamOpts(0, "m"))
		require.NoError(t, err)

		ageBuckets := func(days int) {
			_, err := pool.Exec(ctx, `SELECT create_v1_range_partition('v1_stream_producer_cursor', (NOW() AT TIME ZONE 'UTC')::date - $1::int, 80)`, days)
			require.NoError(t, err)
			_, err = pool.Exec(ctx, `UPDATE v1_stream_producer_cursor SET bucket = bucket - $1::int WHERE tenant_id = $2`, days, tenantId)
			require.NoError(t, err)
		}

		// the producer's only row moves to yesterday; its next write copies the
		// watermark into today's bucket
		ageBuckets(1)

		second, err := repo.InsertOrderedStreamMessage(ctx, tenantId, streamOpts(1, "m"))
		require.NoError(t, err)
		require.True(t, second.Inserted)

		var buckets int
		require.NoError(t, pool.QueryRow(ctx, `SELECT COUNT(*) FROM v1_stream_producer_cursor WHERE tenant_id = $1 AND last_seq = 1`, tenantId).Scan(&buckets))
		assert.Equal(t, 2, buckets, "the watermark must be carried into today's bucket")

		// dropping the older bucket (retention) leaves the watermark intact
		_, err = pool.Exec(ctx, `DELETE FROM v1_stream_producer_cursor WHERE tenant_id = $1 AND bucket < (NOW() AT TIME ZONE 'UTC')::date`, tenantId)
		require.NoError(t, err)

		dup, err := repo.InsertOrderedStreamMessage(ctx, tenantId, streamOpts(1, "m"))
		require.NoError(t, err)
		assert.False(t, dup.Inserted)
		assert.Equal(t, int64(1), dup.CurrentSeq)

		// a watermark older than cursor retention counts as none, so the next
		// publish is a gap the SDK answers with a new producer
		ageBuckets(int(streamProducerCursorRetention/(24*time.Hour)) + 1)

		next, err := repo.InsertOrderedStreamMessage(ctx, tenantId, streamOpts(2, "m"))
		require.NoError(t, err)
		assert.False(t, next.Inserted)
		assert.Equal(t, int64(-1), next.CurrentSeq)
	})

	t.Run("reads hide what's past the tenant's retention", func(t *testing.T) {
		repo := createStreamsRepository(t, pool)
		shortTenant := createLimitTestTenant(t, pool)
		defaultTenant := createLimitTestTenant(t, pool)
		setStreamRetentionHours(t, pool, shortTenant, 1)

		// the migration only seeds partitions from the current hour on
		_, err := pool.Exec(ctx, `SELECT create_v1_hourly_range_partition('v1_stream_message', NOW() - make_interval(hours => h)) FROM generate_series(1, 5) AS h`)
		require.NoError(t, err)

		_, err = pool.Exec(ctx, `
			INSERT INTO v1_stream_message (id, tenant_id, topic, payload, producer_id, producer_seq, inserted_at)
			VALUES (1, $1, 't', 'old', 'p', 0, NOW() - INTERVAL '3 hours'), (2, $1, 't', 'new', 'p', 1, NOW())
		`, shortTenant)
		require.NoError(t, err)

		msgs, err := repo.ListMessagesAfterCursor(ctx, shortTenant, ListStreamMessagesOpts{Topic: "t"})
		require.NoError(t, err)
		assert.Equal(t, []string{"new"}, streamPayloads(t, msgs))

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

		var stored []int64
		rows, err := pool.Query(ctx, `SELECT id FROM v1_stream_message WHERE tenant_id = $1 ORDER BY id`, tenantId)
		require.NoError(t, err)
		for rows.Next() {
			var id int64
			require.NoError(t, rows.Scan(&id))
			stored = append(stored, id)
		}
		require.NoError(t, rows.Err())

		require.Len(t, stored, producers*perProducer)
		assert.Equal(t, stored, seen, "the reader must see every stored message, in order")
	})

	// The expired-message delete runs in batches until a tenant has none left,
	// touching nothing newer and no other tenant.
	t.Run("expired messages are deleted in batches", func(t *testing.T) {
		repo := createTaskRepository(pool)
		tenant, other := uuid.New(), uuid.New()

		_, err := pool.Exec(ctx, `SELECT create_v1_hourly_range_partition('v1_stream_message', NOW() - INTERVAL '3 hours')`)
		require.NoError(t, err)

		expired := deleteExpiredStreamMessagesBatchSize*2 + 7

		_, err = pool.Exec(ctx, `
			INSERT INTO v1_stream_message (id, tenant_id, topic, payload, producer_id, producer_seq, inserted_at)
			SELECT g, t, 't', 'm', 'p', g, NOW() - INTERVAL '3 hours'
			FROM generate_series(1, $1) g, unnest($2::uuid[]) AS t`, expired, []uuid.UUID{tenant, other})
		require.NoError(t, err)

		_, err = pool.Exec(ctx, `INSERT INTO v1_stream_message (id, tenant_id, topic, payload, producer_id, producer_seq) VALUES ($1, $2, 't', 'm', 'p', 0)`, expired+1, tenant)
		require.NoError(t, err)

		require.NoError(t, repo.deleteExpiredStreamMessages(ctx, tenant, time.Now().Add(-time.Hour)))

		count := func(tenant uuid.UUID) int {
			var n int
			require.NoError(t, pool.QueryRow(ctx, `SELECT COUNT(*) FROM v1_stream_message WHERE tenant_id = $1`, tenant).Scan(&n))
			return n
		}

		assert.Equal(t, 1, count(tenant), "only the recent message is left")
		assert.Equal(t, expired, count(other), "other tenants are untouched")
	})

	t.Run("idle topics are deleted using each tenant's retention", func(t *testing.T) {
		repo := createTaskRepository(pool)
		defaultTenant := createLimitTestTenant(t, pool)
		longTenant := createLimitTestTenant(t, pool)
		setStreamRetentionHours(t, pool, longTenant, 2000)

		// 40 days idle: past the default 720h, within longTenant's 2000h. More
		// than one batch so the sweep has to loop.
		_, err := pool.Exec(ctx, `
			INSERT INTO v1_stream_topic (tenant_id, topic, last_published_at)
			SELECT $1, 'idle-' || g, NOW() - INTERVAL '40 days' FROM generate_series(1, $2) g
		`, defaultTenant, deleteIdleStreamTopicsBatchSize+5)
		require.NoError(t, err)

		_, err = pool.Exec(ctx, `
			INSERT INTO v1_stream_topic (tenant_id, topic, last_published_at)
			VALUES ($1, 'active', NOW()), ($2, 'idle-long', NOW() - INTERVAL '40 days')
		`, defaultTenant, longTenant)
		require.NoError(t, err)

		require.NoError(t, repo.deleteIdleStreamTopics(ctx))

		var remaining []string
		rows, err := pool.Query(ctx, `SELECT topic FROM v1_stream_topic WHERE tenant_id = ANY($1) ORDER BY topic`, []uuid.UUID{defaultTenant, longTenant})
		require.NoError(t, err)
		for rows.Next() {
			var topic string
			require.NoError(t, rows.Scan(&topic))
			remaining = append(remaining, topic)
		}
		require.NoError(t, rows.Err())

		assert.Equal(t, []string{"active", "idle-long"}, remaining)
	})

	t.Run("an uploaded payload is published and read by ref", func(t *testing.T) {
		repo := createStreamsRepository(t, pool)
		tenantId, other := uuid.New(), uuid.New()

		ref, err := repo.InsertStreamPayload(ctx, tenantId, []byte("large"))
		require.NoError(t, err)

		require.NoError(t, repo.CheckStreamPayloadExists(ctx, tenantId, ref))
		assert.ErrorIs(t, repo.CheckStreamPayloadExists(ctx, other, ref), ErrStreamPayloadNotFound, "another tenant's payload")

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
}
