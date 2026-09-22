//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// createOLAPPartitionsForDate mirrors what UpdateTablePartitions does for today and
// tomorrow, for an arbitrary date, so tests can write rows with an old inserted_at.
func createOLAPPartitionsForDate(t *testing.T, ctx context.Context, repo *OLAPRepositoryImpl, pool *pgxpool.Pool, date time.Time) {
	t.Helper()

	created, err := repo.queries.CreateOLAPPartitions(ctx, pool, sqlcv1.CreateOLAPPartitionsParams{
		Date:       pgtype.Date{Time: date, Valid: true},
		Partitions: NUM_PARTITIONS,
	})
	require.NoError(t, err)

	if created.V1PayloadsOlap > 0 {
		require.NoError(t, createExternalIdUniqueConstraintsOnDailyPartitions(ctx, pool, "v1_payloads_olap", date))
	}
}

// seedTaskWithStatus creates a standalone task inserted at insertedAt and drives it to
// the given readable status through the OLAP event write path.
func seedTaskWithStatus(t *testing.T, ctx context.Context, repo *OLAPRepositoryImpl, pool *pgxpool.Pool, tenantId uuid.UUID, taskId int64, insertedAt time.Time, status sqlcv1.V1ReadableStatusOlap) replayStatusFixture {
	t.Helper()

	f := replayStatusFixture{
		tenantId:   tenantId,
		taskId:     taskId,
		insertedAt: pgtype.Timestamptz{Time: insertedAt, Valid: true},
		externalId: uuid.New(),
		workflowId: uuid.New(),
		workerId:   uuid.New(),
	}

	createReplayTask(t, ctx, repo, f)

	events := []sqlcv1.CreateTaskEventsOLAPParams{
		f.event(sqlcv1.V1EventTypeOlapQUEUED, sqlcv1.V1ReadableStatusOlapQUEUED, 0),
	}

	switch status {
	case sqlcv1.V1ReadableStatusOlapQUEUED:
	case sqlcv1.V1ReadableStatusOlapRUNNING:
		events = append(events,
			f.event(sqlcv1.V1EventTypeOlapASSIGNED, sqlcv1.V1ReadableStatusOlapRUNNING, 0),
			f.event(sqlcv1.V1EventTypeOlapSTARTED, sqlcv1.V1ReadableStatusOlapRUNNING, 0),
		)
	case sqlcv1.V1ReadableStatusOlapCOMPLETED:
		events = append(events,
			f.event(sqlcv1.V1EventTypeOlapASSIGNED, sqlcv1.V1ReadableStatusOlapRUNNING, 0),
			f.event(sqlcv1.V1EventTypeOlapSTARTED, sqlcv1.V1ReadableStatusOlapRUNNING, 0),
			f.event(sqlcv1.V1EventTypeOlapFINISHED, sqlcv1.V1ReadableStatusOlapCOMPLETED, 0),
		)
	default:
		t.Fatalf("unsupported seed status %s", status)
	}

	_, locksNotAcquired, err := repo.CreateTaskEvents(ctx, tenantId, events, map[uuid.UUID]uuid.UUID{f.externalId: f.externalId}, nil, nil)
	require.NoError(t, err)
	require.Empty(t, locksNotAcquired)

	assertOLAPRunStatus(t, ctx, pool, f, string(status))

	return f
}

type listedRuns struct {
	externalIds []uuid.UUID
	count       int
}

func (l listedRuns) externalIdSet() map[uuid.UUID]struct{} {
	set := make(map[uuid.UUID]struct{}, len(l.externalIds))

	for _, id := range l.externalIds {
		set[id] = struct{}{}
	}

	return set
}

// listRunsFn abstracts over ListWorkflowRuns and ListTasks, which share the same
// time-window shape and must behave identically for active runs outside the window.
type listRunsFn func(t *testing.T, statuses []sqlcv1.V1ReadableStatusOlap, includeOlderActive bool, limit, offset int64) listedRuns

