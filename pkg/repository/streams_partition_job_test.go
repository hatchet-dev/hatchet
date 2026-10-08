//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// Shared partitions are dropped once the longest retention has passed them,
// payloads StreamPayloadRetentionGrace later. Its own database, since the job
// acts on every tenant's partitions.
func TestStreamsPartitionJob(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()
	repo := createTaskRepository(pool)
	config := defaultLimitTestConfig()
	config.DefaultTenantRetentionPeriod = "24h"
	repo.m = newTestTenantLimitRepository(pool, config)
	queries := sqlcv1.New()

	longTenant := createLimitTestTenant(t, pool)
	setStreamRetentionHours(t, pool, longTenant, 48)

	now := time.Now().UTC()
	today := now.Truncate(24 * time.Hour)
	hoursAgo := func(h int) pgtype.Timestamptz {
		return pgtype.Timestamptz{Time: now.Add(-time.Duration(h) * time.Hour), Valid: true}
	}

	// the migration only seeds partitions from the current hour on
	require.NoError(t, queries.CreateStreamMessagePartitions(ctx, pool, sqlcv1.CreateStreamMessagePartitionsParams{Fromtime: hoursAgo(60), Totime: hoursAgo(0)}))
	require.NoError(t, queries.CreateStreamPayloadPartitions(ctx, pool, sqlcv1.CreateStreamPayloadPartitionsParams{Fromtime: hoursAgo(80), Totime: hoursAgo(0)}))

	for _, d := range []int{-2, -10} {
		_, err := queries.CreatePartitions(ctx, pool, pgtype.Date{Time: today.AddDate(0, 0, d), Valid: true})
		require.NoError(t, err)
	}

	require.NoError(t, repo.UpdateTablePartitions(ctx))

	existing := map[string]bool{}
	everything := pgtype.Timestamptz{InfinityModifier: pgtype.Infinity, Valid: true}

	messages, err := queries.ListStreamMessagePartitionsBefore(ctx, pool, everything)
	require.NoError(t, err)
	for _, p := range messages {
		existing[p.PartitionName] = true
	}

	payloads, err := queries.ListStreamPayloadPartitionsBefore(ctx, pool, everything)
	require.NoError(t, err)
	for _, p := range payloads {
		existing[p.PartitionName] = true
	}

	cursors, err := queries.ListStreamProducerCursorPartitionsBeforeDate(ctx, pool, pgtype.Date{InfinityModifier: pgtype.Infinity, Valid: true})
	require.NoError(t, err)
	for _, p := range cursors {
		existing[p.PartitionName] = true
	}

	hourTable := func(table string, ago time.Duration) string {
		return table + "_" + now.Add(-ago).Format("2006010215")
	}

	cursorTable := func(day time.Time) string {
		return "v1_stream_producer_cursor_" + day.Format("20060102")
	}

	kept := map[string]bool{
		// messages follow the longest retention (48h)
		hourTable("v1_stream_message", 30*time.Hour):                  true,
		hourTable("v1_stream_message", 60*time.Hour):                  false,
		hourTable("v1_stream_message", -streamMessagePartitionsAhead): true,
		// payloads are kept StreamPayloadRetentionGrace (24h) longer, to 72h
		hourTable("v1_stream_payload", 60*time.Hour):                  true,
		hourTable("v1_stream_payload", 80*time.Hour):                  false,
		hourTable("v1_stream_payload", -streamMessagePartitionsAhead): true,
		// cursors are kept streamProducerCursorRetention (3 days), whatever the message retention
		cursorTable(today.AddDate(0, 0, -2)):  true,
		cursorTable(today.AddDate(0, 0, -10)): false,
	}

	for name, want := range kept {
		assert.Equal(t, want, existing[name], name)
	}
}
