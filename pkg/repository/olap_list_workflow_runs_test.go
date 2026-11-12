//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

type finishedDAGFixture struct {
	tenantId            uuid.UUID
	dagId               int64
	insertedAt          pgtype.Timestamptz
	lastFinishedEventId uuid.UUID
	dagExternalId       uuid.UUID
	childExternalIds    []uuid.UUID
}

func setupOLAPRepository(t *testing.T) (context.Context, *pgxpool.Pool, *OLAPRepositoryImpl) {
	t.Helper()

	basePool, cleanup := setupPostgresWithMigration(t)
	t.Cleanup(cleanup)

	pool := createEnumAwarePool(t, basePool)
	repo := createOLAPRepositoryWithPayloadStore(t, pool)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	t.Cleanup(cancel)

	require.NoError(t, repo.UpdateTablePartitions(ctx))

	return ctx, pool, repo
}

// seedDAGWithFinishedChildren writes one DAG run whose children each finish in their own write, so
// the FINISHED events get distinct inserted_at values.
func seedDAGWithFinishedChildren(t *testing.T, ctx context.Context, repo *OLAPRepositoryImpl, numChildren int) finishedDAGFixture {
	t.Helper()

	f := finishedDAGFixture{
		tenantId:   uuid.New(),
		dagId:      1,
		insertedAt: pgtype.Timestamptz{Time: time.Now().UTC().Truncate(time.Microsecond), Valid: true},
	}

	workflowId := uuid.New()
	workflowVersionId := uuid.New()
	dagExternalId := uuid.New()
	f.dagExternalId = dagExternalId

	locksNotAcquired, err := repo.CreateDAGs(ctx, f.tenantId, []*DAGWithData{{
		V1Dag: &sqlcv1.V1Dag{
			ID:                f.dagId,
			InsertedAt:        f.insertedAt,
			TenantID:          f.tenantId,
			ExternalID:        dagExternalId,
			DisplayName:       "finished-children-dag",
			WorkflowID:        workflowId,
			WorkflowVersionID: workflowVersionId,
		},
		Input:              []byte(`{}`),
		AdditionalMetadata: []byte(`{}`),
	}})
	require.NoError(t, err)
	require.Empty(t, locksNotAcquired)

	children := make([]*V1TaskWithPayload, 0, numChildren)

	for i := range numChildren {
		children = append(children, &V1TaskWithPayload{
			V1Task: &sqlcv1.V1Task{
				ID:                 f.dagId + int64(i) + 1,
				InsertedAt:         f.insertedAt,
				TenantID:           f.tenantId,
				Queue:              "default",
				ActionID:           "test:finished-children-dag-child",
				StepID:             uuid.New(),
				WorkflowID:         workflowId,
				WorkflowVersionID:  workflowVersionId,
				WorkflowRunID:      dagExternalId,
				ScheduleTimeout:    "5m",
				Sticky:             sqlcv1.V1StickyStrategyNONE,
				ExternalID:         uuid.New(),
				DisplayName:        "finished-children-dag-child",
				Input:              []byte(`{}`),
				AdditionalMetadata: []byte(`{}`),
				DagID:              pgtype.Int8{Int64: f.dagId, Valid: true},
				DagInsertedAt:      f.insertedAt,
			},
			Payload: []byte(`{}`),
		})
	}

	for _, child := range children {
		f.childExternalIds = append(f.childExternalIds, child.ExternalID)
	}

	_, locksNotAcquired, err = repo.CreateTasks(ctx, f.tenantId, children)
	require.NoError(t, err)
	require.Empty(t, locksNotAcquired)

	for _, child := range children {
		event := sqlcv1.CreateTaskEventsOLAPParams{
			TenantID:       f.tenantId,
			TaskID:         child.ID,
			TaskInsertedAt: f.insertedAt,
			EventType:      sqlcv1.V1EventTypeOlapFINISHED,
			WorkflowID:     workflowId,
			EventTimestamp: pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
			ReadableStatus: sqlcv1.V1ReadableStatusOlapCOMPLETED,
			Output:         []byte(`{}`),
			ExternalID:     uuid.New(),
		}
		f.lastFinishedEventId = event.ExternalID

		_, locksNotAcquired, err = repo.CreateTaskEvents(ctx, f.tenantId, []sqlcv1.CreateTaskEventsOLAPParams{event}, map[uuid.UUID]uuid.UUID{event.ExternalID: dagExternalId}, nil, nil, nil)
		require.NoError(t, err)
		require.Empty(t, locksNotAcquired)
	}

	return f
}

