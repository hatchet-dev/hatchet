//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
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

// TestInsertOrderedStreamMessage_FirstMessageScansCleanly guards against a
// real bug found against a live Postgres: current_last_seq was originally
// read via a scalar subquery against v1_stream_producer_cursor, which -- per
// Postgres's WITH-clause visibility rules -- cannot see the row the same
// statement's own CTE just inserted, and sqlc did not detect that this makes
// the column nullable. A producer's very first message therefore always
// returned a real inserted=true row, but Scan errored trying to put SQL NULL
// into a plain int64, which the fake-repository controller unit tests never
// caught since they never touched real SQL. No fake can replace this test.
func TestInsertOrderedStreamMessage_FirstMessageScansCleanly(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	repo := createStreamsRepository(t, pool)
	tenantId := uuid.New()

	res, err := repo.InsertOrderedStreamMessage(context.Background(), tenantId, CreateOrderedStreamMessageOpts{
		Topic:       "t",
		Payload:     []byte("hello"),
		ProducerID:  "p1",
		ProducerSeq: 0,
	})

	require.NoError(t, err)
	assert.True(t, res.Inserted)

	msgs, err := repo.ListMessagesAfterCursor(context.Background(), tenantId, ListStreamMessagesOpts{Topic: "t"})
	require.NoError(t, err)
	require.Len(t, msgs, 1)
	assert.Equal(t, []byte("hello"), msgs[0].Payload)
}

func TestInsertOrderedStreamMessage_GapAndDuplicateReportRealWatermark(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	repo := createStreamsRepository(t, pool)
	tenantId := uuid.New()

	opts := func(seq int64) CreateOrderedStreamMessageOpts {
		return CreateOrderedStreamMessageOpts{
			Topic: "t", Payload: []byte("m"), ProducerID: "p1", ProducerSeq: seq,
		}
	}

	first, err := repo.InsertOrderedStreamMessage(context.Background(), tenantId, opts(0))
	require.NoError(t, err)
	require.True(t, first.Inserted)

	// a gap: seq=2 arrives before its predecessor seq=1
	gap, err := repo.InsertOrderedStreamMessage(context.Background(), tenantId, opts(2))
	require.NoError(t, err)
	assert.False(t, gap.Inserted)
	assert.Equal(t, int64(0), gap.CurrentSeq, "watermark should reflect the real committed row, not NULL")

	// a stale redelivery of the already-applied seq=0
	dup, err := repo.InsertOrderedStreamMessage(context.Background(), tenantId, opts(0))
	require.NoError(t, err)
	assert.False(t, dup.Inserted)
	assert.Equal(t, int64(0), dup.CurrentSeq)
}

// A producer's seq=1 arriving before its seq=0 must be held as a gap, not
// inserted as the first row -- otherwise seq=0 is later dropped as stale.
func TestInsertOrderedStreamMessage_FirstMessageArrivingOutOfOrderIsAGap(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	repo := createStreamsRepository(t, pool)
	tenantId := uuid.New()

	opts := func(seq int64, payload string) CreateOrderedStreamMessageOpts {
		return CreateOrderedStreamMessageOpts{
			Topic: "t", Payload: []byte(payload), ProducerID: "p1", ProducerSeq: seq,
		}
	}

	early, err := repo.InsertOrderedStreamMessage(context.Background(), tenantId, opts(1, "second"))
	require.NoError(t, err)
	assert.False(t, early.Inserted)
	assert.Equal(t, int64(-1), early.CurrentSeq)

	first, err := repo.InsertOrderedStreamMessage(context.Background(), tenantId, opts(0, "first"))
	require.NoError(t, err)
	require.True(t, first.Inserted)

	retried, err := repo.InsertOrderedStreamMessage(context.Background(), tenantId, opts(1, "second"))
	require.NoError(t, err)
	require.True(t, retried.Inserted)

	msgs, err := repo.ListMessagesAfterCursor(context.Background(), tenantId, ListStreamMessagesOpts{Topic: "t"})
	require.NoError(t, err)
	require.Len(t, msgs, 2)
	assert.Equal(t, []byte("first"), msgs[0].Payload)
	assert.Equal(t, []byte("second"), msgs[1].Payload)
}

