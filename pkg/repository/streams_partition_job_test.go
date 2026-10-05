//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// Shared hourly partitions are dropped once the longest retention has passed
// them; tenants with a shorter retention have their expired messages deleted.
// Its own database, since the job acts on every tenant's rows and partitions.
func TestStreamsPartitionJob(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()
	repo := createTaskRepository(pool)
	config := defaultLimitTestConfig()
	config.DefaultTenantRetentionPeriod = "24h"
	repo.m = newTestTenantLimitRepository(pool, config)

	shortTenant := createLimitTestTenant(t, pool)
	longTenant := createLimitTestTenant(t, pool)
	defaultTenant := createLimitTestTenant(t, pool) // no STREAM_RETENTION row: the 24h default
	setStreamRetentionHours(t, pool, shortTenant, 2)
	setStreamRetentionHours(t, pool, longTenant, 48)

	// a tenant without a retention row is found through its topics
	_, err := pool.Exec(ctx, `INSERT INTO v1_stream_topic (tenant_id, topic) VALUES ($1, 't')`, defaultTenant)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `SELECT create_v1_hourly_range_partition('v1_stream_message', NOW() - make_interval(hours => h)) FROM unnest(ARRAY[5, 30, 60]) AS h`)
	require.NoError(t, err)

	payloadAges := []time.Duration{30 * time.Hour, 60 * time.Hour, 80 * time.Hour}

	for _, ago := range payloadAges {
		_, err = pool.Exec(ctx, `SELECT create_v1_hourly_range_partition('v1_stream_payload', NOW() - make_interval(secs => $1))`, ago.Seconds())
		require.NoError(t, err)
	}

	// one message per tenant, 5h, 30h and 60h old
	_, err = pool.Exec(ctx, `
		INSERT INTO v1_stream_message (id, tenant_id, topic, payload, producer_id, producer_seq, inserted_at)
		SELECT row_number() OVER (), tenant, 't', 'm', 'p', 0, NOW() - make_interval(hours => h)
		FROM unnest($1::uuid[]) AS tenant, unnest(ARRAY[5, 30, 60]) AS h`, []uuid.UUID{shortTenant, longTenant, defaultTenant})
	require.NoError(t, err)

	now := time.Now().UTC()
	today := now.Truncate(24 * time.Hour)

	for _, d := range []int{-2, -10} {
		_, err := sqlcv1.New().CreatePartitions(ctx, pool, pgtype.Date{Time: today.AddDate(0, 0, d), Valid: true})
		require.NoError(t, err)
	}

	require.NoError(t, repo.UpdateTablePartitions(ctx))

	ages := func(tenant uuid.UUID) []int {
		var out []int
		rows, err := pool.Query(ctx, `SELECT round(extract(epoch FROM NOW() - inserted_at) / 3600)::int FROM v1_stream_message WHERE tenant_id = $1 ORDER BY 1`, tenant)
		require.NoError(t, err)
		for rows.Next() {
			var h int
			require.NoError(t, rows.Scan(&h))
			out = append(out, h)
		}
		require.NoError(t, rows.Err())
		return out
	}

	assert.Empty(t, ages(shortTenant), "2h retention: everything deleted or dropped")
	assert.Equal(t, []int{5}, ages(defaultTenant), "24h default: the 30h message deleted")
	assert.Equal(t, []int{5, 30}, ages(longTenant), "48h retention: kept, the 60h one went with its partition")

	exists := func(name string) bool {
		var ok bool
		require.NoError(t, pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM pg_class WHERE relname = $1)`, name).Scan(&ok))
		return ok
	}

	hourTable := func(ago time.Duration) string {
		return "v1_stream_message_" + now.Add(-ago).Format("2006010215")
	}

	payloadTable := func(ago time.Duration) string {
		return "v1_stream_payload_" + now.Add(-ago).Format("2006010215")
	}

	cursorTable := func(day time.Time) string {
		return "v1_stream_producer_cursor_" + day.Format("20060102")
	}

	kept := map[string]bool{
		// partitions follow the longest retention (48h)
		hourTable(5 * time.Hour):                 true,
		hourTable(30 * time.Hour):                true,
		hourTable(60 * time.Hour):                false,
		hourTable(-streamMessagePartitionsAhead): true,
		// payloads are kept StreamPayloadRetentionGrace (24h) past the longest retention, so 60h survives where messages don't
		payloadTable(payloadAges[0]):                true,
		payloadTable(payloadAges[1]):                true,
		payloadTable(payloadAges[2]):                false,
		payloadTable(-streamMessagePartitionsAhead): true,
		// cursors are kept streamProducerCursorRetention (3 days), whatever the message retention
		cursorTable(today.AddDate(0, 0, -2)):  true,
		cursorTable(today.AddDate(0, 0, -10)): false,
	}

	for name, want := range kept {
		assert.Equal(t, want, exists(name), name)
	}

	// every publish updates its producer's cursor row, so both the seeded and
	// the job-created partitions leave room for those updates to stay HOT
	for _, partition := range []string{cursorTable(today), cursorTable(today.AddDate(0, 0, 1))} {
		var options []string
		require.NoError(t, pool.QueryRow(ctx, `SELECT COALESCE(reloptions, '{}') FROM pg_class WHERE relname = $1`, partition).Scan(&options))
		assert.Contains(t, options, "fillfactor=80", partition)
	}

	// messages are insert-only apart from retention deletes, so they keep the default autovacuum settings
	for _, partition := range []string{hourTable(0), hourTable(-streamMessagePartitionsAhead)} {
		var options []string
		require.NoError(t, pool.QueryRow(ctx, `SELECT COALESCE(reloptions, '{}') FROM pg_class WHERE relname = $1`, partition).Scan(&options))
		assert.Empty(t, options, partition)
	}
}
