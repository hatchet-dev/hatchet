//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func TestAddMessageEnsuringQueueCreatesMissingQueue(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()
	q := sqlcv1.New()
	const name = "mq-selfheal-missing-queue"

	err := q.AddMessageEnsuringQueue(ctx, pool, sqlcv1.AddMessageEnsuringQueueParams{
		Queueid:     name,
		Payload:     []byte(`{"hello":"world"}`),
		Durable:     false,
		Autodeleted: true,
		Exclusive:   false,
	})
	require.NoError(t, err, "AddMessageEnsuringQueue must create the parent queue, not raise MessageQueueItem_queueId_fkey (23503)")

	var autoDeleted, exclusive bool
	err = pool.QueryRow(ctx, `SELECT "autoDeleted", "exclusive" FROM "MessageQueue" WHERE "name" = $1`, name).Scan(&autoDeleted, &exclusive)
	require.NoError(t, err)
	assert.True(t, autoDeleted, "the recreated queue must mirror the supplied bind attributes")
	assert.False(t, exclusive, "the recreated queue must mirror the supplied exclusive flag")

	var items int
	err = pool.QueryRow(ctx, `SELECT count(*) FROM "MessageQueueItem" WHERE "queueId" = $1`, name).Scan(&items)
	require.NoError(t, err)
	assert.Equal(t, 1, items)
}

func TestAddMessageEnsuringQueueSurvivesQueueGC(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()
	q := sqlcv1.New()
	const name = "mq-selfheal-gc-queue"

	_, err := q.UpsertMessageQueue(ctx, pool, sqlcv1.UpsertMessageQueueParams{
		Name:        name,
		Durable:     false,
		Autodeleted: true,
		Exclusive:   false,
	})
	require.NoError(t, err)

	_, err = pool.Exec(ctx,
		`UPDATE "MessageQueue" SET "lastActive" = NOW() - INTERVAL '2 hours' WHERE "name" = $1`,
		name,
	)
	require.NoError(t, err)

	require.NoError(t, q.CleanupMessageQueue(ctx, pool))

	var reaped int
	err = pool.QueryRow(ctx, `SELECT count(*) FROM "MessageQueue" WHERE "name" = $1`, name).Scan(&reaped)
	require.NoError(t, err)
	require.Equal(t, 0, reaped, "queue should have been reaped, setting up the race")

	err = q.AddMessageEnsuringQueue(ctx, pool, sqlcv1.AddMessageEnsuringQueueParams{
		Queueid:     name,
		Payload:     []byte(`{"hello":"world"}`),
		Durable:     false,
		Autodeleted: true,
	})
	require.NoError(t, err, "AddMessageEnsuringQueue must recreate a GC'd parent, not raise 23503")

	var exists int
	err = pool.QueryRow(ctx, `SELECT count(*) FROM "MessageQueue" WHERE "name" = $1`, name).Scan(&exists)
	require.NoError(t, err)
	assert.Equal(t, 1, exists, "the parent queue must be recreated by AddMessageEnsuringQueue")
}

