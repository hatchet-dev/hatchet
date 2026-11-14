//go:build !e2e && !load && !rampup && !integration

package fairpool_test

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/exaring/otelpgx"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	prommetrics "github.com/hatchet-dev/hatchet/pkg/integrations/metrics/prometheus"
	"github.com/hatchet-dev/hatchet/pkg/repository/fairpool"
)

var (
	dbOnce      sync.Once
	dbContainer *postgres.PostgresContainer
	dbURL       string
	dbErr       error
)

// databaseURL starts a Postgres container on first use and shares it for the package, so tests
// that need no database run without Docker. The tests only issue ad hoc SQL, so no Hatchet
// schema is needed.
func databaseURL(t *testing.T) string {
	t.Helper()

	dbOnce.Do(func() {
		ctx := context.Background()

		dbContainer, dbErr = postgres.Run(ctx,
			"postgres:15.6",
			postgres.WithDatabase("hatchet"),
			postgres.WithUsername("hatchet"),
			postgres.WithPassword("hatchet"),
			testcontainers.WithWaitStrategy(
				wait.ForLog("database system is ready to accept connections").
					WithOccurrence(2).
					WithStartupTimeout(30*time.Second),
			),
		)
		if dbErr != nil {
			return
		}

		dbURL, dbErr = dbContainer.ConnectionString(ctx, "sslmode=disable")
	})

	require.NoError(t, dbErr, "could not start postgres container")

	return dbURL
}

func TestMain(m *testing.M) {
	code := m.Run()

	if dbContainer != nil {
		_ = dbContainer.Terminate(context.Background())
	}

	os.Exit(code)
}

// newPool builds a pool whose tenants are capped at percent and whose shared bucket is ungated.
func newPool(t *testing.T, percent int, maxWait time.Duration, tracer pgx.QueryTracer) (*fairpool.Pool, string) {
	t.Helper()

	return newPoolWithOptions(t, fairpool.Options{MaxPercent: percent, MaxWait: maxWait}, tracer)
}

// newSharedCappedPool builds a pool whose tenants and shared bucket are both capped at percent.
func newSharedCappedPool(t *testing.T, percent int, maxWait time.Duration) (*fairpool.Pool, string) {
	t.Helper()

	return newPoolWithOptions(t, fairpool.Options{MaxPercent: percent, SharedMaxPercent: percent, MaxWait: maxWait}, nil)
}

func newPoolWithOptions(t *testing.T, opts fairpool.Options, tracer pgx.QueryTracer) (*fairpool.Pool, string) {
	t.Helper()

	cfg, err := pgxpool.ParseConfig(databaseURL(t))
	require.NoError(t, err)

	cfg.MaxConns = 4
	cfg.MinConns = 0
	if tracer != nil {
		cfg.ConnConfig.Tracer = tracer
	}

	name := "fairpool-test-" + uuid.NewString()
	opts.PoolName = name
	pool, err := fairpool.NewWithConfig(context.Background(), cfg, opts)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	return pool, name
}

// collect returns the series of c whose labels include every pair in want.
func collect(c prometheus.Collector, want map[string]string) []*dto.Metric {
	ch := make(chan prometheus.Metric)
	go func() {
		c.Collect(ch)
		close(ch)
	}()

	var found []*dto.Metric

	for metric := range ch {
		m := &dto.Metric{}
		if err := metric.Write(m); err != nil {
			continue
		}

		matched := 0
		for _, label := range m.GetLabel() {
			if v, ok := want[label.GetName()]; ok && v == label.GetValue() {
				matched++
			}
		}

		if matched == len(want) {
			found = append(found, m)
		}
	}

	return found
}

func gaugeValue(poolName, tenantID string) (value float64, found bool) {
	series := collect(prommetrics.FairpoolHeldConns, map[string]string{"pool": poolName, "tenant_id": tenantID})
	if len(series) == 0 {
		return 0, false
	}

	return series[0].GetGauge().GetValue(), true
}

func heldSeconds(poolName, tenantID string) (value float64, found bool) {
	series := collect(prommetrics.FairpoolHeldSeconds, map[string]string{"pool": poolName, "tenant_id": tenantID})
	if len(series) == 0 {
		return 0, false
	}

	return series[0].GetCounter().GetValue(), true
}

