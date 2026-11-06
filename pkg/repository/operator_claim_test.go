//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
	"io"
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
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

	return enumerables.Map(claimed, func(row *sqlcv1.ClaimOperatorsRow) uuid.UUID { return row.V1Operator.ID })
}

// host registers a worker on the dispatcher for each operator, releases the operator's dead
// workers and opens the new worker's session, as the claimer does for the rows it claimed.
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

		_, err = f.workers.ReleaseDeadOperatorWorkers(f.ctx, op.TenantID, op.ID, worker.ID)
		require.NoError(f.t, err)

		_, err = f.workers.ActivateWorkerListener(f.ctx, op.TenantID, worker.ID, uuid.New())
		require.NoError(f.t, err)
	}
}

// hosts reports whether the dispatcher has an active worker of the operator.
func (f *claimFixture) hosts(dispatcherId, operatorId uuid.UUID) bool {
	f.t.Helper()

	var hosts bool

	err := f.pool.QueryRow(f.ctx,
		`SELECT EXISTS (SELECT 1 FROM "Worker" WHERE "dispatcherId" = $1 AND "operatorId" = $2 AND "isActive")`,
		dispatcherId, operatorId,
	).Scan(&hosts)
	require.NoError(f.t, err)

	return hosts
}

// unhost closes the session of the dispatcher's worker of the operator, as the claimer's
// teardown does.
func (f *claimFixture) unhost(dispatcherId, operatorId uuid.UUID) {
	f.t.Helper()

	_, err := f.pool.Exec(f.ctx, `UPDATE "Worker" SET "isActive" = FALSE WHERE "dispatcherId" = $1 AND "operatorId" = $2`, dispatcherId, operatorId)
	require.NoError(f.t, err)
}

// replicas is what a dispatcher that runs each DAG row on n dispatchers claims with.
func replicas(n int32) ClaimOperatorsOpts {
	return ClaimOperatorsOpts{Kinds: buildsDAG.Kinds, DAGReplicas: n}
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
	seedDispatcher(t, f.ctx, pool)

	dag := f.dagOperators(2)

	// GRPC rows that sort before every DAG row, so they would be the first free rows taken
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

	withContract := ClaimOperatorsOpts{Kinds: buildsDAG.Kinds, GRPCNames: []string{"contract"}}
	require.Contains(t, f.claim(first, withContract), contract, "a GRPC row of a name the dispatcher builds is claimed")
}

// A DAG row runs on as many dispatchers as it has replicas, each with a worker of its own. A
// dispatcher hosts a row once, and a replica whose worker is torn down is free again.
func TestClaimOperators_DAGRowRunsOnItsReplicas(t *testing.T) {
	pool := isolatedClaimPool(t)
	f := newClaimFixture(t, pool)

	first := seedDispatcher(t, f.ctx, pool)
	second := seedDispatcher(t, f.ctx, pool)
	third := seedDispatcher(t, f.ctx, pool)

	operator := f.dagOperator()

	require.Equal(t, []uuid.UUID{operator.ID}, f.claim(first, replicas(2)))
	f.host(first, operator.ID)

	require.Equal(t, []uuid.UUID{operator.ID}, f.claim(first, replicas(2)), "the first host keeps its replica")
	require.Equal(t, []uuid.UUID{operator.ID}, f.claim(second, replicas(2)), "another dispatcher takes the second replica")
	f.host(second, operator.ID)

	require.Empty(t, f.claim(third, replicas(2)), "both replicas are taken")
	require.Empty(t, f.claim(third, buildsDAG), "with one replica the row is held by one dispatcher only")

	f.unhost(first, operator.ID)

	require.Equal(t, []uuid.UUID{operator.ID}, f.claim(third, replicas(2)), "a torn-down replica is free again")
}

// Two dispatchers can both claim a row's last free replica before either registers a worker.
// Once both host it, the one with the newer worker finds the row missing from its claim, which
// tears its worker down, and does not claim the row back while the other holds it.
func TestClaimOperators_ExtraHostWithNewerWorkerDropsTheRow(t *testing.T) {
	pool := isolatedClaimPool(t)
	f := newClaimFixture(t, pool)

	older := seedDispatcher(t, f.ctx, pool)
	newer := seedDispatcher(t, f.ctx, pool)

	operator := f.dagOperator()

	require.Equal(t, []uuid.UUID{operator.ID}, f.claim(older, buildsDAG))
	require.Equal(t, []uuid.UUID{operator.ID}, f.claim(newer, buildsDAG), "the row looks free until a worker of it is registered")

	f.host(older, operator.ID)
	f.host(newer, operator.ID)

	_, err := pool.Exec(f.ctx, `UPDATE "Worker" SET "createdAt" = NOW() - INTERVAL '1 minute' WHERE "dispatcherId" = $1`, older)
	require.NoError(t, err)

	require.Equal(t, []uuid.UUID{operator.ID}, f.claim(older, buildsDAG), "the dispatcher with the oldest worker keeps the row")
	require.Empty(t, f.claim(newer, buildsDAG), "the other drops it")

	f.unhost(newer, operator.ID)

	require.Empty(t, f.claim(newer, buildsDAG), "and does not claim it back while the row's one replica is held")
}

