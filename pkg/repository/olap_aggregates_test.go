//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
	"fmt"
	"math/rand"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/config/limits"
	"github.com/hatchet-dev/hatchet/pkg/repository/fairpool"
	"github.com/hatchet-dev/hatchet/pkg/repository/olapagg"
	"github.com/hatchet-dev/hatchet/pkg/validator"
)

// Aggregate rows plus their unfolded deltas match the source rows while a writer and the
// compactors race each other, and the aggregate rows alone match once compaction catches up.
func TestOLAPAggregatesMatchSourceRows(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()

	require.NoError(t, createOLAPRepositoryManagingAllPartitions(t, pool).UpdateTablePartitions(ctx))

	l := zerolog.Nop()
	stopCompactors := olapagg.New(pool, &l, olapagg.Opts{
		BatchSize:    50,
		Partitions:   NUM_PARTITIONS,
		PollInterval: 20 * time.Millisecond,
	}).Start()
	defer stopCompactors()

	writer := newAggregateSourceWriter(pool)
	writesDone := make(chan error)

	go func() {
		writesDone <- writer.writeTransactions(ctx, 300)
	}()

	checks := 0

	for writing := true; writing; checks++ {
		select {
		case err := <-writesDone:
			require.NoError(t, err)
			writing = false
		default:
		}

		requireAggregatesMatchSource(t, pool, true)
	}

	t.Logf("compared aggregates plus deltas with the source %d times during writes", checks)

	require.Eventually(t, func() bool {
		var unfoldedDeltas int

		err := pool.QueryRow(ctx, `
SELECT
    (SELECT COUNT(*) FROM v1_agg_task_statuses_minute_delta_olap) +
    (SELECT COUNT(*) FROM v1_agg_task_events_minute_delta_olap) +
    (SELECT COUNT(*) FROM v1_agg_events_minute_delta_olap)`).Scan(&unfoldedDeltas)

		return err == nil && unfoldedDeltas == 0
	}, time.Minute, 50*time.Millisecond)

	requireAggregatesMatchSource(t, pool, false)
}