func requireHeld(t *testing.T, poolName, tenantID string, want float64) {
	t.Helper()
	got, ok := gaugeValue(poolName, tenantID)
	require.True(t, ok, "missing held-conn series for %s", tenantID)
	require.Equal(t, want, got)
}

func requireGone(t *testing.T, poolName, tenantID string) {
	t.Helper()
	_, ok := gaugeValue(poolName, tenantID)
	require.False(t, ok, "held-conn series still present for %s", tenantID)
}

func waitGone(t *testing.T, poolName, tenantID string) {
	t.Helper()
	require.Eventually(t, func() bool {
		runtime.GC()
		_, ok := gaugeValue(poolName, tenantID)
		return !ok
	}, 2*time.Second, 20*time.Millisecond)
}

func TestConnectionLimit(t *testing.T) {
	capped, _ := newPool(t, 50, time.Second, nil)
	require.Equal(t, int64(2), capped.ConnectionLimit())

	open, _ := newPool(t, 100, time.Second, nil)
	require.Equal(t, int64(0), open.ConnectionLimit())
}

func TestSharedConnectionLimit(t *testing.T) {
	tenantOnly, _ := newPool(t, 50, time.Second, nil)
	require.Equal(t, int64(2), tenantOnly.ConnectionLimit())
	require.Equal(t, int64(0), tenantOnly.SharedConnectionLimit())

	sharedOnly, _ := newPoolWithOptions(t, fairpool.Options{MaxPercent: 100, SharedMaxPercent: 25, MaxWait: time.Second}, nil)
	require.Equal(t, int64(0), sharedOnly.ConnectionLimit())
	require.Equal(t, int64(1), sharedOnly.SharedConnectionLimit())

	both, _ := newPoolWithOptions(t, fairpool.Options{MaxPercent: 50, SharedMaxPercent: 75, MaxWait: time.Second}, nil)
	require.Equal(t, int64(2), both.ConnectionLimit())
	require.Equal(t, int64(3), both.SharedConnectionLimit())
}

func TestRejectsPercentOutsideRange(t *testing.T) {
	cfg, err := pgxpool.ParseConfig("postgres://localhost/db")
	require.NoError(t, err)

	_, err = fairpool.NewWithConfig(context.Background(), cfg, fairpool.Options{MaxPercent: 101})
	require.Error(t, err)

	_, err = fairpool.NewWithConfig(context.Background(), cfg, fairpool.Options{SharedMaxPercent: 101})
	require.Error(t, err)
}

func TestTenantCapLetsOtherTenantsThrough(t *testing.T) {
	pool, name := newPool(t, 50, 2*time.Second, nil)
	ctx := context.Background()
	tenantA := uuid.New()
	tenantB := uuid.New()
	dbA := pool.ForTenant(tenantA)

	tx1, err := dbA.Begin(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { _ = tx1.Rollback(context.Background()) })

	tx2, err := dbA.Begin(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { _ = tx2.Rollback(context.Background()) })
	requireHeld(t, name, tenantA.String(), 2)

	done := make(chan error, 1)
	go func() {
		tx3, err := dbA.Begin(ctx)
		if err != nil {
			done <- err
			return
		}
		done <- tx3.Rollback(ctx)
	}()

	time.Sleep(50 * time.Millisecond)
	select {
	case err := <-done:
		t.Fatalf("acquire past the cap returned early: %v", err)
	default:
	}

	txB, err := pool.ForTenant(tenantB).Begin(ctx)
	require.NoError(t, err)
	require.NoError(t, txB.Rollback(ctx))
	requireGone(t, name, tenantB.String())

	sharedTx, err := pool.ForShared().Begin(ctx)
	require.NoError(t, err)
	require.NoError(t, sharedTx.Rollback(ctx))
	requireGone(t, name, "shared")

	shared, err := pool.Unwrap().Begin(ctx)
	require.NoError(t, err)
	require.NoError(t, shared.Rollback(ctx))

	require.NoError(t, tx1.Commit(ctx))

	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("capped acquire did not proceed after a slot was released")
	}

	require.NoError(t, tx2.Rollback(ctx))
	requireGone(t, name, tenantA.String())
}