func TestForceInsertOrderedStreamMessage_InsertsAndBumpsWatermark(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	repo := createStreamsRepository(t, pool)
	tenantId := uuid.New()

	err := repo.ForceInsertOrderedStreamMessage(context.Background(), tenantId, CreateOrderedStreamMessageOpts{
		Topic: "t", Payload: []byte("forced"), ProducerID: "p1", ProducerSeq: 5,
	})
	require.NoError(t, err)

	msgs, err := repo.ListMessagesAfterCursor(context.Background(), tenantId, ListStreamMessagesOpts{Topic: "t"})
	require.NoError(t, err)
	require.Len(t, msgs, 1)
	assert.Equal(t, []byte("forced"), msgs[0].Payload)

	// a later in-order message (seq=6) must see the forced watermark, not a gap
	res, err := repo.InsertOrderedStreamMessage(context.Background(), tenantId, CreateOrderedStreamMessageOpts{
		Topic: "t", Payload: []byte("next"), ProducerID: "p1", ProducerSeq: 6,
	})
	require.NoError(t, err)
	assert.True(t, res.Inserted)
}

// An active producer's watermark is copied into today's bucket on its first
// write of the day, so dropping older buckets (retention) must not reset it.
func TestInsertOrderedStreamMessage_WatermarkSurvivesDroppingOldBuckets(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()
	repo := createStreamsRepository(t, pool)
	tenantId := uuid.New()

	opts := func(seq int64) CreateOrderedStreamMessageOpts {
		return CreateOrderedStreamMessageOpts{
			Topic: "t", Payload: []byte("m"), ProducerID: "p1", ProducerSeq: seq,
		}
	}

	first, err := repo.InsertOrderedStreamMessage(ctx, tenantId, opts(0))
	require.NoError(t, err)
	require.True(t, first.Inserted)

	// age the producer's only row into yesterday's bucket
	_, err = pool.Exec(ctx, `SELECT create_v1_range_partition('v1_stream_producer_cursor', (NOW() AT TIME ZONE 'UTC')::date - 1)`)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `UPDATE v1_stream_producer_cursor SET bucket = bucket - 1`)
	require.NoError(t, err)

	second, err := repo.InsertOrderedStreamMessage(ctx, tenantId, opts(1))
	require.NoError(t, err)
	require.True(t, second.Inserted)

	var buckets int
	require.NoError(t, pool.QueryRow(ctx, `SELECT COUNT(*) FROM v1_stream_producer_cursor WHERE last_seq = 1`).Scan(&buckets))
	assert.Equal(t, 2, buckets, "the watermark must be carried into today's bucket")

	// simulate retention dropping yesterday's partition
	_, err = pool.Exec(ctx, `DELETE FROM v1_stream_producer_cursor WHERE bucket < (NOW() AT TIME ZONE 'UTC')::date`)
	require.NoError(t, err)

	dup, err := repo.InsertOrderedStreamMessage(ctx, tenantId, opts(1))
	require.NoError(t, err)
	assert.False(t, dup.Inserted)
	assert.Equal(t, int64(1), dup.CurrentSeq)

	third, err := repo.InsertOrderedStreamMessage(ctx, tenantId, opts(2))
	require.NoError(t, err)
	assert.True(t, third.Inserted)
}

func TestCheckCursorRetained(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()
	repo := createStreamsRepository(t, pool)
	shortTenant := createLimitTestTenant(t, pool)
	defaultTenant := createLimitTestTenant(t, pool)
	setStreamRetentionHours(t, pool, shortTenant, 1)

	// the migration only seeds partitions from the current hour on
	_, err := pool.Exec(ctx, `SELECT create_v1_hourly_range_partition('v1_stream_message', NOW() - make_interval(hours => h)) FROM generate_series(1, 5) AS h`)
	require.NoError(t, err)

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
}