// A dispatcher spends its share on rows short of a first replica before giving any row another,
// so every row ends up on as many dispatchers as it has replicas.
func TestClaimOperators_EveryRowGetsItsReplicas(t *testing.T) {
	pool := isolatedClaimPool(t)
	f := newClaimFixture(t, pool)

	dispatchers := []uuid.UUID{seedDispatcher(t, f.ctx, pool), seedDispatcher(t, f.ctx, pool), seedDispatcher(t, f.ctx, pool)}
	rows := f.dagOperators(3)

	for range 2 {
		for _, d := range dispatchers {
			claimed := f.claim(d, replicas(2))
			f.host(d, enumerables.Filter(claimed, func(id uuid.UUID) bool { return !f.hosts(d, id) })...)
		}
	}

	for _, row := range rows {
		hosts := 0

		for _, d := range dispatchers {
			if f.hosts(d, row) {
				hosts++
			}
		}

		require.Equal(t, 2, hosts, "row %s", row)
	}
}

// A dispatcher whose heartbeat lags looks gone to the others, and one of them takes its row.
// That registration releases the lagging dispatcher's worker, so once the heartbeat recovers
// the row stays with the dispatcher whose worker still has its actions.
func TestClaimOperators_LaggingHostLosesTheRowToTheNewOne(t *testing.T) {
	pool := isolatedClaimPool(t)
	f := newClaimFixture(t, pool)

	lagging := seedDispatcher(t, f.ctx, pool)
	other := seedDispatcher(t, f.ctx, pool)

	operator := f.dagOperator()
	f.host(lagging, operator.ID)

	_, err := pool.Exec(f.ctx, `UPDATE "Dispatcher" SET "lastHeartbeatAt" = NOW() - INTERVAL '1 minute' WHERE "id" = $1`, lagging)
	require.NoError(t, err)

	require.Equal(t, []uuid.UUID{operator.ID}, f.claim(other, buildsDAG), "the row looks free while the heartbeat lags")
	f.host(other, operator.ID)

	_, err = pool.Exec(f.ctx, `UPDATE "Dispatcher" SET "lastHeartbeatAt" = NOW() WHERE "id" = $1`, lagging)
	require.NoError(t, err)

	require.Empty(t, f.claim(lagging, buildsDAG), "the lagging dispatcher's released worker no longer holds the row")
	require.Equal(t, []uuid.UUID{operator.ID}, f.claim(other, buildsDAG))

	// if the new host leaves too, the row comes back to the lagging dispatcher as newly taken,
	// not held, so its claimer opens it on a new worker instead of running on the released one
	require.NoError(t, f.dispatchers.MarkInactive(f.ctx, other))

	claimed, err := f.operators.ClaimOperators(f.ctx, lagging, buildsDAG)
	require.NoError(t, err)
	require.Len(t, claimed, 1)
	assert.Equal(t, operator.ID, claimed[0].V1Operator.ID)
	assert.False(t, claimed[0].Held)
}

// When a dispatcher leaves, its rows are each one replica short. A short row can end up hosted
// by every dispatcher still under its share while the others have reached theirs; a dispatcher at
// its share then takes it anyway, one per poll, so the row does not stay short.
func TestClaimOperators_StrandedRowIsTakenPastTheShare(t *testing.T) {
	pool := isolatedClaimPool(t)
	f := newClaimFixture(t, pool)

	d := []uuid.UUID{seedDispatcher(t, f.ctx, pool), seedDispatcher(t, f.ctx, pool), seedDispatcher(t, f.ctx, pool), seedDispatcher(t, f.ctx, pool)}
	r := f.dagOperators(5)
	slices.SortFunc(r, func(a, b uuid.UUID) int { return strings.Compare(a.String(), b.String()) })

	// every row on two of the four dispatchers
	for row, hosts := range map[uuid.UUID][]uuid.UUID{r[0]: {d[0], d[1]}, r[1]: {d[0], d[2]}, r[2]: {d[0], d[2]}, r[3]: {d[1], d[2]}, r[4]: {d[1], d[3]}} {
		for _, host := range hosts {
			f.host(host, row)
		}
	}

	require.NoError(t, f.dispatchers.MarkInactive(f.ctx, d[1]))

	live := []uuid.UUID{d[0], d[2], d[3]}

	for range 4 {
		for _, dispatcherId := range live {
			claimed := f.claim(dispatcherId, replicas(2))
			f.host(dispatcherId, enumerables.Filter(claimed, func(id uuid.UUID) bool { return !f.hosts(dispatcherId, id) })...)
		}
	}

	for _, row := range r {
		hosts := 0

		for _, dispatcherId := range live {
			if f.hosts(dispatcherId, row) {
				hosts++
			}
		}

		require.Equal(t, 2, hosts, "row %s", row)
	}
}