// Waiters on a full bucket must proceed one at a time as slots free, with
// exactly one pool acquire each. Guards against checking out a connection
// before owning a slot, which costs a pool acquire per failed attempt.
func TestQueuedWaitersEachProceed(t *testing.T) {
	const waiters = 5

	pool, name := newPool(t, 50, 3*time.Second, nil)
	ctx := context.Background()
	tenantA := uuid.New()
	dbA := pool.ForTenant(tenantA)

	acquiresBefore := pool.Unwrap().Stat().AcquireCount()

	tx1, err := dbA.Begin(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { _ = tx1.Rollback(context.Background()) })
	tx2, err := dbA.Begin(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { _ = tx2.Rollback(context.Background()) })

	acquired := make(chan pgx.Tx, waiters)
	failed := make(chan error, waiters)
	for i := 0; i < waiters; i++ {
		go func() {
			tx, err := dbA.Begin(ctx)
			if err != nil {
				failed <- err
				return
			}
			acquired <- tx
		}()
	}

	time.Sleep(100 * time.Millisecond)
	require.NoError(t, tx1.Rollback(ctx))

	for i := 0; i < waiters; i++ {
		select {
		case tx := <-acquired:
			requireHeld(t, name, tenantA.String(), 2)
			require.NoError(t, tx.Rollback(ctx))
		case err := <-failed:
			t.Fatalf("waiter %d failed instead of taking the freed slot: %v", i, err)
		case <-time.After(2 * time.Second):
			t.Fatalf("waiter %d did not proceed after a slot was released", i)
		}
	}

	require.NoError(t, tx2.Rollback(ctx))
	requireGone(t, name, tenantA.String())

	acquires := pool.Unwrap().Stat().AcquireCount() - acquiresBefore
	require.Equal(t, int64(2+waiters), acquires, "each Begin should check out exactly one pool connection")
}

func TestLimitErrorAndCancel(t *testing.T) {
	pool, _ := newPool(t, 50, 150*time.Millisecond, nil)
	ctx := context.Background()
	tenantA := uuid.New()
	dbA := pool.ForTenant(tenantA)

	tx1, err := dbA.Begin(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { _ = tx1.Rollback(context.Background()) })
	tx2, err := dbA.Begin(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { _ = tx2.Rollback(context.Background()) })

	_, err = dbA.Begin(ctx)
	var limitErr *fairpool.LimitError
	require.ErrorAs(t, err, &limitErr)
	require.Equal(t, tenantA.String(), limitErr.Key)
	require.Equal(t, tenantA, limitErr.TenantID)
	require.Equal(t, int64(2), limitErr.Limit)

	cancelCtx, cancel := context.WithCancel(ctx)
	done := make(chan error, 1)
	go func() {
		_, acquireErr := dbA.Begin(cancelCtx)
		done <- acquireErr
	}()
	time.Sleep(30 * time.Millisecond)
	cancel()

	select {
	case acquireErr := <-done:
		require.ErrorIs(t, acquireErr, context.Canceled)
	case <-time.After(2 * time.Second):
		t.Fatal("canceled acquire did not return")
	}
}

func TestUnwrapIsUngated(t *testing.T) {
	pool, name := newPool(t, 50, time.Second, nil)
	ctx := context.Background()
	raw := pool.Unwrap()

	var txs []pgx.Tx
	for i := 0; i < 3; i++ {
		tx, err := raw.Begin(ctx)
		require.NoError(t, err)
		txs = append(txs, tx)
	}
	_, sharedOK := gaugeValue(name, "shared")
	require.False(t, sharedOK)
	_, nilOK := gaugeValue(name, uuid.Nil.String())
	require.False(t, nilOK)

	for _, tx := range txs {
		require.NoError(t, tx.Rollback(ctx))
	}
}

