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
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/hatchet-dev/hatchet/internal/testutils"
	prommetrics "github.com/hatchet-dev/hatchet/pkg/integrations/metrics/prometheus"
	"github.com/hatchet-dev/hatchet/pkg/repository/fairpool"
)

var (
	prepareMu sync.Mutex
	prepared  bool
)

func prepareDB(t *testing.T) {
	t.Helper()

	prepareMu.Lock()
	done := prepared
	prepareMu.Unlock()
	if done {
		return
	}

	// The dev config selects RabbitMQ. These tests only need a database, and the
	// postgres queue uses the same local instance Prepare connects to.
	t.Setenv("SERVER_MSGQUEUE_KIND", "postgres")
	t.Setenv("SERVER_MSGQUEUE_PUBSUB_KIND", "postgres")
	testutils.Prepare(t)

	prepareMu.Lock()
	prepared = true
	prepareMu.Unlock()
}

func newPool(t *testing.T, percent int, maxWait time.Duration, tracer pgx.QueryTracer) (*fairpool.Pool, string) {
	t.Helper()
	prepareDB(t)

	cfg, err := pgxpool.ParseConfig(os.Getenv("DATABASE_URL"))
	require.NoError(t, err)

	cfg.MaxConns = 4
	cfg.MinConns = 0
	if tracer != nil {
		cfg.ConnConfig.Tracer = tracer
	}

	name := "fairpool-test-" + uuid.NewString()
	pool, err := fairpool.NewWithConfig(context.Background(), cfg, fairpool.Options{
		MaxPercent: percent,
		MaxWait:    maxWait,
		PoolName:   name,
	})
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	return pool, name
}

func gaugeValue(poolName, tenantID string) (float64, bool) {
	ch := make(chan prometheus.Metric)
	go func() {
		prommetrics.TenantPoolHeldConns.Collect(ch)
		close(ch)
	}()

	for metric := range ch {
		var m dto.Metric
		if err := metric.Write(&m); err != nil {
			continue
		}

		var gotPool, gotTenant string
		for _, label := range m.GetLabel() {
			switch label.GetName() {
			case "pool":
				gotPool = label.GetValue()
			case "tenant_id":
				gotTenant = label.GetValue()
			}
		}

		if gotPool == poolName && gotTenant == tenantID {
			return m.GetGauge().GetValue(), true
		}
	}

	return 0, false
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
	pool, name := newPool(t, 50, 150*time.Millisecond, nil)
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
	pool, name := newPool(t, 50, 150*time.Millisecond, nil)
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
	pool, name := newPool(t, 100, time.Second, nil)
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

func TestAbandonedResultsReleaseSlots(t *testing.T) {
	pool, name := newPool(t, 50, time.Second, nil)
	tenantA := uuid.New()
	db := pool.ForTenant(tenantA)
	id := tenantA.String()

	abandonRows(t, db)
	waitGone(t, name, id)

	abandonTx(t, db)
	waitGone(t, name, id)

	abandonConn(t, db)
	waitGone(t, name, id)
}

func abandonRows(t *testing.T, db fairpool.DB) {
	t.Helper()
	rows, err := db.Query(context.Background(), "SELECT 1")
	require.NoError(t, err)
	runtime.KeepAlive(rows)
}

func abandonTx(t *testing.T, db fairpool.DB) {
	t.Helper()
	tx, err := db.Begin(context.Background())
	require.NoError(t, err)
	runtime.KeepAlive(tx)
}

func abandonConn(t *testing.T, db fairpool.DB) {
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