func TestListMessagesAfterCursor_HidesMessagesPastTenantRetention(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()
	repo := createStreamsRepository(t, pool)
	tenantId := createLimitTestTenant(t, pool)
	setStreamRetentionHours(t, pool, tenantId, 1)

	_, err := pool.Exec(ctx, `SELECT create_v1_hourly_range_partition('v1_stream_message', NOW() - INTERVAL '3 hours')`)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `
		INSERT INTO v1_stream_message (tenant_id, topic, payload, producer_id, producer_seq, inserted_at)
		VALUES ($1, 't', 'old', 'p', 0, NOW() - INTERVAL '3 hours'), ($1, 't', 'new', 'p', 1, NOW())
	`, tenantId)
	require.NoError(t, err)

	msgs, err := repo.ListMessagesAfterCursor(ctx, tenantId, ListStreamMessagesOpts{Topic: "t"})
	require.NoError(t, err)
	require.Len(t, msgs, 1)
	assert.Equal(t, []byte("new"), msgs[0].Payload)
}

func TestStreamPartitionsFollowLongestTenantRetention(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()
	repo := createTaskRepository(pool)
	config := defaultLimitTestConfig()
	config.DefaultStreamRetentionHours = 24
	repo.m = newTestTenantLimitRepository(pool, config)

	// shared partitions are kept for the longest retention of any tenant
	setStreamRetentionHours(t, pool, createLimitTestTenant(t, pool), 48)

	now := time.Now().UTC()
	today := now.Truncate(24 * time.Hour)
	queries := sqlcv1.New()

	messageHours := map[string]bool{ // partition hour -> kept
		now.Add(-5 * time.Hour).Format("2006010215"):  true,
		now.Add(-30 * time.Hour).Format("2006010215"): true,
		now.Add(-60 * time.Hour).Format("2006010215"): false,
	}
	cursorDays := map[string]bool{ // partition day -> kept (4 x 48h = 8 days)
		today.AddDate(0, 0, -2).Format("20060102"):  true,
		today.AddDate(0, 0, -10).Format("20060102"): false,
	}

	for _, h := range []int{5, 30, 60} {
		_, err := pool.Exec(ctx, `SELECT create_v1_hourly_range_partition('v1_stream_message', NOW() - make_interval(hours => $1::int))`, h)
		require.NoError(t, err)
	}

	for _, d := range []int{-2, -10} {
		_, err := queries.CreatePartitions(ctx, pool, pgtype.Date{Time: today.AddDate(0, 0, d), Valid: true})
		require.NoError(t, err)
	}

	require.NoError(t, repo.UpdateTablePartitions(ctx))

	exists := func(name string) bool {
		var ok bool
		require.NoError(t, pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM pg_tables WHERE tablename = $1)`, name).Scan(&ok))
		return ok
	}

	for hour, kept := range messageHours {
		assert.Equal(t, kept, exists("v1_stream_message_"+hour), "message partition %s", hour)
	}

	for day, kept := range cursorDays {
		assert.Equal(t, kept, exists("v1_stream_producer_cursor_"+day), "cursor partition %s", day)
	}

	assert.True(t, exists("v1_stream_message_"+now.Add(streamMessagePartitionsAhead).Format("2006010215")), "hourly partitions are created a day ahead")
}

func TestDeleteIdleStreamTopics_UsesEachTenantsRetention(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()
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
	rows, err := pool.Query(ctx, `SELECT topic FROM v1_stream_topic ORDER BY topic`)
	require.NoError(t, err)
	for rows.Next() {
		var topic string
		require.NoError(t, rows.Scan(&topic))
		remaining = append(remaining, topic)
	}
	require.NoError(t, rows.Err())

	assert.Equal(t, []string{"active", "idle-long"}, remaining)
}