func TestSharedCapLetsTenantsThrough(t *testing.T) {
	pool, name := newSharedCappedPool(t, 50, 150*time.Millisecond)
	ctx := context.Background()
	shared := pool.ForShared()

	tx1, err := shared.Begin(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { _ = tx1.Rollback(context.Background()) })
	tx2, err := shared.Begin(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { _ = tx2.Rollback(context.Background()) })
	requireHeld(t, name, "shared", 2)

	_, err = shared.Begin(ctx)
	var limitErr *fairpool.LimitError
	require.ErrorAs(t, err, &limitErr)
	require.Equal(t, "shared", limitErr.Key)
	require.Equal(t, uuid.Nil, limitErr.TenantID)
	require.Equal(t, int64(2), limitErr.Limit)
	require.Contains(t, limitErr.Error(), "fairpool-exhausted:")

	tenant := uuid.New()
	txT, err := pool.ForTenant(tenant).Begin(ctx)
	require.NoError(t, err)
	require.NoError(t, txT.Rollback(ctx))
	requireGone(t, name, tenant.String())

	require.NoError(t, tx1.Rollback(ctx))
	require.NoError(t, tx2.Rollback(ctx))
	requireGone(t, name, "shared")
}

func TestNilTenantUsesSharedBucket(t *testing.T) {
	pool, name := newSharedCappedPool(t, 50, 150*time.Millisecond)
	ctx := context.Background()

	tx1, err := pool.ForTenant(uuid.Nil).Begin(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { _ = tx1.Rollback(context.Background()) })
	tx2, err := pool.ForShared().Begin(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { _ = tx2.Rollback(context.Background()) })
	requireHeld(t, name, "shared", 2)

	_, err = pool.ForTenant(uuid.Nil).Begin(ctx)
	var limitErr *fairpool.LimitError
	require.ErrorAs(t, err, &limitErr)
	require.Equal(t, "shared", limitErr.Key)
	require.Equal(t, uuid.Nil, limitErr.TenantID)

	require.NoError(t, tx1.Rollback(ctx))
	require.NoError(t, tx2.Rollback(ctx))
	requireGone(t, name, "shared")
}

func TestSharedUngatedAtFullPercent(t *testing.T) {
	pool, name := newPoolWithOptions(t, fairpool.Options{MaxPercent: 50, SharedMaxPercent: 100, MaxWait: time.Second}, nil)
	ctx := context.Background()

	var txs []pgx.Tx
	for i := 0; i < 3; i++ {
		tx, err := pool.ForShared().Begin(ctx)
		require.NoError(t, err)
		txs = append(txs, tx)
	}
	_, ok := gaugeValue(name, "shared")
	require.False(t, ok)

	for _, tx := range txs {
		require.NoError(t, tx.Rollback(ctx))
	}
}

func TestQueryKeepsCallerDeadline(t *testing.T) {
	pool, _ := newPool(t, 50, 100*time.Millisecond, nil)
	ctx := context.Background()
	tenantA := uuid.New()
	db := pool.ForTenant(tenantA)

	tx1, err := db.Begin(ctx)
	require.NoError(t, err)
	tx2, err := db.Begin(ctx)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = tx1.Rollback(context.Background())
		_ = tx2.Rollback(context.Background())
	})

	go func() {
		time.Sleep(30 * time.Millisecond)
		_ = tx1.Rollback(context.Background())
	}()

	callerCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	_, err = db.Exec(callerCtx, "SELECT pg_sleep(0.25)")
	require.NoError(t, err)
	require.NoError(t, tx2.Rollback(ctx))
}

