//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// seedPlainDag mirrors seedOperatorDag but for a regular (non-operator) DAG,
// whose readable status is derived purely from counting its child tasks.
func seedPlainDag(t *testing.T, ctx context.Context, repo *OLAPRepositoryImpl, dagId int64, totalTasks int) operatorDagFixture {
	t.Helper()

	f := operatorDagFixture{
		tenantId:      uuid.New(),
		dagId:         dagId,
		dagInsertedAt: pgtype.Timestamptz{Time: time.Now().UTC().Truncate(time.Microsecond), Valid: true},
		dagExternalId: uuid.New(),
		workflowId:    uuid.New(),
	}

	dag := &DAGWithData{
		V1Dag: &sqlcv1.V1Dag{
			ID:                f.dagId,
			InsertedAt:        f.dagInsertedAt,
			TenantID:          f.tenantId,
			ExternalID:        f.dagExternalId,
			DisplayName:       "dag-rollup-race-test",
			WorkflowID:        f.workflowId,
			WorkflowVersionID: uuid.New(),
		},
		Input:              []byte(`{}`),
		AdditionalMetadata: []byte(`{}`),
		TotalTasks:         totalTasks,
	}

	locksNotAcquired, err := repo.CreateDAGs(ctx, f.tenantId, []*DAGWithData{dag})
	require.NoError(t, err)
	require.Empty(t, locksNotAcquired)

	return f
}

// TestOLAPDAGRollup_ConcurrentRollupsDoNotStrandDAG is a regression test for
// the write-skew that the run-level advisory locks used to paper over.
//
// The DAG rollup recomputes a DAG's status from a join over its task rows.
// Under READ COMMITTED, a single-statement rollup that blocks on the DAG row
// lock resumes with the snapshot it started with: it re-reads the DAG row via
// EvalPlanQual but still sees the *task* rows as of statement start. Two
// transactions each completing one of a DAG's final two tasks could then both
// count the other's task as RUNNING and both leave the DAG RUNNING — with no
// later event ever arriving to fix it.
//
// The fix takes the DAG row locks in a separate LockDAGsForStatusUpdate
// statement first, so the rollup statement's snapshot is opened only after
// any concurrent rollup for the same DAG has committed.
func TestOLAPDAGRollup_ConcurrentRollupsDoNotStrandDAG(t *testing.T) {
	basePool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	pool := createEnumAwarePool(t, basePool)
	repo := createOLAPRepositoryWithPayloadStore(t, pool)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	require.NoError(t, repo.UpdateTablePartitions(ctx))

	f := seedPlainDag(t, ctx, repo, 100, 2)
	childA := f.createChild(t, ctx, repo, 101)
	childB := f.createChild(t, ctx, repo, 102)

	// Move both children to RUNNING through the normal ingestion path so the
	// DAG itself is RUNNING before the race.
	f.applyChildEvents(t, ctx, repo, []sqlcv1.CreateTaskEventsOLAPParams{
		childA.event(sqlcv1.V1EventTypeOlapSTARTED, sqlcv1.V1ReadableStatusOlapRUNNING, 0),
		childB.event(sqlcv1.V1EventTypeOlapSTARTED, sqlcv1.V1ReadableStatusOlapRUNNING, 0),
	})
	f.assertDagStatus(t, ctx, pool, "RUNNING")

	completeTask := func(tx pgx.Tx, c replayStatusFixture) error {
		_, err := repo.queries.UpdateTaskStatusesFromMQ(ctx, tx, sqlcv1.UpdateTaskStatusesFromMQParams{
			Tenantids:       []uuid.UUID{c.tenantId},
			Taskids:         []int64{c.taskId},
			Taskinsertedats: []pgtype.Timestamptz{c.insertedAt},
			Statuses:        []sqlcv1.V1ReadableStatusOlap{sqlcv1.V1ReadableStatusOlapCOMPLETED},
			Workerids:       []uuid.UUID{uuid.Nil},
			Retrycounts:     []int32{0},
		})
		return err
	}

	rollup := func(tx pgx.Tx) error {
		if err := repo.queries.LockDAGsForStatusUpdate(ctx, tx, sqlcv1.LockDAGsForStatusUpdateParams{
			Tenantids:      []uuid.UUID{f.tenantId},
			Dagids:         []int64{f.dagId},
			Daginsertedats: []pgtype.Timestamptz{f.dagInsertedAt},
		}); err != nil {
			return err
		}

		_, err := repo.queries.UpdateDAGStatusesFromMQ(ctx, tx, sqlcv1.UpdateDAGStatusesFromMQParams{
			Tenantids:      []uuid.UUID{f.tenantId},
			Dagids:         []int64{f.dagId},
			Daginsertedats: []pgtype.Timestamptz{f.dagInsertedAt},
		})
		return err
	}

	tx1, err := pool.Begin(ctx)
	require.NoError(t, err)
	defer func() { _ = tx1.Rollback(ctx) }()

	tx2, err := pool.Begin(ctx)
	require.NoError(t, err)
	defer func() { _ = tx2.Rollback(ctx) }()

	// Each transaction completes one of the DAG's two remaining tasks.
	require.NoError(t, completeTask(tx1, childA))
	require.NoError(t, completeTask(tx2, childB))

	// tx1's rollup runs while tx2's completion of child B is uncommitted, so
	// it (correctly, at that instant) sees B as RUNNING, leaves the DAG
	// RUNNING, and holds the DAG row lock until commit.
	require.NoError(t, rollup(tx1))

	// tx2's rollup must block on the DAG row lock until tx1 commits and then
	// count child A's committed completion; with a single-statement rollup it
	// would resume with its stale snapshot and strand the DAG in RUNNING.
	tx2Done := make(chan error, 1)
	go func() {
		if err := rollup(tx2); err != nil {
			tx2Done <- err
			return
		}
		tx2Done <- tx2.Commit(ctx)
	}()

	// Let tx2 reach the DAG row lock wait before tx1 commits, so the blocked
	// path is what's exercised rather than a sequential fast path.
	time.Sleep(300 * time.Millisecond)

	require.NoError(t, tx1.Commit(ctx))
	require.NoError(t, <-tx2Done)

	f.assertDagStatus(t, ctx, pool, "COMPLETED")
}
