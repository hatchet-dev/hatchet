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

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func insertDurableLogFile(t *testing.T, ctx context.Context, repo *TaskRepositoryImpl, tenantId uuid.UUID, taskId int64, taskInsertedAt time.Time, latestSatisfiedOrder int64) {
	t.Helper()

	_, err := repo.pool.Exec(
		ctx,
		`INSERT INTO v1_durable_event_log_file (tenant_id, durable_task_id, durable_task_inserted_at, latest_invocation_count, latest_inserted_at, latest_node_id, latest_branch_id, latest_satisfied_order)
		 VALUES ($1, $2, $3, 1, NOW(), 0, 1, $4)`,
		tenantId,
		taskId,
		taskInsertedAt,
		latestSatisfiedOrder,
	)
	require.NoError(t, err)
}

func insertDurableLogEntry(t *testing.T, ctx context.Context, repo *TaskRepositoryImpl, tenantId uuid.UUID, taskId int64, taskInsertedAt time.Time, branchId, nodeId int64) {
	t.Helper()

	_, err := repo.pool.Exec(
		ctx,
		`INSERT INTO v1_durable_event_log_entry (tenant_id, external_id, durable_task_id, durable_task_inserted_at, inserted_at, kind, node_id, branch_id, idempotency_key)
		 VALUES ($1, $2, $3, $4, NOW(), 'RUN', $5, $6, $7)`,
		tenantId,
		uuid.New(),
		taskId,
		taskInsertedAt,
		nodeId,
		branchId,
		[]byte("test-idempotency-key"),
	)
	require.NoError(t, err)
}

func TestUpdateDurableEventLogEntriesSatisfied_MultiParentBatch(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()
	repo := createTaskRepository(pool)

	require.NoError(t, repo.UpdateTablePartitions(ctx))

	tenantId := uuid.New()
	now := time.Now().UTC().Truncate(time.Millisecond)

	parentA := int64(11)
	parentAInsertedAt := now
	parentB := int64(22)
	parentBInsertedAt := now.Add(-1 * time.Minute)

	// parent A starts with a non-zero satisfied order; parent B starts at zero
	insertDurableLogFile(t, ctx, repo, tenantId, parentA, parentAInsertedAt, 5)
	insertDurableLogFile(t, ctx, repo, tenantId, parentB, parentBInsertedAt, 0)

	insertDurableLogEntry(t, ctx, repo, tenantId, parentA, parentAInsertedAt, 1, 0)
	insertDurableLogEntry(t, ctx, repo, tenantId, parentA, parentAInsertedAt, 1, 1)
	insertDurableLogEntry(t, ctx, repo, tenantId, parentA, parentAInsertedAt, 1, 2)
	insertDurableLogEntry(t, ctx, repo, tenantId, parentB, parentBInsertedAt, 1, 0)

	toTimestamptz := func(ts time.Time) pgtype.Timestamptz {
		return sqlchelpers.TimestamptzFromTime(ts)
	}

	// satisfy two entries of parent A (out of order) and one of parent B in a single batch
	rows, err := repo.queries.UpdateDurableEventLogEntriesSatisfied(ctx, pool, sqlcv1.UpdateDurableEventLogEntriesSatisfiedParams{
		Durabletaskids:         []int64{parentA, parentB, parentA},
		Durabletaskinsertedats: []pgtype.Timestamptz{toTimestamptz(parentAInsertedAt), toTimestamptz(parentBInsertedAt), toTimestamptz(parentAInsertedAt)},
		Nodeids:                []int64{2, 0, 0},
		Branchids:              []int64{1, 1, 1},
		Childtaskisfailures:    []bool{false, false, true},
		Childtaskerrormessages: []string{"", "", "boom"},
	})
	require.NoError(t, err)
	require.Len(t, rows, 3)

	satisfiedOrders := make(map[int64]map[int64]int64) // parent -> node -> satisfied_order

	for _, row := range rows {
		assert.True(t, row.IsSatisfied)
		require.True(t, row.SatisfiedOrder.Valid, "satisfied_order should be set for parent %d node %d", row.DurableTaskID, row.NodeID)

		if satisfiedOrders[row.DurableTaskID] == nil {
			satisfiedOrders[row.DurableTaskID] = make(map[int64]int64)
		}

		satisfiedOrders[row.DurableTaskID][row.NodeID] = row.SatisfiedOrder.Int64

		if row.DurableTaskID == parentA && row.NodeID == 0 {
			assert.True(t, row.ChildTaskIsFailure)
			assert.Equal(t, "boom", row.ChildTaskErrorMessage.String)
		}
	}

	// parent A orders continue from its latest_satisfied_order (5), assigned by (branch_id, node_id)
	assert.Equal(t, int64(6), satisfiedOrders[parentA][0])
	assert.Equal(t, int64(7), satisfiedOrders[parentA][2])
	// parent B starts from 0
	assert.Equal(t, int64(1), satisfiedOrders[parentB][0])

	// log files advance to the max satisfied order per parent
	var latestA, latestB int64
	require.NoError(t, pool.QueryRow(ctx, `SELECT latest_satisfied_order FROM v1_durable_event_log_file WHERE durable_task_id = $1`, parentA).Scan(&latestA))
	require.NoError(t, pool.QueryRow(ctx, `SELECT latest_satisfied_order FROM v1_durable_event_log_file WHERE durable_task_id = $1`, parentB).Scan(&latestB))
	assert.Equal(t, int64(7), latestA)
	assert.Equal(t, int64(1), latestB)

	// re-satisfying an entry keeps its original satisfied_order
	rows, err = repo.queries.UpdateDurableEventLogEntriesSatisfied(ctx, pool, sqlcv1.UpdateDurableEventLogEntriesSatisfiedParams{
		Durabletaskids:         []int64{parentA},
		Durabletaskinsertedats: []pgtype.Timestamptz{toTimestamptz(parentAInsertedAt)},
		Nodeids:                []int64{0},
		Branchids:              []int64{1},
		Childtaskisfailures:    []bool{false},
		Childtaskerrormessages: []string{""},
	})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, int64(6), rows[0].SatisfiedOrder.Int64)

	// the untouched entry of parent A is still unsatisfied
	var unsatisfiedCount int
	require.NoError(t, pool.QueryRow(ctx, `SELECT COUNT(*) FROM v1_durable_event_log_entry WHERE durable_task_id = $1 AND NOT is_satisfied`, parentA).Scan(&unsatisfiedCount))
	assert.Equal(t, 1, unsatisfiedCount)
}