func TestSlotReleasedForEachOperation(t *testing.T) {
	pool, name := newPool(t, 50, time.Second, nil)
	ctx := context.Background()
	tenantA := uuid.New()
	db := pool.ForTenant(tenantA)
	id := tenantA.String()

	_, err := db.Exec(ctx, "SELECT 1")
	require.NoError(t, err)
	requireGone(t, name, id)

	rows, err := db.Query(ctx, "SELECT 1")
	require.NoError(t, err)
	requireHeld(t, name, id, 1)
	rows.Close()
	requireGone(t, name, id)

	rows, err = db.Query(ctx, "SELECT 1")
	require.NoError(t, err)
	for rows.Next() {
	}
	require.NoError(t, rows.Err())
	requireGone(t, name, id)

	var n int
	err = db.QueryRow(ctx, "SELECT 1").Scan(&n)
	require.NoError(t, err)
	require.Equal(t, 1, n)
	requireGone(t, name, id)

	batch := &pgx.Batch{}
	batch.Queue("SELECT 1")
	results := db.SendBatch(ctx, batch)
	_, err = results.Exec()
	require.NoError(t, err)
	require.NoError(t, results.Close())
	requireGone(t, name, id)

	table := "tp_copy_" + uuid.NewString()[:8]
	_, err = pool.Unwrap().Exec(ctx, fmt.Sprintf("CREATE TABLE %s (n int)", table))
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = pool.Unwrap().Exec(context.Background(), "DROP TABLE IF EXISTS "+table)
	})
	copied, err := db.CopyFrom(ctx, pgx.Identifier{table}, []string{"n"}, pgx.CopyFromRows([][]any{{1}}))
	require.NoError(t, err)
	require.Equal(t, int64(1), copied)
	requireGone(t, name, id)

	tx, err := db.Begin(ctx)
	require.NoError(t, err)
	requireHeld(t, name, id, 1)
	require.NoError(t, tx.Commit(ctx))
	requireGone(t, name, id)

	tx, err = db.BeginTx(ctx, pgx.TxOptions{})
	require.NoError(t, err)
	require.NoError(t, tx.Rollback(ctx))
	requireGone(t, name, id)

	conn, err := db.Acquire(ctx)
	require.NoError(t, err)
	requireHeld(t, name, id, 1)
	conn.Release()
	requireGone(t, name, id)

	conn, err = db.Acquire(ctx)
	require.NoError(t, err)
	raw := conn.Hijack()
	requireGone(t, name, id)
	require.NoError(t, raw.Close(ctx))

	tx, err = db.Begin(ctx)
	require.NoError(t, err)
	require.NoError(t, tx.Conn().Close(ctx))
	_ = tx.Rollback(ctx)
	requireGone(t, name, id)
}

func TestFinishedResultsRejectReuse(t *testing.T) {
	pool, _ := newPool(t, 50, time.Second, nil)
	ctx := context.Background()
	db := pool.ForTenant(uuid.New())

	tx, err := db.Begin(ctx)
	require.NoError(t, err)
	require.NoError(t, tx.Rollback(ctx))
	require.ErrorIs(t, tx.Commit(ctx), pgx.ErrTxClosed)

	tx, err = db.Begin(ctx)
	require.NoError(t, err)
	require.NoError(t, tx.Commit(ctx))
	require.ErrorIs(t, tx.Rollback(ctx), pgx.ErrTxClosed)

	batch := &pgx.Batch{}
	batch.Queue("SELECT 1/0")
	results := db.SendBatch(ctx, batch)
	first := results.Close()
	require.Error(t, first)
	require.Equal(t, first, results.Close())
}

func TestAbandonedResultsReleaseSlots(t *testing.T) {
	pool, name := newPool(t, 50, time.Second, nil)
	tenantA := uuid.New()
	db := pool.ForTenant(tenantA)
	id := tenantA.String()

	abandonRows(t, db)
	waitGone(t, name, id)

	abandonRow(t, db)
	waitGone(t, name, id)

	abandonTx(t, db)
	waitGone(t, name, id)

	abandonConn(t, db)
	waitGone(t, name, id)
}

func abandonRows(t *testing.T, db fairpool.Handle) {
	t.Helper()
	rows, err := db.Query(context.Background(), "SELECT 1")
	require.NoError(t, err)
	runtime.KeepAlive(rows)
}

func abandonRow(t *testing.T, db fairpool.Handle) {
	t.Helper()
	row := db.QueryRow(context.Background(), "SELECT 1")
	runtime.KeepAlive(row)
}

func abandonTx(t *testing.T, db fairpool.Handle) {
	t.Helper()
	tx, err := db.Begin(context.Background())
	require.NoError(t, err)
	runtime.KeepAlive(tx)
}

func abandonConn(t *testing.T, db fairpool.Handle) {
	t.Helper()
	conn, err := db.Acquire(context.Background())
	require.NoError(t, err)
	runtime.KeepAlive(conn)
}

func TestAcquireSpanStaysOnInnerPool(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	tracer := otelpgx.NewTracer(otelpgx.WithTracerProvider(provider))

	pool, _ := newPool(t, 50, time.Second, tracer)
	ctx, span := provider.Tracer("fairpool-test").Start(context.Background(), "parent")

	rows, err := pool.ForTenant(uuid.New()).Query(ctx, "SELECT 1")
	require.NoError(t, err)
	rows.Close()
	span.End()

	var found bool
	for _, recorded := range recorder.Ended() {
		if recorded.Name() == "pool.acquire" {
			found = true
		}
	}
	require.True(t, found, "pool.acquire span was not recorded")
}