func TestAddMessageEnsuringQueueRecreatesReapedExclusiveQueue(t *testing.T) {
	// Regression for the autoDeleted+exclusive gap: CleanupMessageQueue reaps on
	// "autoDeleted" alone, so exclusive auto-deleted queues — the dispatcher queue
	// (expirable ⇒ autoDeleted, exclusive) and controller consumer queues — are
	// reap-eligible too. A message after a reap must recreate the parent with
	// exclusive preserved, not raise 23503. (The `!exclusive` gate that this test
	// guards against left exactly these queues exposed.)
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()
	q := sqlcv1.New()
	const name = "mq-selfheal-exclusive-gc-queue"

	_, err := q.UpsertMessageQueue(ctx, pool, sqlcv1.UpsertMessageQueueParams{
		Name:        name,
		Durable:     true,
		Autodeleted: true,
		Exclusive:   true,
	})
	require.NoError(t, err)

	_, err = pool.Exec(ctx,
		`UPDATE "MessageQueue" SET "lastActive" = NOW() - INTERVAL '2 hours' WHERE "name" = $1`,
		name,
	)
	require.NoError(t, err)

	require.NoError(t, q.CleanupMessageQueue(ctx, pool))

	var reaped int
	err = pool.QueryRow(ctx, `SELECT count(*) FROM "MessageQueue" WHERE "name" = $1`, name).Scan(&reaped)
	require.NoError(t, err)
	require.Equal(t, 0, reaped, "an exclusive auto-deleted queue must also be reaped — autoDeleted alone triggers cleanup")

	err = q.AddMessageEnsuringQueue(ctx, pool, sqlcv1.AddMessageEnsuringQueueParams{
		Queueid:     name,
		Payload:     []byte(`{"hello":"world"}`),
		Durable:     true,
		Autodeleted: true,
		Exclusive:   true,
	})
	require.NoError(t, err, "AddMessageEnsuringQueue must recreate a reaped EXCLUSIVE queue, not raise 23503")

	var exclusive, consumerIsNull bool
	err = pool.QueryRow(ctx,
		`SELECT "exclusive", "exclusiveConsumerId" IS NULL FROM "MessageQueue" WHERE "name" = $1`,
		name,
	).Scan(&exclusive, &consumerIsNull)
	require.NoError(t, err)
	assert.True(t, exclusive, "the recreated queue must preserve exclusive=true")
	assert.True(t, consumerIsNull, "a self-healed exclusive queue has no consumer until the next BindQueue")

	var items int
	err = pool.QueryRow(ctx, `SELECT count(*) FROM "MessageQueueItem" WHERE "queueId" = $1`, name).Scan(&items)
	require.NoError(t, err)
	assert.Equal(t, 1, items, "the message must be stored against the recreated queue")
}

func TestBindRefreshesLastActive(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()
	q := sqlcv1.New()
	const name = "mq-selfheal-bind-refresh-queue"

	bind := func() {
		_, err := q.UpsertMessageQueue(ctx, pool, sqlcv1.UpsertMessageQueueParams{
			Name:        name,
			Durable:     false,
			Autodeleted: true,
			Exclusive:   false,
		})
		require.NoError(t, err)
	}

	bind()

	_, err := pool.Exec(ctx,
		`UPDATE "MessageQueue" SET "lastActive" = NOW() - INTERVAL '2 hours' WHERE "name" = $1`,
		name,
	)
	require.NoError(t, err)

	bind()

	var lastActive time.Time
	err = pool.QueryRow(ctx, `SELECT "lastActive" FROM "MessageQueue" WHERE "name" = $1`, name).Scan(&lastActive)
	require.NoError(t, err)
	assert.WithinDuration(t, time.Now(), lastActive, time.Minute)

	require.NoError(t, q.CleanupMessageQueue(ctx, pool))

	var exists int
	err = pool.QueryRow(ctx, `SELECT count(*) FROM "MessageQueue" WHERE "name" = $1`, name).Scan(&exists)
	require.NoError(t, err)
	assert.Equal(t, 1, exists, "a recently rebound queue must survive cleanup")
}

func mqRepoForTest(pool *pgxpool.Pool) *messageQueueRepository {
	l := zerolog.Nop()

	return &messageQueueRepository{
		sharedRepository: &sharedRepository{
			pool:    pool,
			queries: sqlcv1.New(),
			l:       &l,
		},
	}
}

