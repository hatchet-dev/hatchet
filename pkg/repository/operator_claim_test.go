//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"bytes"
	"context"
	"io"
	"slices"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/internal/enumerables"
	"github.com/hatchet-dev/hatchet/pkg/repository/fairpool"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/validator"
)

// buildsDAG is what a dispatcher with only the DAG factory claims with.
var buildsDAG = ClaimOperatorsOpts{Kinds: []sqlcv1.V1OperatorKind{sqlcv1.V1OperatorKindDAG}}

// claimFixture plays both sides of the claim: the dispatchers polling it, and the claimer
// registering a worker for each claimed row and pointing the row at it.
type claimFixture struct {
	t           *testing.T
	ctx         context.Context
	pool        *pgxpool.Pool
	operators   OperatorRepository
	dispatchers DispatcherRepository
	workers     WorkerRepository
}

func newClaimFixture(t *testing.T, pool *pgxpool.Pool) *claimFixture {
	logger := zerolog.New(io.Discard)
	shared := &sharedRepository{
		pool:    fairpool.Ungated(pool),
		l:       &logger,
		v:       validator.NewDefaultValidator(),
		queries: sqlcv1.New(),
	}

	return &claimFixture{
		t:           t,
		ctx:         context.Background(),
		pool:        pool,
		operators:   newOperatorRepository(shared),
		dispatchers: newDispatcherRepository(shared),
		workers:     newWorkerRepository(shared),
	}
}

// isolatedClaimPool is a database of its own: the fair share counts every live dispatcher and
// every DISPATCHER row in the database, so these tests cannot share one.
func isolatedClaimPool(t *testing.T) *pgxpool.Pool {
	pool, cleanup := setupPostgresWithMigration(t)
	t.Cleanup(cleanup)

	return pool
}

// dagOperator creates a tenant and its DAG operator row, the way workflow creation does.
func (f *claimFixture) dagOperator() *sqlcv1.V1Operator {
	f.t.Helper()

	op, err := f.operators.CreateOperator(f.ctx, seedTenantForActions(f.t, f.ctx, f.pool), CreateOperatorOpts{
		Name:           "default",
		Kind:           sqlcv1.V1OperatorKindDAG,
		LeasingManager: sqlcv1.V1OperatorLeasingManagerDISPATCHER,
		Config:         []byte("{}"),
	})
	require.NoError(f.t, err)

	return op
}

func (f *claimFixture) dagOperators(n int) []uuid.UUID {
	ids := make([]uuid.UUID, n)

	for i := range ids {
		ids[i] = f.dagOperator().ID
	}

	return ids
}

func (f *claimFixture) claim(dispatcherId uuid.UUID, opts ClaimOperatorsOpts) []uuid.UUID {
	f.t.Helper()

	claimed, err := f.operators.ClaimOperators(f.ctx, dispatcherId, opts)
	require.NoError(f.t, err)

	return enumerables.Map(claimed, func(op *sqlcv1.V1Operator) uuid.UUID { return op.ID })
}

// host registers a worker on the dispatcher for each operator and points the operator at it,
// as the claimer does for the rows it claimed.
func (f *claimFixture) host(dispatcherId uuid.UUID, operatorIds ...uuid.UUID) {
	f.t.Helper()

	for _, operatorId := range operatorIds {
		op, err := f.operators.GetOperatorById(f.ctx, operatorId)
		require.NoError(f.t, err)

		worker, err := f.workers.CreateNewWorker(f.ctx, op.TenantID, &CreateWorkerOpts{
			DispatcherId:       dispatcherId,
			Name:               "operator-worker",
			Actions:            []string{"svc:run"},
			OperatorId:         &op.ID,
			IsExemptFromLimits: true,
		})
		require.NoError(f.t, err)

		_, err = f.operators.UpdateOperator(f.ctx, op.TenantID, op.ID, UpdateOperatorOpts{WorkerId: &worker.ID})
		require.NoError(f.t, err)
	}
}