// An uncapped tenant may hold the whole pool. Its next checkout waits for the pool, past MaxWait
// and without a LimitError, until a connection is released or the caller's context ends.
func TestUncappedTenantWaitsOnlyForPool(t *testing.T) {
	const maxWait = 100 * time.Millisecond

	for _, percent := range []int{0, 100} {
		t.Run(fmt.Sprintf("percent_%d", percent), func(t *testing.T) {
			pool, name := newPoolWithOptions(t, fairpool.Options{MaxPercent: percent, MaxWait: maxWait}, nil)
			ctx := context.Background()
			tenantA := uuid.New()
			dbA := pool.ForTenant(tenantA)

			var txs []pgx.Tx
			for i := 0; i < int(pool.Config().MaxConns); i++ {
				tx, err := dbA.Begin(ctx)
				require.NoError(t, err)
				t.Cleanup(func() { _ = tx.Rollback(context.Background()) })
				txs = append(txs, tx)
			}
			// held_connections is reported only for capped buckets
			requireGone(t, name, tenantA.String())

			deadlineCtx, cancel := context.WithTimeout(ctx, 3*maxWait)
			defer cancel()
			_, err := dbA.Begin(deadlineCtx)
			require.ErrorIs(t, err, context.DeadlineExceeded)
			require.NotErrorAs(t, err, new(*fairpool.LimitError))

			done := make(chan error, 1)
			go func() {
				tx, err := dbA.Begin(ctx)
				if err != nil {
					done <- err
					return
				}
				done <- tx.Rollback(ctx)
			}()

			time.Sleep(3 * maxWait)
			select {
			case err := <-done:
				t.Fatalf("checkout on a full pool returned before a connection was released: %v", err)
			default:
			}

			require.NoError(t, txs[0].Rollback(ctx))

			select {
			case err := <-done:
				require.NoError(t, err)
			case <-time.After(2 * time.Second):
				t.Fatal("checkout did not proceed after a connection was released")
			}

			for _, tx := range txs[1:] {
				require.NoError(t, tx.Rollback(ctx))
			}

			require.Empty(t, collect(prommetrics.FairpoolRejections, map[string]string{"pool": name}))
			require.Empty(t, collect(prommetrics.FairpoolWait, map[string]string{"pool": name}))
		})
	}
}

// The upper bounds only catch a hold counted many times over; they leave room for a slow runner.
func TestHeldSecondsCountsEachBucket(t *testing.T) {
	const sleep = 0.1

	for _, percent := range []int{0, 50} {
		t.Run(fmt.Sprintf("percent_%d", percent), func(t *testing.T) {
			pool, name := newSharedCappedPool(t, percent, time.Second)
			ctx := context.Background()
			tenantA := uuid.New()
			tenantB := uuid.New()

			_, err := pool.ForTenant(tenantA).Exec(ctx, fmt.Sprintf("SELECT pg_sleep(%g)", sleep))
			require.NoError(t, err)

			tx, err := pool.ForTenant(tenantA).Begin(ctx)
			require.NoError(t, err)
			_, err = tx.Exec(ctx, fmt.Sprintf("SELECT pg_sleep(%g)", sleep))
			require.NoError(t, err)
			require.NoError(t, tx.Commit(ctx))

			heldA, ok := heldSeconds(name, tenantA.String())
			require.True(t, ok)
			require.GreaterOrEqual(t, heldA, 2*sleep)
			require.Less(t, heldA, 2*sleep+2)

			_, err = pool.ForShared().Exec(ctx, fmt.Sprintf("SELECT pg_sleep(%g)", sleep))
			require.NoError(t, err)

			heldShared, ok := heldSeconds(name, "shared")
			require.True(t, ok)
			require.GreaterOrEqual(t, heldShared, sleep)
			require.Less(t, heldShared, sleep+2)

			_, err = pool.ForTenant(tenantB).Exec(ctx, "SELECT 1")
			require.NoError(t, err)

			heldB, ok := heldSeconds(name, tenantB.String())
			require.True(t, ok)
			require.Greater(t, heldB, 0.0)
			require.Less(t, heldB, 2.0)

			_, err = pool.Unwrap().Exec(ctx, "SELECT 1")
			require.NoError(t, err)

			_, err = fairpool.Ungated(pool.Unwrap()).ForTenant(tenantA).Exec(ctx, fmt.Sprintf("SELECT pg_sleep(%g)", sleep))
			require.NoError(t, err)

			stillA, _ := heldSeconds(name, tenantA.String())
			require.Equal(t, heldA, stillA, "other buckets, Unwrap and Ungated must not add to tenant A")
		})
	}
}

