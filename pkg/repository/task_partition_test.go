//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/hatchet-dev/hatchet/cmd/hatchet-migrate/migrate"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func setupPostgresWithMigration(t *testing.T) (*pgxpool.Pool, func()) {
	ctx := context.Background()

	postgresContainer, err := postgres.Run(ctx,
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
	require.NoError(t, err)

	connStr, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	t.Logf("PostgreSQL container started with connection string: %s", connStr)

	originalDatabaseURL := os.Getenv("DATABASE_URL")
	err = os.Setenv("DATABASE_URL", connStr)
	require.NoError(t, err)

	t.Log("Running database migration...")
	migrate.RunMigrations(ctx)
	t.Log("Migration completed successfully")

	config, err := pgxpool.ParseConfig(connStr)
	require.NoError(t, err)

	// Set timezone to UTC for all test connections
	config.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		_, err := conn.Exec(ctx, "SET TIME ZONE 'UTC'")
		return err
	}

	pool, err := pgxpool.NewWithConfig(ctx, config)
	require.NoError(t, err)

	err = pool.Ping(ctx)
	require.NoError(t, err)

	cleanup := func() {
		pool.Close()
		postgresContainer.Terminate(ctx)
		if originalDatabaseURL != "" {
			os.Setenv("DATABASE_URL", originalDatabaseURL)
		} else {
			os.Unsetenv("DATABASE_URL")
		}
	}

	return pool, cleanup
}

func createTaskRepository(pool *pgxpool.Pool) *TaskRepositoryImpl {
	return createTaskRepositoryWithDDLPool(pool, pool)
}

// createTaskRepositoryWithDDLPool lets tests use a separate ddlPool so that lock-holding
// connections and DDL connections never compete for slots in the same pool.
func createTaskRepositoryWithDDLPool(pool, ddlPool *pgxpool.Pool) *TaskRepositoryImpl {
	logger := zerolog.Nop()
	shared := &sharedRepository{
		pool:    pool,
		ddlPool: ddlPool,
		l:       &logger,
		queries: sqlcv1.New(),
	}
	return &TaskRepositoryImpl{
		sharedRepository:      shared,
		taskRetentionPeriod:   24 * time.Hour,
		maxInternalRetryCount: 3,
	}
}