// A dispatcher that marks itself inactive at the start of its shutdown claims nothing more,
// and its operators become claimable by a live dispatcher at once, before its row is deleted
// at the end of the shutdown.
func TestClaimOperators_InactiveDispatcherHandsOverAndClaimsNothing(t *testing.T) {
	pool := isolatedClaimPool(t)
	f := newClaimFixture(t, pool)

	leavingDispatcherId := seedDispatcher(t, f.ctx, pool)
	liveDispatcherId := seedDispatcher(t, f.ctx, pool)

	operator := f.dagOperator()

	require.Contains(t, f.claim(leavingDispatcherId, buildsDAG), operator.ID, "an unassigned operator is claimable")

	f.host(leavingDispatcherId, operator.ID)

	require.Contains(t, f.claim(leavingDispatcherId, buildsDAG), operator.ID, "the hosting dispatcher keeps its claim")
	require.NotContains(t, f.claim(liveDispatcherId, buildsDAG), operator.ID, "an operator on a live dispatcher is not claimable by another")

	require.NoError(t, f.dispatchers.MarkInactive(f.ctx, leavingDispatcherId))

	require.Empty(t, f.claim(leavingDispatcherId, buildsDAG), "an inactive dispatcher claims nothing")
	require.Contains(t, f.claim(liveDispatcherId, buildsDAG), operator.ID, "an operator on an inactive dispatcher is claimable before the dispatcher row is deleted")
}

// Free rows are split across the live dispatchers: each claims up to ceil(rows / live
// dispatchers), less what it already holds, so the first dispatcher to poll after rows come
// free does not take them all.
func TestClaimOperators_ClaimsUpToFairShare(t *testing.T) {
	pool := isolatedClaimPool(t)
	f := newClaimFixture(t, pool)

	first := seedDispatcher(t, f.ctx, pool)
	second := seedDispatcher(t, f.ctx, pool)

	all := f.dagOperators(4)

	firstClaim := f.claim(first, buildsDAG)
	require.Len(t, firstClaim, 2, "the first dispatcher to poll takes its share of the free rows, not all of them")
	f.host(first, firstClaim...)

	require.ElementsMatch(t, firstClaim, f.claim(first, buildsDAG), "a dispatcher at its share claims nothing more")

	secondClaim := f.claim(second, buildsDAG)
	require.Len(t, secondClaim, 2)
	require.ElementsMatch(t, all, append(firstClaim, secondClaim...), "the other dispatcher takes the rest")
	f.host(second, secondClaim...)

	// the first dispatcher shuts down: the two live dispatchers split its rows
	third := seedDispatcher(t, f.ctx, pool)
	require.NoError(t, f.dispatchers.MarkInactive(f.ctx, first))

	require.ElementsMatch(t, secondClaim, f.claim(second, buildsDAG), "a dispatcher at its share takes none of the released rows")
	require.ElementsMatch(t, firstClaim, f.claim(third, buildsDAG), "the dispatcher under its share takes them")
}

// The share limits new claims only. A dispatcher over its share keeps every row it holds,
// since nobody else could claim a held row it left out, and new rows go to the dispatchers
// under their share.
func TestClaimOperators_KeepsHeldRowsOverItsShare(t *testing.T) {
	pool := isolatedClaimPool(t)
	f := newClaimFixture(t, pool)

	loaded := seedDispatcher(t, f.ctx, pool)
	held := f.dagOperators(3)
	f.host(loaded, held...)

	idle := seedDispatcher(t, f.ctx, pool)
	added := f.dagOperator()

	require.ElementsMatch(t, held, f.claim(loaded, buildsDAG), "a dispatcher over its share keeps its rows and claims no new one")
	require.Equal(t, []uuid.UUID{added.ID}, f.claim(idle, buildsDAG), "a new row goes to the dispatcher under its share")

	// a late heartbeat makes the dispatcher's rows free for the others, not for itself
	_, err := pool.Exec(f.ctx, `UPDATE "Dispatcher" SET "lastHeartbeatAt" = NOW() - INTERVAL '1 minute' WHERE "id" = $1`, loaded)
	require.NoError(t, err)

	require.ElementsMatch(t, held, f.claim(loaded, buildsDAG), "a dispatcher with a late heartbeat keeps the rows it holds")
}

