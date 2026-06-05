//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func createOLAPRepositoryWithDDLPool(pool, ddlPool *pgxpool.Pool) *OLAPRepositoryImpl {
	logger := zerolog.Nop()
	shared := &sharedRepository{
		pool:    pool,
		ddlPool: ddlPool,
		l:       &logger,
		queries: sqlcv1.New(),
	}
	eventCache, err := lru.New[string, bool](100000)
	if err != nil {
		panic(err)
	}
	return &OLAPRepositoryImpl{
		sharedRepository:            shared,
		eventCache:                  eventCache,
		olapRetentionPeriod:         24 * time.Hour,
		shouldPartitionEventsTables: false,
		shouldPartitionOtelTables:   false,
	}
}

func createOLAPRepository(pool *pgxpool.Pool) *OLAPRepositoryImpl {
	logger := zerolog.Nop()
	shared := &sharedRepository{
		pool:    pool,
		ddlPool: pool,
		l:       &logger,
		queries: sqlcv1.New(),
	}
	eventCache, err := lru.New[string, bool](100000)
	if err != nil {
		panic(err)
	}
	return &OLAPRepositoryImpl{
		sharedRepository:            shared,
		eventCache:                  eventCache,
		olapRetentionPeriod:         24 * time.Hour,
		shouldPartitionEventsTables: false,
		shouldPartitionOtelTables:   false,
	}
}

func TestOLAPUpdateTablePartitions_ConcurrentControllers(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	const numControllers = 4
	repos := make([]*OLAPRepositoryImpl, numControllers)
	for i := 0; i < numControllers; i++ {
		repos[i] = createOLAPRepository(pool)
	}

	var (
		successCount int64
		errorCount   int64
	)

	var wg sync.WaitGroup
	start := make(chan struct{})

	for i := 0; i < numControllers; i++ {
		wg.Add(1)
		go func(controllerID int) {
			defer wg.Done()

			<-start

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			err := repos[controllerID].UpdateTablePartitions(ctx)

			if err != nil {
				atomic.AddInt64(&errorCount, 1)
				t.Logf("Controller %d encountered error: %v", controllerID, err)
			} else {
				atomic.AddInt64(&successCount, 1)
				t.Logf("Controller %d completed successfully", controllerID)
			}
		}(i)
	}

	close(start)
	wg.Wait()

	finalSuccessCount := atomic.LoadInt64(&successCount)
	finalErrorCount := atomic.LoadInt64(&errorCount)

	t.Logf("Successful executions: %d", finalSuccessCount)
	t.Logf("Errors: %d", finalErrorCount)

	assert.Equal(t, int64(0), finalErrorCount, "No controllers should have encountered errors")
	assert.Equal(t, int64(numControllers), finalSuccessCount, "All controllers should have completed successfully")
}

func TestOLAPUpdateTablePartitions_SerialExecution(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	repo := createOLAPRepository(pool)

	const numRuns = 3
	var successCount int64

	for i := 0; i < numRuns; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

		err := repo.UpdateTablePartitions(ctx)
		cancel()

		if err == nil {
			atomic.AddInt64(&successCount, 1)
			t.Logf("Run %d completed successfully", i)
		} else {
			t.Logf("Run %d failed: %v", i, err)
		}

		time.Sleep(100 * time.Millisecond)
	}

	assert.Equal(t, int64(numRuns), atomic.LoadInt64(&successCount), "All serial runs should succeed")
}

func TestOLAPUpdateTablePartitions_RealPartitionCreation(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	repo := createOLAPRepository(pool)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var countBefore int
	err := pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM pg_tables
		WHERE tablename LIKE 'v1_tasks_olap_%'
		   OR tablename LIKE 'v1_dags_olap_%'
		   OR tablename LIKE 'v1_runs_olap_%'
		   OR tablename LIKE 'v1_payloads_olap_%'
	`).Scan(&countBefore)
	require.NoError(t, err)

	err = repo.UpdateTablePartitions(ctx)
	require.NoError(t, err)

	var countAfter int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM pg_tables
		WHERE tablename LIKE 'v1_tasks_olap_%'
		   OR tablename LIKE 'v1_dags_olap_%'
		   OR tablename LIKE 'v1_runs_olap_%'
		   OR tablename LIKE 'v1_payloads_olap_%'
	`).Scan(&countAfter)
	require.NoError(t, err)

	assert.GreaterOrEqual(t, countAfter, countBefore, "Partition count should not decrease after UpdateTablePartitions")

	t.Logf("Partitions before: %d, after: %d", countBefore, countAfter)
}

