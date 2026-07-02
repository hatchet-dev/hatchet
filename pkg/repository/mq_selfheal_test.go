//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
	"testing"
	"time"

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