// A held row is returned while another transaction holds its row lock, as an action delta on
// the operator's worker does; leaving it out would tear the operator down.
func TestClaimOperators_KeepsHeldRowWhileLocked(t *testing.T) {
	pool := isolatedClaimPool(t)
	f := newClaimFixture(t, pool)

	dispatcherId := seedDispatcher(t, f.ctx, pool)
	operator := f.dagOperator()
	f.host(dispatcherId, operator.ID)

	tx, err := pool.Begin(f.ctx)
	require.NoError(t, err)

	defer func() { _ = tx.Rollback(f.ctx) }()

	_, err = tx.Exec(f.ctx, `SELECT id FROM v1_operator WHERE id = $1 FOR UPDATE`, operator.ID)
	require.NoError(t, err)

	require.Equal(t, []uuid.UUID{operator.ID}, f.claim(dispatcherId, buildsDAG))
}

// Only rows the dispatcher can build are claimed, GRPC rows by name and every other kind by
// kind, and rows nobody builds do not count towards the share: they are never registered, so
// they would stay free and use up the share of every dispatcher.
func TestClaimOperators_OnlyBuildableRows(t *testing.T) {
	pool := isolatedClaimPool(t)
	f := newClaimFixture(t, pool)

	first := seedDispatcher(t, f.ctx, pool)
	second := seedDispatcher(t, f.ctx, pool)

	dag := f.dagOperators(2)

	tenantId := seedTenantForActions(t, f.ctx, pool)
	unbuilt := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	contract := uuid.MustParse("00000000-0000-0000-0000-000000000002")

	for id, name := range map[uuid.UUID]string{unbuilt: "unbuilt", contract: "contract"} {
		_, err := pool.Exec(f.ctx,
			`INSERT INTO v1_operator (id, tenant_id, name, kind, leasing_manager, config) VALUES ($1, $2, $3, 'GRPC', 'DISPATCHER', '{}')`,
			id, tenantId, name,
		)
		require.NoError(t, err)
	}

	claimed := f.claim(first, buildsDAG)
	require.Len(t, claimed, 1, "the share is half of the two DAG rows")
	require.Contains(t, dag, claimed[0], "rows nobody here builds are neither claimed nor counted")

	withGRPC := ClaimOperatorsOpts{Kinds: []sqlcv1.V1OperatorKind{sqlcv1.V1OperatorKindDAG, sqlcv1.V1OperatorKindGRPC}}
	require.NotContains(t, f.claim(first, withGRPC), unbuilt, "GRPC rows are built by name, not by kind")

	// with the DAG rows hosted elsewhere, the contract row is the only free one
	f.host(second, dag...)

	withContract := ClaimOperatorsOpts{Kinds: buildsDAG.Kinds, GRPCNames: []string{"contract"}}
	require.Contains(t, f.claim(first, withContract), contract, "a GRPC row of a name the dispatcher builds is claimed")
}

// A row that fails to open on every poll is never hosted, so it stays free. It must not take
// the dispatcher's last free slot on every poll and leave the row behind it unclaimed.
func TestClaimOperators_FailingRowDoesNotStarveTheRest(t *testing.T) {
	pool := isolatedClaimPool(t)
	f := newClaimFixture(t, pool)

	first := seedDispatcher(t, f.ctx, pool)
	second := seedDispatcher(t, f.ctx, pool)

	ids := f.dagOperators(4)
	slices.SortFunc(ids, func(a, b uuid.UUID) int { return bytes.Compare(a[:], b[:]) })

	// a share of 2 each, one held each: the first has room for one of the free ids[0] and ids[1]
	f.host(first, ids[2])
	f.host(second, ids[3])

	claimed := map[uuid.UUID]bool{}

	for range 50 {
		for _, id := range f.claim(first, buildsDAG) {
			claimed[id] = true
		}
	}

	require.True(t, claimed[ids[1]], "the row after one that never opens is still claimed")
}