func TestAddMessageEnsuringQueueFastPathSkipsParentUpsert(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()
	repo := mqRepoForTest(pool)
	q := sqlcv1.New()
	const name = "mq-optimistic-fast-path-queue"

	_, err := q.UpsertMessageQueue(ctx, pool, sqlcv1.UpsertMessageQueueParams{
		Name:        name,
		Durable:     false,
		Autodeleted: true,
		Exclusive:   false,
	})
	require.NoError(t, err)

	// Backdate lastActive so any parent-row write by the publish is detectable.
	_, err = pool.Exec(ctx,
		`UPDATE "MessageQueue" SET "lastActive" = NOW() - INTERVAL '30 minutes' WHERE "name" = $1`,
		name,
	)
	require.NoError(t, err)

	err = repo.AddMessageEnsuringQueue(ctx, name, []byte(`{"hello":"world"}`), false, true, false)
	require.NoError(t, err)

	var lastActive time.Time
	err = pool.QueryRow(ctx, `SELECT "lastActive" FROM "MessageQueue" WHERE "name" = $1`, name).Scan(&lastActive)
	require.NoError(t, err)
	assert.Less(t, time.Since(lastActive), 45*time.Minute)
	assert.Greater(t, time.Since(lastActive), 15*time.Minute,
		"a publish to an existing queue must not rewrite the parent row (no ON CONFLICT DO UPDATE on the hot path)")

	var items int
	err = pool.QueryRow(ctx, `SELECT count(*) FROM "MessageQueueItem" WHERE "queueId" = $1`, name).Scan(&items)
	require.NoError(t, err)
	assert.Equal(t, 1, items)
}

func TestAddMessageEnsuringQueueHealsReapedParentOnFKViolation(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()
	repo := mqRepoForTest(pool)
	q := sqlcv1.New()
	const name = "mq-optimistic-heal-queue"

	_, err := q.UpsertMessageQueue(ctx, pool, sqlcv1.UpsertMessageQueueParams{
		Name:        name,
		Durable:     true,
		Autodeleted: true,
		Exclusive:   true,
	})
	require.NoError(t, err)

	err = repo.AddMessageEnsuringQueue(ctx, name, []byte(`{"seq":1}`), true, true, true)
	require.NoError(t, err)

	// Reap the queue mid-stream, exactly as CleanupMessageQueue does in the race.
	_, err = pool.Exec(ctx,
		`UPDATE "MessageQueue" SET "lastActive" = NOW() - INTERVAL '2 hours' WHERE "name" = $1`,
		name,
	)
	require.NoError(t, err)
	require.NoError(t, q.CleanupMessageQueue(ctx, pool))

	var reaped int
	err = pool.QueryRow(ctx, `SELECT count(*) FROM "MessageQueue" WHERE "name" = $1`, name).Scan(&reaped)
	require.NoError(t, err)
	require.Equal(t, 0, reaped, "queue should have been reaped, setting up the race")

	err = repo.AddMessageEnsuringQueue(ctx, name, []byte(`{"seq":2}`), true, true, true)
	require.NoError(t, err, "a publish against a reaped queue must self-heal, not surface 23503")

	var exclusive bool
	var lastActive time.Time
	err = pool.QueryRow(ctx,
		`SELECT "exclusive", "lastActive" FROM "MessageQueue" WHERE "name" = $1`, name,
	).Scan(&exclusive, &lastActive)
	require.NoError(t, err)
	assert.True(t, exclusive, "the healed queue must preserve the supplied bind attributes")
	assert.WithinDuration(t, time.Now(), lastActive, time.Minute,
		"the heal must refresh lastActive so the queue is not immediately reap-eligible again")

	var items int
	err = pool.QueryRow(ctx, `SELECT count(*) FROM "MessageQueueItem" WHERE "queueId" = $1`, name).Scan(&items)
	require.NoError(t, err)
	assert.Equal(t, 1, items, "the post-reap message must be stored against the recreated queue")
}

func TestAddMessageEnsuringQueueNonAutoDeletedPropagatesFKViolation(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()
	repo := mqRepoForTest(pool)

	err := repo.AddMessageEnsuringQueue(ctx, "mq-optimistic-missing-static-queue", []byte(`{}`), true, false, false)
	require.Error(t, err, "non-auto-deleted queues are never reaped, so a missing parent is a real bug and must not be masked by a self-heal")
	assert.True(t, isForeignKeyViolation(err))
}