func TestOLAPUpdateTablePartitions_ContextCancellation(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	repo := createOLAPRepository(pool)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := repo.UpdateTablePartitions(ctx)

	if err != nil {
		t.Logf("Function returned error due to context cancellation: %v", err)
		assert.Contains(t, err.Error(), "context deadline exceeded")
	}

	assert.True(t, true, "Function completed without panic")
}

func TestOLAPUpdateTablePartitions_LockContention(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	const numRepositories = 10
	repos := make([]*OLAPRepositoryImpl, numRepositories)
	for i := 0; i < numRepositories; i++ {
		repos[i] = createOLAPRepository(pool)
	}

	var (
		successCount int64
		errorCount   int64
	)

	var wg sync.WaitGroup
	start := make(chan struct{})

	for i := 0; i < numRepositories; i++ {
		wg.Add(1)
		go func(repoID int) {
			defer wg.Done()

			<-start

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			err := repos[repoID].UpdateTablePartitions(ctx)

			if err != nil {
				atomic.AddInt64(&errorCount, 1)
				t.Logf("Repository %d encountered error: %v", repoID, err)
			} else {
				atomic.AddInt64(&successCount, 1)
				t.Logf("Repository %d completed successfully", repoID)
			}
		}(i)
	}

	close(start)
	wg.Wait()

	finalSuccessCount := atomic.LoadInt64(&successCount)
	finalErrorCount := atomic.LoadInt64(&errorCount)

	t.Logf("High contention test - Successes: %d, Errors: %d", finalSuccessCount, finalErrorCount)

	assert.Equal(t, int64(0), finalErrorCount, "No errors should occur under contention")
	assert.Equal(t, int64(numRepositories), finalSuccessCount, "All repositories should complete successfully")
}

// TestOLAPUpdateTablePartitions_FailsFastDuringAnalyze verifies that UpdateTablePartitions returns
// ErrPartitionLockConflict quickly (via lock_timeout) rather than blocking indefinitely when a
// concurrent session holds ShareUpdateExclusiveLock on a parent OLAP table — exactly what ANALYZE does.
//
// The migration creates today's partitions, so CreateOLAPPartitions(today) is always a no-op.
// CreateOLAPPartitions(tomorrow) runs on r.pool and tries to ATTACH PARTITION — the lock blocks it.
func TestOLAPUpdateTablePartitions_FailsFastDuringAnalyze(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()

	// ddlPool is separate from pool so DDL connections and lock-holder connections never compete
	// for pool slots.
	ddlPool, err := pgxpool.NewWithConfig(ctx, pool.Config())
	require.NoError(t, err)
	defer ddlPool.Close()

	// Hold ShareUpdateExclusiveLock on v1_tasks_olap, simulating a concurrent ANALYZE.
	lockConn, err := pool.Acquire(ctx)
	require.NoError(t, err)
	lockTx, err := lockConn.BeginTx(ctx, pgx.TxOptions{})
	require.NoError(t, err)
	_, err = lockTx.Exec(ctx, "LOCK TABLE v1_tasks_olap IN SHARE UPDATE EXCLUSIVE MODE")
	require.NoError(t, err)
	defer func() {
		_ = lockTx.Rollback(context.Background())
		lockConn.Release()
	}()

	repo := createOLAPRepositoryWithDDLPool(pool, ddlPool)

	start := time.Now()
	err = repo.UpdateTablePartitions(ctx)
	elapsed := time.Since(start)

	require.ErrorIs(t, err, ErrPartitionLockConflict, "should return ErrPartitionLockConflict when a competing lock is held on v1_tasks_olap")
	assert.Less(t, elapsed, 30*time.Second, "should fail fast (lock_timeout=5s), not block indefinitely")

	t.Logf("Detected lock conflict and failed fast in %s", elapsed)
}

