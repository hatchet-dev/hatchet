//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/validator"
)

func createStreamsRepository(pool *pgxpool.Pool) *streamsRepositoryImpl {
	logger := zerolog.Nop()

	return &streamsRepositoryImpl{
		sharedRepository: &sharedRepository{
			pool:    pool,
			ddlPool: pool,
			v:       validator.NewDefaultValidator(),
			l:       &logger,
			queries: sqlcv1.New(),
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

	repo := createStreamsRepository(pool)
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

	repo := createStreamsRepository(pool)
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

	repo := createStreamsRepository(pool)
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

	repo := createStreamsRepository(pool)
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
