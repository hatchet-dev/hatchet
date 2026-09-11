//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
	"io"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/validator"
)

// createMeteredWorkerRepositoryForTest builds a worker repository whose creates are metered
// against the tenant limits the returned limit repository enforces.
func createMeteredWorkerRepositoryForTest(t *testing.T, pool *pgxpool.Pool) (WorkerRepository, *tenantLimitRepository) {
	t.Helper()

	logger := zerolog.New(io.Discard)
	limits := createTenantLimitRepositoryForTest(t, pool, defaultLimitTestConfig())

	return newWorkerRepository(&sharedRepository{
		pool:    pool,
		l:       &logger,
		v:       validator.NewDefaultValidator(),
		queries: sqlcv1.New(),
		m:       limits,
	}), limits
}

func seedDispatcherForLimits(t *testing.T, ctx context.Context, pool *pgxpool.Pool) uuid.UUID {
	t.Helper()

	dispatcherId := uuid.New()

	_, err := pool.Exec(ctx, `INSERT INTO "Dispatcher" ("id", "lastHeartbeatAt") VALUES ($1, now())`, dispatcherId)
	require.NoError(t, err)

	return dispatcherId
}

func seedOperatorForLimits(t *testing.T, ctx context.Context, pool *pgxpool.Pool, tenantId uuid.UUID, kind sqlcv1.V1OperatorKind) uuid.UUID {
	t.Helper()

	operatorId := uuid.New()

	_, err := pool.Exec(ctx,
		`INSERT INTO v1_operator (id, tenant_id, name, kind, config) VALUES ($1, $2, $3, $4, '{}'::jsonb)`,
		operatorId, tenantId, "op-"+operatorId.String()[:8], string(kind),
	)
	require.NoError(t, err)

	return operatorId
}

// seedActiveWorkerForLimits inserts a worker the limit queries count: active, heartbeating,
// and backing operatorId when one is given.
func seedActiveWorkerForLimits(t *testing.T, ctx context.Context, pool *pgxpool.Pool, tenantId uuid.UUID, operatorId *uuid.UUID) uuid.UUID {
	t.Helper()

	workerId := uuid.New()

	_, err := pool.Exec(ctx,
		`INSERT INTO "Worker" ("id", "tenantId", "name", "actionHash", "isActive", "lastHeartbeatAt", "operatorId") VALUES ($1, $2, $3, $4, true, now(), $5)`,
		workerId, tenantId, "worker-"+workerId.String()[:8], hashActions(nil), operatorId,
	)
	require.NoError(t, err)

	return workerId
}

// Operator workers count towards the tenant's worker and slot limits like SDK workers, with one
// exception: the DAG operator's workers are engine infrastructure and are neither counted nor
// refused.
func TestCreateNewWorkerMetersOperatorWorkers(t *testing.T) {
	pool := workerActionsPool(t)
	ctx := context.Background()
	repo, limits := createMeteredWorkerRepositoryForTest(t, pool)
	dispatcherId := seedDispatcherForLimits(t, ctx, pool)

	sdkWorker := func(name string) *CreateWorkerOpts {
		return &CreateWorkerOpts{DispatcherId: dispatcherId, Name: name, SlotConfig: map[string]int32{SlotTypeDefault: 1}}
	}

	operatorWorker := func(name string, operatorId uuid.UUID, kind sqlcv1.V1OperatorKind, slots int32) *CreateWorkerOpts {
		return &CreateWorkerOpts{
			DispatcherId: dispatcherId,
			Name:         name,
			SlotConfig:   map[string]int32{SlotTypeDefault: slots},
			OperatorId:   &operatorId,
			OperatorKind: kind,
		}
	}

	t.Run("a gRPC operator worker over the worker limit is refused like an SDK worker", func(t *testing.T) {
		tenantId := createLimitTestTenant(t, pool)
		require.NoError(t, limits.UpdateLimits(ctx, tenantId, []Limit{{Resource: sqlcv1.LimitResourceWORKER, Limit: 1}}))

		grpcOp := seedOperatorForLimits(t, ctx, pool, tenantId, sqlcv1.V1OperatorKindGRPC)
		dagOp := seedOperatorForLimits(t, ctx, pool, tenantId, sqlcv1.V1OperatorKindDAG)

		// the one worker the limit allows is an operator's, and it counts
		seedActiveWorkerForLimits(t, ctx, pool, tenantId, &grpcOp)

		count, err := limits.queries.CountTenantWorkers(ctx, pool, tenantId)
		require.NoError(t, err)
		assert.EqualValues(t, 1, count, "the gRPC operator's worker is counted")

		_, err = repo.CreateNewWorker(ctx, tenantId, operatorWorker("grpc-op", grpcOp, sqlcv1.V1OperatorKindGRPC, 1))
		assert.ErrorIs(t, err, ErrResourceExhausted, "an operator worker over the limit is refused")

		_, err = repo.CreateNewWorker(ctx, tenantId, sdkWorker("sdk"))
		assert.ErrorIs(t, err, ErrResourceExhausted, "with the same error an SDK worker gets")

		_, err = repo.CreateNewWorker(ctx, tenantId, operatorWorker("dag-op", dagOp, sqlcv1.V1OperatorKindDAG, 10000))
		assert.NoError(t, err, "the DAG operator's worker is infrastructure and is not metered")
	})

	t.Run("the DAG operator's workers do not count", func(t *testing.T) {
		tenantId := createLimitTestTenant(t, pool)
		require.NoError(t, limits.UpdateLimits(ctx, tenantId, []Limit{{Resource: sqlcv1.LimitResourceWORKER, Limit: 1}}))

		dagOp := seedOperatorForLimits(t, ctx, pool, tenantId, sqlcv1.V1OperatorKindDAG)
		seedActiveWorkerForLimits(t, ctx, pool, tenantId, &dagOp)

		count, err := limits.queries.CountTenantWorkers(ctx, pool, tenantId)
		require.NoError(t, err)
		assert.Zero(t, count, "the DAG operator's worker is left out of the count")

		_, err = repo.CreateNewWorker(ctx, tenantId, sdkWorker("sdk"))
		assert.NoError(t, err, "the limit is still free for the tenant's own worker")
	})

	t.Run("a gRPC operator worker over the slot limit is refused", func(t *testing.T) {
		tenantId := createLimitTestTenant(t, pool)
		require.NoError(t, limits.UpdateLimits(ctx, tenantId, []Limit{
			{Resource: sqlcv1.LimitResourceWORKER, Limit: 10},
			{Resource: sqlcv1.LimitResourceWORKERSLOT, Limit: 5},
		}))

		grpcOp := seedOperatorForLimits(t, ctx, pool, tenantId, sqlcv1.V1OperatorKindGRPC)

		_, err := repo.CreateNewWorker(ctx, tenantId, operatorWorker("grpc-op", grpcOp, sqlcv1.V1OperatorKindGRPC, 10))
		assert.ErrorIs(t, err, ErrResourceExhausted, "the operator worker's slots are metered")
	})

	t.Run("an operator worker names its operator's kind", func(t *testing.T) {
		tenantId := createLimitTestTenant(t, pool)
		grpcOp := seedOperatorForLimits(t, ctx, pool, tenantId, sqlcv1.V1OperatorKindGRPC)

		opts := operatorWorker("grpc-op", grpcOp, sqlcv1.V1OperatorKindGRPC, 1)
		opts.OperatorKind = ""

		_, err := repo.CreateNewWorker(ctx, tenantId, opts)
		assert.Error(t, err, "OperatorId without OperatorKind is refused before anything is metered")
	})
}