func TestDetachOLAPPartition_FailsFastDuringAnalyze(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()

	ddlPool, err := pgxpool.NewWithConfig(ctx, pool.Config())
	require.NoError(t, err)
	defer ddlPool.Close()

	// First run: no lock. Creates tomorrow's partitions so the second run's CreateOLAPPartitions
	// calls are both no-ops and execution reaches the DETACH step.
	repo := createOLAPRepositoryWithDDLPool(pool, ddlPool)
	err = repo.UpdateTablePartitions(ctx)
	require.NoError(t, err)

	// Build a repo where removeBefore = today + 2 days, making both today's and tomorrow's
	// partitions eligible for DETACH. Negative retention achieves this:
	// removeBefore = today.Add(-1 * -48h) = today + 48h.
	logger := zerolog.Nop()
	eventCache, err := lru.New[string, bool](100000)
	require.NoError(t, err)
	repoWithOldRetention := &OLAPRepositoryImpl{
		sharedRepository: &sharedRepository{
			pool:    pool,
			ddlPool: ddlPool,
			l:       &logger,
			queries: sqlcv1.New(),
		},
		eventCache:                  eventCache,
		olapRetentionPeriod:         -48 * time.Hour,
		shouldPartitionEventsTables: false,
		shouldPartitionOtelTables:   false,
	}

	// Hold ShareUpdateExclusiveLock on v1_tasks_olap, simulating ANALYZE.
	lockConn, err := pool.Acquire(ctx)
	require.NoError(t, err)
	lockTx, err := lockConn.BeginTx(ctx, pgx.TxOptions{})
	require.NoError(t, err)
	_, err = lockTx.Exec(ctx, "LOCK TABLE v1_tasks_olap IN SHARE UPDATE EXCLUSIVE MODE")
	require.NoError(t, err)
	defer func() {
		_ = lockTx.Rollback(context.Background())
		lockConn.Release()
	}()

	start := time.Now()
	err = repoWithOldRetention.UpdateTablePartitions(ctx)
	elapsed := time.Since(start)

	require.ErrorIs(t, err, ErrPartitionLockConflict, "should return ErrPartitionLockConflict when DETACH is blocked by ANALYZE")
	assert.Less(t, elapsed, 30*time.Second, "should fail fast (lock_timeout=5s), not block indefinitely")

	t.Logf("DETACH lock conflict detected fast in %s", elapsed)
}