// A task event is counted in the minute it happened rather than the minute its task was created,
// and an event stamped in the future is counted in the minute it was written.
func TestOLAPTaskEventsAggregateBucketsByEventTime(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()

	require.NoError(t, createOLAPRepositoryManagingAllPartitions(t, pool).UpdateTablePartitions(ctx))

	now := time.Now().UTC()
	taskInsertedAt := now.Add(-3 * time.Hour)
	finishedAt := now.Add(-time.Minute)

	// the task may have been created yesterday
	yesterday := pgtype.Date{Time: now.AddDate(0, 0, -1), Valid: true}
	_, err := pool.Exec(ctx, `
SELECT
    create_v1_range_partition('v1_task_events_olap', $1::DATE),
    create_v1_range_partition('v1_agg_task_events_minute_olap', $1::DATE, 70)`, yesterday)
	require.NoError(t, err)

	tenantID, workflowID := uuid.New(), uuid.New()

	var futureEventWrittenAt time.Time

	err = pool.QueryRow(ctx, `
WITH inserted AS (
    INSERT INTO v1_task_events_olap (tenant_id, task_id, task_inserted_at, event_type, workflow_id, event_timestamp, readable_status)
    VALUES
        ($1, 1, $3, 'QUEUED', $2, $3, 'QUEUED'),
        ($1, 1, $3, 'FINISHED', $2, $4, 'COMPLETED'),
        ($1, 2, $3, 'FAILED', $2, $5, 'FAILED')
    RETURNING inserted_at
)
SELECT MAX(inserted_at) FROM inserted`,
		tenantID, workflowID, taskInsertedAt, finishedAt, now.Add(72*time.Hour)).Scan(&futureEventWrittenAt)
	require.NoError(t, err)

	l := zerolog.Nop()
	stopCompactors := olapagg.New(pool, &l, olapagg.Opts{
		BatchSize:    50,
		Partitions:   NUM_PARTITIONS,
		PollInterval: 20 * time.Millisecond,
	}).Start()
	defer stopCompactors()

	require.Eventually(t, func() bool {
		var unfoldedDeltas int
		err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM v1_agg_task_events_minute_delta_olap`).Scan(&unfoldedDeltas)
		return err == nil && unfoldedDeltas == 0
	}, time.Minute, 50*time.Millisecond)

	type counts struct{ queued, completed, failed int64 }

	bucketKey := func(ts time.Time) string { return ts.UTC().Truncate(time.Minute).Format(time.RFC3339) }

	expected := map[string]counts{}
	add := func(ts time.Time, c counts) {
		e := expected[bucketKey(ts)]
		expected[bucketKey(ts)] = counts{e.queued + c.queued, e.completed + c.completed, e.failed + c.failed}
	}
	add(taskInsertedAt, counts{queued: 1})
	add(finishedAt, counts{completed: 1})
	add(futureEventWrittenAt, counts{failed: 1})

	rows, err := pool.Query(ctx, `
SELECT bucket, queued_count, completed_count, failed_count
FROM v1_agg_task_events_minute_olap
WHERE tenant_id = $1`, tenantID)
	require.NoError(t, err)

	actual := map[string]counts{}
	var bucket time.Time
	var c counts
	_, err = pgx.ForEachRow(rows, []any{&bucket, &c.queued, &c.completed, &c.failed}, func() error {
		actual[bucketKey(bucket)] = c
		return nil
	})
	require.NoError(t, err)

	require.Equal(t, expected, actual)
}

func createOLAPRepositoryManagingAllPartitions(t *testing.T, pool *pgxpool.Pool) OLAPRepository {
	logger := zerolog.Nop()

	shared, cleanupShared := newSharedRepository(
		fairpool.Ungated(pool),
		pool,
		validator.NewDefaultValidator(),
		&logger,
		PayloadStoreRepositoryOpts{},
		limits.LimitConfigFile{},
		false,
		time.Minute,
		false,
		false,
	)
	t.Cleanup(func() { _ = cleanupShared() })

	return newOLAPRepository(shared, 24*time.Hour, StatusUpdateBatchSizeLimits{Task: 1000, DAG: 1000}, true)
}

var readableStatuses = []string{"QUEUED", "RUNNING", "COMPLETED", "CANCELLED", "FAILED", "EVICTED"}

// aggregateSourceWriter writes to the tables the aggregate triggers watch: run inserts, run
// status changes, run deletions, task event and event inserts, and transactions that roll back.
type aggregateSourceWriter struct {
	pool      *pgxpool.Pool
	rng       *rand.Rand
	tenants   []uuid.UUID
	workflows []uuid.UUID
	runs      []aggregateSourceRun
}

type aggregateSourceRun struct {
	id                   int64
	tenantID, workflowID uuid.UUID
	insertedAt           time.Time
}

func newAggregateSourceWriter(pool *pgxpool.Pool) *aggregateSourceWriter {
	w := &aggregateSourceWriter{pool: pool, rng: rand.New(rand.NewSource(1))} // nolint:gosec

	// more tenants than delta partitions, so that every partition gets deltas
	for range 40 {
		w.tenants = append(w.tenants, uuid.New())
	}

	for range 5 {
		w.workflows = append(w.workflows, uuid.New())
	}

	return w
}

func (w *aggregateSourceWriter) writeTransactions(ctx context.Context, count int) error {
	for range count {
		if err := w.writeTransaction(ctx, w.rng.Intn(10) == 0); err != nil {
			return err
		}
	}

	return nil
}

func (w *aggregateSourceWriter) writeTransaction(ctx context.Context, rollback bool) error {
	tx, err := w.pool.Begin(ctx)
	if err != nil {
		return err
	}

	defer func() { _ = tx.Rollback(ctx) }()

	// Timestamps stay within today, the only day the aggregates have partitions for.
	now := time.Now().UTC()
	earliest := now.Add(-10 * time.Minute)
	if startOfToday := now.Truncate(24 * time.Hour); earliest.Before(startOfToday) {
		earliest = startOfToday
	}
	randomTimestamp := func() time.Time {
		return earliest.Add(time.Duration(w.rng.Int63n(int64(now.Sub(earliest)) + 1)))
	}

	randomStatus := func() string { return readableStatuses[w.rng.Intn(len(readableStatuses))] }
	randomTenant := func() uuid.UUID { return w.tenants[w.rng.Intn(len(w.tenants))] }
	randomWorkflow := func() uuid.UUID { return w.workflows[w.rng.Intn(len(w.workflows))] }

	var insertedRuns []aggregateSourceRun

	for range 1 + w.rng.Intn(20) {
		run := aggregateSourceRun{id: w.rng.Int63(), tenantID: randomTenant(), workflowID: randomWorkflow(), insertedAt: randomTimestamp()}

		if _, err := tx.Exec(ctx, `
INSERT INTO v1_runs_olap (tenant_id, id, inserted_at, readable_status, kind, workflow_id, workflow_version_id)
VALUES ($1, $2, $3, $4, 'TASK', $5, gen_random_uuid())`,
			run.tenantID, run.id, run.insertedAt, randomStatus(), run.workflowID); err != nil {
			return err
		}

		insertedRuns = append(insertedRuns, run)
	}

	if len(w.runs) > 0 {
		var ids []int64
		var insertedAts []time.Time
		var newStatuses []string

		for range w.rng.Intn(15) {
			run := w.runs[w.rng.Intn(len(w.runs))]

			if slices.Contains(ids, run.id) {
				continue
			}

			ids = append(ids, run.id)
			insertedAts = append(insertedAts, run.insertedAt)
			newStatuses = append(newStatuses, randomStatus())
		}

		if _, err := tx.Exec(ctx, `
UPDATE v1_runs_olap r
SET readable_status = u.status::v1_readable_status_olap
FROM unnest($1::BIGINT[], $2::TIMESTAMPTZ[], $3::TEXT[]) AS u(id, inserted_at, status)
WHERE r.id = u.id AND r.inserted_at = u.inserted_at`,
			ids, insertedAts, newStatuses); err != nil {
			return err
		}

		if w.rng.Intn(10) == 0 {
			run := w.runs[w.rng.Intn(len(w.runs))]

			if _, err := tx.Exec(ctx, `DELETE FROM v1_runs_olap WHERE id = $1 AND inserted_at = $2`, run.id, run.insertedAt); err != nil {
				return err
			}
		}
	}

	var taskEvents [][]any

	for range 1 + w.rng.Intn(50) {
		taskEvents = append(taskEvents, []any{
			randomTenant(), w.rng.Int63(), randomTimestamp(), "FINISHED", randomWorkflow(), randomTimestamp(), randomStatus(), w.rng.Intn(4) == 0,
		})
	}

	if _, err := tx.CopyFrom(ctx, pgx.Identifier{"v1_task_events_olap"},
		[]string{"tenant_id", "task_id", "task_inserted_at", "event_type", "workflow_id", "event_timestamp", "readable_status", "is_dag_orchestrator"},
		pgx.CopyFromRows(taskEvents)); err != nil {
		return err
	}

	for range w.rng.Intn(10) {
		if _, err := tx.Exec(ctx, `INSERT INTO v1_events_olap (tenant_id, seen_at, key, payload) VALUES ($1, $2, 'key', '{}')`,
			randomTenant(), randomTimestamp()); err != nil {
			return err
		}
	}

	if rollback {
		return tx.Rollback(ctx)
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	w.runs = append(w.runs, insertedRuns...)

	return nil
}

type aggregateUnderTest struct {
	table      string
	deltaTable string
	keys       string
	counts     []string

	// sourceCounts counts every source row into the aggregate's keys and counts
	sourceCounts string
}

var aggregatesUnderTest = []aggregateUnderTest{
	{
		table:      "v1_agg_task_statuses_minute_olap",
		deltaTable: "v1_agg_task_statuses_minute_delta_olap",
		keys:       "tenant_id, bucket, workflow_id",
		counts:     []string{"queued_count", "running_count", "completed_count", "cancelled_count", "failed_count", "evicted_count"},
		sourceCounts: `
SELECT
    tenant_id,
    date_bin('1 minute', inserted_at, TIMESTAMPTZ '1970-01-01 00:00:00+00'),
    workflow_id,
    COUNT(*) FILTER (WHERE readable_status = 'QUEUED'),
    COUNT(*) FILTER (WHERE readable_status = 'RUNNING'),
    COUNT(*) FILTER (WHERE readable_status = 'COMPLETED'),
    COUNT(*) FILTER (WHERE readable_status = 'CANCELLED'),
    COUNT(*) FILTER (WHERE readable_status = 'FAILED'),
    COUNT(*) FILTER (WHERE readable_status = 'EVICTED')
FROM v1_runs_olap
GROUP BY 1, 2, 3`,
	},
	{
		table:      "v1_agg_task_events_minute_olap",
		deltaTable: "v1_agg_task_events_minute_delta_olap",
		keys:       "tenant_id, bucket, workflow_id",
		counts:     []string{"queued_count", "running_count", "completed_count", "cancelled_count", "failed_count"},
		// inserted_at defaults to the inserting transaction's time, which caps the event time
		sourceCounts: `
SELECT
    tenant_id,
    date_bin('1 minute', LEAST(event_timestamp, inserted_at), TIMESTAMPTZ '1970-01-01 00:00:00+00'),
    workflow_id,
    COUNT(*) FILTER (WHERE readable_status = 'QUEUED'),
    COUNT(*) FILTER (WHERE readable_status = 'RUNNING'),
    COUNT(*) FILTER (WHERE readable_status = 'COMPLETED'),
    COUNT(*) FILTER (WHERE readable_status = 'CANCELLED'),
    COUNT(*) FILTER (WHERE readable_status = 'FAILED')
FROM v1_task_events_olap
WHERE NOT is_dag_orchestrator
GROUP BY 1, 2, 3`,
	},
	{
		table:      "v1_agg_events_minute_olap",
		deltaTable: "v1_agg_events_minute_delta_olap",
		keys:       "tenant_id, bucket",
		counts:     []string{"event_count"},
		sourceCounts: `
SELECT
    tenant_id,
    date_bin('1 minute', seen_at, TIMESTAMPTZ '1970-01-01 00:00:00+00'),
    COUNT(*)
FROM v1_events_olap
GROUP BY 1, 2`,
	},
}

// requireAggregatesMatchSource compares every aggregate with the counts of its source rows. With
// includeDeltas, an aggregate's counts are its rows plus its unfolded delta rows.
func requireAggregatesMatchSource(t *testing.T, pool *pgxpool.Pool, includeDeltas bool) {
	t.Helper()

	for _, agg := range aggregatesUnderTest {
		counts := strings.Join(agg.counts, ", ")

		nonZeroConditions := make([]string, len(agg.counts))
		summedCounts := make([]string, len(agg.counts))

		for i, count := range agg.counts {
			nonZeroConditions[i] = count + " <> 0"
			summedCounts[i] = fmt.Sprintf("SUM(%[1]s)::BIGINT AS %[1]s", count)
		}

		// Keys whose counts are all zero are left out on both sides: a delta can bring an aggregate
		// row back to zero, and a source row can be in a status the aggregate does not count.
		anyCountNonZero := strings.Join(nonZeroConditions, " OR ")

		aggregateRows := fmt.Sprintf("SELECT %s, %s FROM %s", agg.keys, counts, agg.table)

		if includeDeltas {
			aggregateRows = fmt.Sprintf(`
SELECT %[1]s, %[2]s
FROM (SELECT %[1]s, %[3]s FROM %[4]s UNION ALL SELECT %[1]s, %[3]s FROM %[5]s) rows_and_deltas
GROUP BY %[1]s`, agg.keys, strings.Join(summedCounts, ", "), counts, agg.table, agg.deltaTable)
		}

		expected := fmt.Sprintf("SELECT * FROM (%s) s (%s, %s) WHERE %s", agg.sourceCounts, agg.keys, counts, anyCountNonZero)
		actual := fmt.Sprintf("SELECT * FROM (%s) a WHERE %s", aggregateRows, anyCountNonZero)

		rows, err := pool.Query(context.Background(), fmt.Sprintf(`
SELECT 'only in source' AS side, * FROM ((%[1]s) EXCEPT ALL (%[2]s)) d
UNION ALL
SELECT 'only in aggregate', * FROM ((%[2]s) EXCEPT ALL (%[1]s)) d
LIMIT 20`, expected, actual))
		require.NoError(t, err)

		mismatches, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) ([]any, error) { return row.Values() })
		require.NoError(t, err)

		require.Empty(t, mismatches, "%s does not match its source rows", agg.table)
	}
}