// TestOLAPListRuns_ActiveRunsOlderThanWindow checks that QUEUED/RUNNING runs inserted
// before the requested window are still listed (and counted) when the caller opts in,
// while finished runs keep the time bound.
func TestOLAPListRuns_ActiveRunsOlderThanWindow(t *testing.T) {
	basePool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	pool := createEnumAwarePool(t, basePool)
	repo := createOLAPRepositoryWithRetention(t, pool, 30*24*time.Hour)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	require.NoError(t, repo.UpdateTablePartitions(ctx))

	now := time.Now().UTC().Truncate(time.Microsecond)
	olderInsertedAt := now.Add(-3 * 24 * time.Hour)
	since := now.Add(-24 * time.Hour)

	createOLAPPartitionsForDate(t, ctx, repo, pool, olderInsertedAt)

	tenantId := uuid.New()

	olderRunning := seedTaskWithStatus(t, ctx, repo, pool, tenantId, 1, olderInsertedAt, sqlcv1.V1ReadableStatusOlapRUNNING)
	olderQueued := seedTaskWithStatus(t, ctx, repo, pool, tenantId, 2, olderInsertedAt.Add(time.Second), sqlcv1.V1ReadableStatusOlapQUEUED)
	olderCompleted := seedTaskWithStatus(t, ctx, repo, pool, tenantId, 3, olderInsertedAt.Add(2*time.Second), sqlcv1.V1ReadableStatusOlapCOMPLETED)
	recentCompleted := seedTaskWithStatus(t, ctx, repo, pool, tenantId, 4, now.Add(-time.Hour), sqlcv1.V1ReadableStatusOlapCOMPLETED)
	recentRunning := seedTaskWithStatus(t, ctx, repo, pool, tenantId, 5, now.Add(-time.Minute), sqlcv1.V1ReadableStatusOlapRUNNING)

	listWorkflowRuns := func(t *testing.T, statuses []sqlcv1.V1ReadableStatusOlap, includeOlderActive bool, limit, offset int64) listedRuns {
		t.Helper()

		runs, count, err := repo.ListWorkflowRuns(ctx, tenantId, ListWorkflowRunOpts{
			CreatedAfter:           since,
			Statuses:               statuses,
			Limit:                  limit,
			Offset:                 offset,
			IncludeOlderActiveRuns: includeOlderActive,
		})
		require.NoError(t, err)

		out := listedRuns{count: count}
		for _, run := range runs {
			out.externalIds = append(out.externalIds, run.ExternalID)
		}

		return out
	}

	listTasks := func(t *testing.T, statuses []sqlcv1.V1ReadableStatusOlap, includeOlderActive bool, limit, offset int64) listedRuns {
		t.Helper()

		tasks, count, err := repo.ListTasks(ctx, tenantId, ListTaskRunOpts{
			CreatedAfter:           since,
			Statuses:               statuses,
			Limit:                  limit,
			Offset:                 offset,
			IncludeOlderActiveRuns: includeOlderActive,
		})
		require.NoError(t, err)

		out := listedRuns{count: count}
		for _, task := range tasks {
			out.externalIds = append(out.externalIds, task.ExternalID)
		}

		return out
	}

	for name, list := range map[string]listRunsFn{"workflow_runs": listWorkflowRuns, "tasks": listTasks} {
		t.Run(name, func(t *testing.T) {
			t.Run("active_run_older_than_since_is_returned", func(t *testing.T) {
				listed := list(t, nil, true, 50, 0)
				ids := listed.externalIdSet()

				assert.Contains(t, ids, olderRunning.externalId)
				assert.Contains(t, ids, olderQueued.externalId)
				assert.Contains(t, ids, recentRunning.externalId)
				assert.Contains(t, ids, recentCompleted.externalId)

				// Rows stay in inserted_at DESC order across both branches, so the
				// in-window rows come first and the older active rows last.
				assert.Equal(t, []uuid.UUID{
					recentRunning.externalId,
					recentCompleted.externalId,
					olderQueued.externalId,
					olderRunning.externalId,
				}, listed.externalIds)
			})

			t.Run("completed_run_older_than_since_is_not_returned", func(t *testing.T) {
				listed := list(t, nil, true, 50, 0)
				assert.NotContains(t, listed.externalIdSet(), olderCompleted.externalId)

				// Filtering to finished statuses only keeps the time bound for every row.
				onlyCompleted := list(t, []sqlcv1.V1ReadableStatusOlap{sqlcv1.V1ReadableStatusOlapCOMPLETED}, true, 50, 0)
				assert.Equal(t, []uuid.UUID{recentCompleted.externalId}, onlyCompleted.externalIds)
				assert.Equal(t, 1, onlyCompleted.count)
			})

			t.Run("count_matches_list", func(t *testing.T) {
				listed := list(t, nil, true, 50, 0)
				assert.Equal(t, len(listed.externalIds), listed.count)
				assert.Equal(t, 4, listed.count)

				// Paginating through the merged result must yield every row exactly
				// once and the count must be stable across pages.
				var paged []uuid.UUID
				for offset := int64(0); offset < int64(listed.count); offset += 3 {
					page := list(t, nil, true, 3, offset)
					assert.Equal(t, listed.count, page.count)
					paged = append(paged, page.externalIds...)
				}
				assert.Equal(t, listed.externalIds, paged)
			})

			t.Run("older_active_runs_are_opt_in", func(t *testing.T) {
				listed := list(t, nil, false, 50, 0)

				assert.Equal(t, []uuid.UUID{recentRunning.externalId, recentCompleted.externalId}, listed.externalIds)
				assert.Equal(t, 2, listed.count)
			})
		})
	}
}