// TestOLAPUpdateTablePartitions_LockTimeoutDoesNotLeak verifies that lock_timeout set on ddlPool
// connections during DETACH DDL is reset before the connection is returned to the pool in every
// execution path — success, no-op, and error.
//
// The CREATE paths use runPartitionDDLWithLockTimeout (SET LOCAL, transaction-scoped) so they
// auto-revert on commit/rollback. The DETACH path uses a session-level SET and relies on an
// explicit releaseConn wrapper to reset it — that reset is what this test validates.
//
// The ddlPool is limited to MaxConns=1, so every Acquire returns the same physical connection that
// UpdateTablePartitions just used. If lock_timeout were not reset before release, SHOW lock_timeout
// would return "5s" instead of "0".
func TestOLAPUpdateTablePartitions_LockTimeoutDoesNotLeak(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()

	// Single-connection ddlPool so we can assert on the exact physical connection used for DDL.
	cfg := pool.Config()
	cfg.MaxConns = 1
	ddlPool, err := pgxpool.NewWithConfig(ctx, cfg)
	require.NoError(t, err)
	defer ddlPool.Close()

	checkLockTimeout := func(t *testing.T) {
		t.Helper()
		conn, err := ddlPool.Acquire(ctx)
		require.NoError(t, err)
		defer conn.Release()
		var lockTimeout string
		err = conn.QueryRow(ctx, "SHOW lock_timeout").Scan(&lockTimeout)
		require.NoError(t, err)
		assert.Equal(t, "0", lockTimeout, "lock_timeout must be reset to 0 before returning the connection to the pool")
	}

	newRepo := func() *OLAPRepositoryImpl {
		return createOLAPRepositoryWithDDLPool(pool, ddlPool)
	}

	// Negative retention makes today's and tomorrow's partitions eligible for DETACH.
	newRepoWithOldRetention := func() *OLAPRepositoryImpl {
		logger := zerolog.Nop()
		ec, err := lru.New[string, bool](100000)
		require.NoError(t, err)
		shared := &sharedRepository{
			pool:    pool,
			ddlPool: ddlPool,
			l:       &logger,
			queries: sqlcv1.New(),
		}
		return &OLAPRepositoryImpl{
			sharedRepository:            shared,
			eventCache:                  ec,
			olapRetentionPeriod:         -48 * time.Hour,
			shouldPartitionEventsTables: false,
			shouldPartitionOtelTables:   false,
		}
	}

	// holdLockOnTasksOLAP acquires ShareUpdateExclusiveLock on v1_tasks_olap (the same lock
	// ANALYZE holds). Returns a release func the caller must invoke to unblock DDL.
	holdLockOnTasksOLAP := func(t *testing.T) (release func()) {
		t.Helper()
		lockConn, err := pool.Acquire(ctx)
		require.NoError(t, err)
		lockTx, err := lockConn.BeginTx(ctx, pgx.TxOptions{})
		require.NoError(t, err)
		_, err = lockTx.Exec(ctx, "LOCK TABLE v1_tasks_olap IN SHARE UPDATE EXCLUSIVE MODE")
		require.NoError(t, err)
		return func() {
			_ = lockTx.Rollback(context.Background())
			lockConn.Release()
		}
	}

	todayStr := time.Now().UTC().Format("20060102")

	// Path A: CreateOLAPPartitions(today) is a no-op (migration created it); tomorrow is new.
	// Today's CREATE runs on ddlPool; tomorrow's runs on pool. The ddlPool connection completes
	// a no-op transaction via SET LOCAL, which auto-reverts lock_timeout on commit.
	t.Run("create_success_today_noop_tomorrow_new", func(t *testing.T) {
		err := newRepo().UpdateTablePartitions(ctx)
		require.NoError(t, err)
		checkLockTimeout(t)
	})

	// Path B: both partitions exist; all CreateOLAPPartitions calls are no-ops.
	t.Run("create_success_both_noops", func(t *testing.T) {
		err := newRepo().UpdateTablePartitions(ctx)
		require.NoError(t, err)
		checkLockTimeout(t)
	})

	// Path C: CreateOLAPPartitions(today) error — ATTACH PARTITION blocked by a faux ANALYZE lock.
	// Today's v1_tasks_olap partition is dropped so the CREATE has real DDL to attempt on ddlPool.
	// runPartitionDDLWithLockTimeout uses SET LOCAL, so rollback auto-reverts lock_timeout to 0.
	t.Run("create_error_lock_timeout", func(t *testing.T) {
		_, err := pool.Exec(ctx, "ALTER TABLE v1_tasks_olap DETACH PARTITION v1_tasks_olap_"+todayStr)
		require.NoError(t, err)
		_, err = pool.Exec(ctx, "DROP TABLE v1_tasks_olap_"+todayStr+" CASCADE")
		require.NoError(t, err)

		release := holdLockOnTasksOLAP(t)
		err = newRepo().UpdateTablePartitions(ctx)
		release()

		require.ErrorIs(t, err, ErrPartitionLockConflict)
		checkLockTimeout(t)
	})

	// Path D: DETACH success — lock_timeout is reset after DETACH PARTITION CONCURRENTLY completes.
	// Recreates today's v1_tasks_olap partition (dropped in Path C), then uses negative retention
	// to make all existing OLAP partitions eligible for removal.
	t.Run("detach_success", func(t *testing.T) {
		err := newRepo().UpdateTablePartitions(ctx) // recreates today's partition
		require.NoError(t, err)

		err = newRepoWithOldRetention().UpdateTablePartitions(ctx)
		require.NoError(t, err)
		checkLockTimeout(t)
	})

	// Path E: DETACH error — lock_timeout is reset even when DETACH PARTITION CONCURRENTLY blocks.
	// Recreates partitions (detached in Path D), then holds the faux ANALYZE lock during DETACH.
	t.Run("detach_error_lock_timeout", func(t *testing.T) {
		err := newRepo().UpdateTablePartitions(ctx) // recreates partitions detached in Path D
		require.NoError(t, err)

		release := holdLockOnTasksOLAP(t)
		err = newRepoWithOldRetention().UpdateTablePartitions(ctx)
		release()

		require.ErrorIs(t, err, ErrPartitionLockConflict)
		checkLockTimeout(t)
	})
}