func TestUpdateTablePartitions_ConcurrentControllers(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	repos := make([]*TaskRepositoryImpl, 4)
	for i := 0; i < 4; i++ {
		repos[i] = createTaskRepository(pool)
	}

	var (
		successCount int64
		errorCount   int64
	)

	const numControllers = 4
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

func TestUpdateTablePartitions_SerialExecution(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	repo := createTaskRepository(pool)

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

func TestUpdateTablePartitions_LeaseBehavior(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()
	repo := createTaskRepository(pool)

	// Acquire the lease directly to verify the mechanism.
	leases, err := repo.acquirePartitionLease(ctx, repo.ddlPool, "v1_task_partitions_test")
	require.NoError(t, err)
	assert.Len(t, leases, 1, "First caller should acquire the lease")

	// A second caller with the same key should get nothing while the lease is held.
	repo2 := createTaskRepository(pool)
	leases2, err := repo2.acquirePartitionLease(ctx, repo.ddlPool, "v1_task_partitions_test")
	require.NoError(t, err)
	assert.Len(t, leases2, 0, "Second caller should not acquire while the lease is held")

	// Release and verify a new caller can now acquire.
	err = repo.releasePartitionLease(ctx, repo.ddlPool, leases)
	require.NoError(t, err)

	leases3, err := repo2.acquirePartitionLease(ctx, repo.ddlPool, "v1_task_partitions_test")
	require.NoError(t, err)
	assert.Len(t, leases3, 1, "Third caller should acquire after release")

	err = repo2.releasePartitionLease(ctx, repo.ddlPool, leases3)
	require.NoError(t, err)
}

func TestUpdateTablePartitions_RealPartitionCreation(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	repo := createTaskRepository(pool)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var countBefore int
	err := pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM pg_tables
		WHERE tablename LIKE 'v1_task_%'
		   OR tablename LIKE 'v1_dag_%'
		   OR tablename LIKE 'v1_task_event_%'
		   OR tablename LIKE 'v1_log_line_%'
	`).Scan(&countBefore)
	require.NoError(t, err)

	err = repo.UpdateTablePartitions(ctx)
	require.NoError(t, err)

	var countAfter int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM pg_tables
		WHERE tablename LIKE 'v1_task_%'
		   OR tablename LIKE 'v1_dag_%'
		   OR tablename LIKE 'v1_task_event_%'
		   OR tablename LIKE 'v1_log_line_%'
	`).Scan(&countAfter)
	require.NoError(t, err)

	// The migration already creates partitions for today, so we should only see tomorrow's partitions created
	// Tomorrow's partitions: 4 tables = 4 new partitions
	expectedIncrease := 4
	assert.Equal(t, countBefore+expectedIncrease, countAfter, "Should have created partitions for tomorrow (today already exists from migration)")

	t.Logf("Partitions before: %d, after: %d (increase: %d)", countBefore, countAfter, countAfter-countBefore)
}

func TestUpdateTablePartitions_ContextCancellation(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	repo := createTaskRepository(pool)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := repo.UpdateTablePartitions(ctx)

	if err != nil {
		t.Logf("Function returned error due to context cancellation: %v", err)
		assert.Contains(t, err.Error(), "context deadline exceeded")
	}

	assert.True(t, true, "Function completed without panic")
}

// TestUpdateTablePartitions_LockTimeoutDoesNotLeak verifies that the lock_timeout set on ddlPool
// connections during partition DDL is reset before the connection is returned to the pool in every
// execution path — success, no-op, and error.
//
// The ddlPool is limited to MaxConns=1, so every Acquire returns the same physical connection that
// UpdateTablePartitions just used. If lock_timeout were not reset before release, SHOW lock_timeout
// would return "5s" instead of "0".
func TestUpdateTablePartitions_LockTimeoutDoesNotLeak(t *testing.T) {
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

	newRepo := func() *TaskRepositoryImpl {
		return createTaskRepositoryWithDDLPool(pool, ddlPool)
	}

	// Negative retention makes today's and tomorrow's partitions eligible for DETACH.
	newRepoWithOldRetention := func() *TaskRepositoryImpl {
		logger := zerolog.Nop()
		shared := &sharedRepository{
			pool:    pool,
			ddlPool: ddlPool,
			l:       &logger,
			queries: sqlcv1.New(),
		}
		return &TaskRepositoryImpl{
			sharedRepository:      shared,
			taskRetentionPeriod:   -48 * time.Hour,
			maxInternalRetryCount: 3,
		}
	}

	// holdLockOnVTask acquires ShareUpdateExclusiveLock on v1_task (the same lock ANALYZE holds).
	// Returns a release func the caller must invoke to unblock DDL.
	holdLockOnVTask := func(t *testing.T) (release func()) {
		t.Helper()
		lockConn, err := pool.Acquire(ctx)
		require.NoError(t, err)
		lockTx, err := lockConn.BeginTx(ctx, pgx.TxOptions{})
		require.NoError(t, err)
		_, err = lockTx.Exec(ctx, "LOCK TABLE v1_task IN SHARE UPDATE EXCLUSIVE MODE")
		require.NoError(t, err)
		return func() {
			_ = lockTx.Rollback(context.Background())
			lockConn.Release()
		}
	}

	tomorrowStr := time.Now().UTC().AddDate(0, 0, 1).Format("20060102")

	// Path A: CreatePartitions — today is a no-op (migration created it), tomorrow is new.
	t.Run("create_success_today_noop_tomorrow_new", func(t *testing.T) {
		err := newRepo().UpdateTablePartitions(ctx)
		require.NoError(t, err)
		checkLockTimeout(t)
	})

	// Path B: CreatePartitions — both partitions exist; both calls are no-ops.
	t.Run("create_success_both_noops", func(t *testing.T) {
		err := newRepo().UpdateTablePartitions(ctx)
		require.NoError(t, err)
		checkLockTimeout(t)
	})

	// Path C: CreatePartitions error — ATTACH PARTITION is blocked by a faux ANALYZE lock.
	// Tomorrow's v1_task partition is dropped so CreatePartitions has real DDL to attempt.
	t.Run("create_error_lock_timeout", func(t *testing.T) {
		_, err := pool.Exec(ctx, "ALTER TABLE v1_task DETACH PARTITION v1_task_"+tomorrowStr)
		require.NoError(t, err)
		_, err = pool.Exec(ctx, "DROP TABLE v1_task_"+tomorrowStr)
		require.NoError(t, err)

		release := holdLockOnVTask(t)
		err = newRepo().UpdateTablePartitions(ctx)
		release()

		require.ErrorIs(t, err, ErrPartitionLockConflict)
		checkLockTimeout(t)
	})

	// Path D: DETACH success — lock_timeout is reset after DETACH PARTITION CONCURRENTLY completes.
	// First restores tomorrow's v1_task partition (dropped in Path C), then uses negative retention
	// to make all existing partitions eligible for removal.
	t.Run("detach_success", func(t *testing.T) {
		err := newRepo().UpdateTablePartitions(ctx) // recreates tomorrow's partition
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

		release := holdLockOnVTask(t)
		err = newRepoWithOldRetention().UpdateTablePartitions(ctx)
		release()

		require.ErrorIs(t, err, ErrPartitionLockConflict)
		checkLockTimeout(t)
	})
}

// TestUpdateTablePartitions_FailsFastDuringAnalyze verifies that UpdateTablePartitions returns
// ErrPartitionLockConflict quickly (via lock_timeout) rather than blocking indefinitely when
// a concurrent session holds ShareUpdateExclusiveLock on a parent table — exactly what ANALYZE does.
func TestUpdateTablePartitions_FailsFastDuringAnalyze(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()

	// ddlPool is a separate connection pool to the same database. Using a separate pool ensures
	// the lock-holder connection (on pool) and the DDL connections (on ddlPool) never compete
	// for the same pool slots — no exhaustion, no deadlock.
	ddlPool, err := pgxpool.NewWithConfig(ctx, pool.Config())
	require.NoError(t, err)
	defer ddlPool.Close()

	// Hold ShareUpdateExclusiveLock on v1_task from the main pool, simulating ANALYZE.
	// The transaction stays open for the duration of the test so the lock persists while
	// UpdateTablePartitions runs DDL operations on ddlPool.
	lockConn, err := pool.Acquire(ctx)
	require.NoError(t, err)

	lockTx, err := lockConn.BeginTx(ctx, pgx.TxOptions{})
	require.NoError(t, err)

	_, err = lockTx.Exec(ctx, "LOCK TABLE v1_task IN SHARE UPDATE EXCLUSIVE MODE")
	require.NoError(t, err)

	defer func() {
		_ = lockTx.Rollback(context.Background())
		lockConn.Release()
	}()

	repo := createTaskRepositoryWithDDLPool(pool, ddlPool)

	start := time.Now()
	err = repo.UpdateTablePartitions(ctx)
	elapsed := time.Since(start)

	require.ErrorIs(t, err, ErrPartitionLockConflict, "should return ErrPartitionLockConflict when a competing lock is held on the parent table")
	// lock_timeout is 5s, so even with connection-acquire overhead the failure should be well
	// under 30s.
	assert.Less(t, elapsed, 30*time.Second, "should fail fast (lock_timeout=5s), not block indefinitely")

	t.Logf("Detected lock conflict and failed fast in %s", elapsed)
}

func TestDetachPartition_FailsFastDuringAnalyze(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()

	ddlPool, err := pgxpool.NewWithConfig(ctx, pool.Config())
	require.NoError(t, err)
	defer ddlPool.Close()

	// First run: no lock held. Creates tomorrow's partitions so the second run's
	// CreatePartitions calls are both no-ops and execution reaches the DETACH step.
	repo := createTaskRepositoryWithDDLPool(pool, ddlPool)
	err = repo.UpdateTablePartitions(ctx)
	require.NoError(t, err)

	// Build a repo where removeBefore = today + 2 days, making both today's and tomorrow's
	// partitions eligible for DETACH. Negative retention achieves this:
	// removeBefore = today.Add(-1 * -48h) = today + 48h.
	logger := zerolog.Nop()
	repoWithOldRetention := &TaskRepositoryImpl{
		sharedRepository: &sharedRepository{
			pool:    pool,
			ddlPool: ddlPool,
			l:       &logger,
			queries: sqlcv1.New(),
		},
		taskRetentionPeriod:   -48 * time.Hour,
		maxInternalRetryCount: 3,
	}

	// Hold ShareUpdateExclusiveLock on v1_task from the main pool, simulating ANALYZE.
	lockConn, err := pool.Acquire(ctx)
	require.NoError(t, err)

	lockTx, err := lockConn.BeginTx(ctx, pgx.TxOptions{})
	require.NoError(t, err)

	_, err = lockTx.Exec(ctx, "LOCK TABLE v1_task IN SHARE UPDATE EXCLUSIVE MODE")
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

func TestUpdateTablePartitions_LockContention(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	const numRepositories = 10
	repos := make([]*TaskRepositoryImpl, numRepositories)
	for i := 0; i < numRepositories; i++ {
		repos[i] = createTaskRepository(pool)
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