func TestPopulateDAGMetadata_OneRowPerDAGWithLatestChildOutput(t *testing.T) {
	ctx, pool, repo := setupOLAPRepository(t)
	f := seedDAGWithFinishedChildren(t, ctx, repo, 2)

	rows, err := repo.queries.PopulateDAGMetadata(ctx, pool, sqlcv1.PopulateDAGMetadataParams{
		Ids:             []int64{f.dagId},
		Insertedats:     []pgtype.Timestamptz{f.insertedAt},
		Tenantid:        f.tenantId,
		Includepayloads: true,
	})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.NotNil(t, rows[0].OutputEventExternalID)
	require.Equal(t, f.lastFinishedEventId, *rows[0].OutputEventExternalID)
}

func TestListWorkflowRuns_PayloadCapCountsReturnedRowsNotLimit(t *testing.T) {
	ctx, _, repo := setupOLAPRepository(t)
	f := seedDAGWithFinishedChildren(t, ctx, repo, 2)

	runs, _, err := repo.ListWorkflowRuns(ctx, f.tenantId, ListWorkflowRunOpts{
		CreatedAfter:    f.insertedAt.Time.Add(-time.Hour),
		Limit:           MaxTasksWithPayloads * 5,
		IncludePayloads: true,
	})
	require.NoError(t, err)
	require.Len(t, runs, 1)
}

func TestListFailedWorkflowRunsByFinishedAt_ReturnsRunFailedAfterInsert(t *testing.T) {
	ctx, pool, repo := setupOLAPRepository(t)
	f := seedDAGWithFinishedChildren(t, ctx, repo, 1)

	// finished_at is written from the database clock, which can lag the host clock by a millisecond or so under Docker
	var beforeFailure time.Time
	require.NoError(t, pool.QueryRow(ctx, `SELECT clock_timestamp()`).Scan(&beforeFailure))
	_, err := pool.Exec(ctx, `UPDATE v1_dags_olap SET readable_status = 'FAILED' WHERE tenant_id = $1 AND id = $2 AND inserted_at = $3`, f.tenantId, f.dagId, f.insertedAt)
	require.NoError(t, err)

	var finishedAt pgtype.Timestamptz
	require.NoError(t, pool.QueryRow(ctx, `SELECT finished_at FROM v1_runs_olap WHERE tenant_id = $1 AND id = $2`, f.tenantId, f.dagId).Scan(&finishedAt))
	require.True(t, finishedAt.Valid)

	runs, err := repo.ListFailedWorkflowRunsByFinishedAt(ctx, f.tenantId, beforeFailure, time.Now().Add(time.Minute), 10)
	require.NoError(t, err)
	require.Len(t, runs, 1)

	_, err = pool.Exec(ctx, `UPDATE v1_dags_olap SET latest_retry_count = latest_retry_count + 1 WHERE tenant_id = $1 AND id = $2 AND inserted_at = $3`, f.tenantId, f.dagId, f.insertedAt)
	require.NoError(t, err)

	var refailedAt pgtype.Timestamptz
	require.NoError(t, pool.QueryRow(ctx, `SELECT finished_at FROM v1_runs_olap WHERE tenant_id = $1 AND id = $2`, f.tenantId, f.dagId).Scan(&refailedAt))
	require.True(t, refailedAt.Time.After(finishedAt.Time))
}

func TestCheckTasksWithPayloads(t *testing.T) {
	require.NoError(t, checkTasksWithPayloads(true, MaxTasksWithPayloads))
	require.ErrorIs(t, checkTasksWithPayloads(true, MaxTasksWithPayloads+1), ErrTooManyTasksWithPayloads)
	require.NoError(t, checkTasksWithPayloads(false, MaxTasksWithPayloads+1))
}