// Pools in one process may share a PoolName. Their holds add to one series, and each keeps adding
// after the other's bucket for the same tenant has come and gone.
func TestPoolsWithOneNameShareSeries(t *testing.T) {
	const sleep = 0.05

	capped, name := newPool(t, 50, time.Second, nil)
	cfg := capped.Config()
	uncapped, err := fairpool.NewWithConfig(context.Background(), cfg, fairpool.Options{PoolName: name})
	require.NoError(t, err)
	t.Cleanup(uncapped.Close)

	ctx := context.Background()
	tenant := uuid.New()
	total := 0.0

	for _, pool := range []*fairpool.Pool{capped, uncapped, capped, uncapped} {
		_, err := pool.ForTenant(tenant).Exec(ctx, fmt.Sprintf("SELECT pg_sleep(%g)", sleep))
		require.NoError(t, err)

		held, ok := heldSeconds(name, tenant.String())
		require.True(t, ok)
		require.GreaterOrEqual(t, held, total+sleep)
		total = held
	}
}

// A Query that fails before it returns rows still returns rows that carry the error, as pgxpool's
// Query does, so a caller that ignores Query's error and checks rows.Err sees it.
func TestFailedQueryReturnsRowsWithError(t *testing.T) {
	capped, _ := newPool(t, 50, 50*time.Millisecond, nil)
	uncapped, _ := newPoolWithOptions(t, fairpool.Options{}, nil)
	ctx := context.Background()
	tenant := uuid.New()

	canceled, cancel := context.WithCancel(ctx)
	cancel()

	queries := map[string]func(context.Context) (pgx.Rows, error){
		"pgxpool": func(ctx context.Context) (pgx.Rows, error) { return uncapped.Unwrap().Query(ctx, "SELECT 1") },
		"ungated": func(ctx context.Context) (pgx.Rows, error) {
			return fairpool.Ungated(uncapped.Unwrap()).ForTenant(tenant).Query(ctx, "SELECT 1")
		},
		"uncapped": func(ctx context.Context) (pgx.Rows, error) { return uncapped.ForTenant(tenant).Query(ctx, "SELECT 1") },
		"capped":   func(ctx context.Context) (pgx.Rows, error) { return capped.ForTenant(tenant).Query(ctx, "SELECT 1") },
		"batch": func(ctx context.Context) (pgx.Rows, error) {
			batch := &pgx.Batch{}
			batch.Queue("SELECT 1")
			return uncapped.ForTenant(tenant).SendBatch(ctx, batch).Query()
		},
	}

	requireErrRows := func(t *testing.T, rows pgx.Rows, err error, target error) {
		t.Helper()
		require.ErrorIs(t, err, target)
		require.NotNil(t, rows)
		require.False(t, rows.Next())
		require.ErrorIs(t, rows.Err(), target)
		rows.Close()
		_, collectErr := pgx.CollectRows(rows, pgx.RowTo[int])
		require.ErrorIs(t, collectErr, target)
	}

	for name, query := range queries {
		t.Run(name, func(t *testing.T) {
			rows, err := query(canceled)
			requireErrRows(t, rows, err, context.Canceled)
		})
	}

	t.Run("limit", func(t *testing.T) {
		db := capped.ForTenant(tenant)
		for i := 0; i < 2; i++ {
			tx, err := db.Begin(ctx)
			require.NoError(t, err)
			t.Cleanup(func() { _ = tx.Rollback(context.Background()) })
		}

		var limitErr *fairpool.LimitError
		rows, err := db.Query(ctx, "SELECT 1")
		require.ErrorAs(t, err, &limitErr)
		requireErrRows(t, rows, err, limitErr)
	})
}
