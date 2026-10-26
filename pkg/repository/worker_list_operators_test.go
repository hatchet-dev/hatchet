//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
	"sort"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// The workers list hides engine-managed workers unless operators are included: those of DAG
// operators and the serverless operator's, which carry its process label. Workers of any
// other operator, such as one a user runs over gRPC, are listed like SDK workers.
func TestListWorkersHidesEngineManagedOperators(t *testing.T) {
	pool := workerActionsPool(t)
	ctx := context.Background()
	repo, _ := createMeteredWorkerRepositoryForTest(t, pool)
	tenantId := createLimitTestTenant(t, pool)

	sdkWorker := seedActiveWorkerForLimits(t, ctx, pool, tenantId, nil, false)

	dagOp := seedOperatorForLimits(t, ctx, pool, tenantId, sqlcv1.V1OperatorKindDAG)
	dagWorker := seedActiveWorkerForLimits(t, ctx, pool, tenantId, &dagOp, true)

	userOp := seedOperatorForLimits(t, ctx, pool, tenantId, sqlcv1.V1OperatorKindGRPC)
	userWorker := seedActiveWorkerForLimits(t, ctx, pool, tenantId, &userOp, false)

	serverlessOp := seedOperatorForLimits(t, ctx, pool, tenantId, sqlcv1.V1OperatorKindGRPC)
	serverlessWorker := seedActiveWorkerForLimits(t, ctx, pool, tenantId, &serverlessOp, true)

	_, err := pool.Exec(ctx,
		`INSERT INTO "WorkerLabel" ("workerId", "key", "strValue") VALUES ($1, 'hatchet-serverless-process', $2)`,
		serverlessWorker, uuid.NewString(),
	)
	require.NoError(t, err)

	list := func(includeOperators *bool) ([]uuid.UUID, int64) {
		rows, count, err := repo.ListWorkers(ctx, tenantId, &ListWorkersOpts{IncludeOperators: includeOperators})
		require.NoError(t, err)

		ids := make([]uuid.UUID, 0, len(rows))

		for _, row := range rows {
			ids = append(ids, row.Worker.ID)
		}

		sort.Slice(ids, func(i, j int) bool { return ids[i].String() < ids[j].String() })

		return ids, count
	}

	sorted := func(ids ...uuid.UUID) []uuid.UUID {
		sort.Slice(ids, func(i, j int) bool { return ids[i].String() < ids[j].String() })
		return ids
	}

	ids, count := list(nil)
	assert.Equal(t, sorted(sdkWorker, userWorker), ids, "DAG and serverless operator workers are hidden by default")
	assert.Equal(t, int64(2), count)

	include := true
	ids, count = list(&include)
	assert.Equal(t, sorted(sdkWorker, dagWorker, userWorker, serverlessWorker), ids)
	assert.Equal(t, int64(4), count)
}
