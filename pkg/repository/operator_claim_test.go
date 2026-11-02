//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
	"io"
	"testing"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/internal/enumerables"
	"github.com/hatchet-dev/hatchet/pkg/repository/fairpool"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/validator"
)

// A dispatcher that marks itself inactive at the start of its shutdown claims nothing more,
// and its operators become claimable by a live dispatcher at once, before its row is deleted
// at the end of the shutdown.
func TestClaimOperators_InactiveDispatcherHandsOverAndClaimsNothing(t *testing.T) {
	pool := workerActionsPool(t)
	logger := zerolog.New(io.Discard)
	shared := &sharedRepository{
		pool:    fairpool.Ungated(pool),
		l:       &logger,
		v:       validator.NewDefaultValidator(),
		queries: sqlcv1.New(),
	}
	operators := newOperatorRepository(shared)
	dispatchers := newDispatcherRepository(shared)
	workers := newWorkerRepository(shared)

	ctx := context.Background()
	tenantId := seedTenantForActions(t, ctx, pool)
	leavingDispatcherId := seedDispatcher(t, ctx, pool)
	liveDispatcherId := seedDispatcher(t, ctx, pool)

	operator, err := operators.CreateOperator(ctx, tenantId, CreateOperatorOpts{
		Name:           "default",
		Kind:           sqlcv1.V1OperatorKindDAG,
		LeasingManager: sqlcv1.V1OperatorLeasingManagerDISPATCHER,
		Config:         []byte("{}"),
	})
	require.NoError(t, err)

	claimedIds := func(dispatcherId uuid.UUID) []uuid.UUID {
		claimed, err := operators.ClaimOperators(ctx, dispatcherId)
		require.NoError(t, err)

		return enumerables.Map(claimed, func(op *sqlcv1.V1Operator) uuid.UUID { return op.ID })
	}

	require.Contains(t, claimedIds(leavingDispatcherId), operator.ID, "an unassigned operator is claimable")

	worker, err := workers.CreateNewWorker(ctx, tenantId, &CreateWorkerOpts{
		DispatcherId:       leavingDispatcherId,
		Name:               "operator-worker",
		Actions:            []string{"svc:run"},
		OperatorId:         &operator.ID,
		IsExemptFromLimits: true,
	})
	require.NoError(t, err)

	_, err = operators.UpdateOperator(ctx, tenantId, operator.ID, UpdateOperatorOpts{WorkerId: &worker.ID})
	require.NoError(t, err)

	require.Contains(t, claimedIds(leavingDispatcherId), operator.ID, "the hosting dispatcher keeps its claim")
	require.NotContains(t, claimedIds(liveDispatcherId), operator.ID, "an operator on a live dispatcher is not claimable by another")

	require.NoError(t, dispatchers.MarkInactive(ctx, leavingDispatcherId))

	require.Empty(t, claimedIds(leavingDispatcherId), "an inactive dispatcher claims nothing")
	require.Contains(t, claimedIds(liveDispatcherId), operator.ID, "an operator on an inactive dispatcher is claimable before the dispatcher row is deleted")
}
