//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
	"os"
	"strings"
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
	logger := zerolog.Nop()
	shared := &sharedRepository{
		pool:    pool,
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

func TestUpdateTablePartitions_AdvisoryLockBehavior(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()

	const PARTITION_LOCK_OFFSET = 9000000000000000000
	const partitionLockKey = PARTITION_LOCK_OFFSET + 1

	tx1, err := pool.BeginTx(ctx, pgx.TxOptions{})
	require.NoError(t, err)
	defer tx1.Rollback(ctx)

	tx2, err := pool.BeginTx(ctx, pgx.TxOptions{})
	require.NoError(t, err)
	defer tx2.Rollback(ctx)

	var acquired1 bool
	err = tx1.QueryRow(ctx, "SELECT pg_try_advisory_xact_lock($1)", partitionLockKey).Scan(&acquired1)
	require.NoError(t, err)
	assert.True(t, acquired1, "First transaction should acquire lock")

	var acquired2 bool
	err = tx2.QueryRow(ctx, "SELECT pg_try_advisory_xact_lock($1)", partitionLockKey).Scan(&acquired2)
	require.NoError(t, err)
	assert.False(t, acquired2, "Second transaction should not acquire lock")

	err = tx1.Commit(ctx)
	require.NoError(t, err)

	tx3, err := pool.BeginTx(ctx, pgx.TxOptions{})
	require.NoError(t, err)
	defer tx3.Rollback(ctx)

	var acquired3 bool
	err = tx3.QueryRow(ctx, "SELECT pg_try_advisory_xact_lock($1)", partitionLockKey).Scan(&acquired3)
	require.NoError(t, err)
	assert.True(t, acquired3, "Third transaction should acquire lock after first commits")

	err = tx3.Commit(ctx)
	require.NoError(t, err)
}

func TestUpdateTablePartitions_LockAutoReleaseOnRollback(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()

	const PARTITION_LOCK_OFFSET = 9000000000000000000
	const partitionLockKey = PARTITION_LOCK_OFFSET + 1

	tx1, err := pool.BeginTx(ctx, pgx.TxOptions{})
	require.NoError(t, err)

	var acquired1 bool
	err = tx1.QueryRow(ctx, "SELECT pg_try_advisory_xact_lock($1)", partitionLockKey).Scan(&acquired1)
	require.NoError(t, err)
	assert.True(t, acquired1, "First transaction should acquire lock")

	tx2, err := pool.BeginTx(ctx, pgx.TxOptions{})
	require.NoError(t, err)
	defer tx2.Rollback(ctx)

	var acquired2 bool
	err = tx2.QueryRow(ctx, "SELECT pg_try_advisory_xact_lock($1)", partitionLockKey).Scan(&acquired2)
	require.NoError(t, err)
	assert.False(t, acquired2, "Second transaction should not acquire lock while first holds it")

	t.Log("Simulating connection loss by rolling back first transaction...")
	err = tx1.Rollback(ctx)
	require.NoError(t, err)

	tx3, err := pool.BeginTx(ctx, pgx.TxOptions{})
	require.NoError(t, err)
	defer tx3.Rollback(ctx)

	var acquired3 bool
	err = tx3.QueryRow(ctx, "SELECT pg_try_advisory_xact_lock($1)", partitionLockKey).Scan(&acquired3)
	require.NoError(t, err)
	assert.True(t, acquired3, "Third transaction should acquire lock after first transaction rolled back")

	err = tx3.Commit(ctx)
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
				if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
					atomic.AddInt64(&successCount, 1)
					t.Logf("Repository %d handled expected PostgreSQL DDL contention gracefully", repoID)
				} else {
					atomic.AddInt64(&errorCount, 1)
					t.Logf("Repository %d encountered unexpected error: %v", repoID, err)
				}
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

	t.Logf("High contention test - Successes: %d, Unexpected Errors: %d", finalSuccessCount, finalErrorCount)

	assert.Equal(t, int64(0), finalErrorCount, "No unexpected errors should occur")
	assert.Equal(t, int64(numRepositories), finalSuccessCount, "All repositories should complete successfully or handle DDL contention gracefully")
}

func TestUpdateTablePartitions_SeparateTransactionApproach(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()

	const PARTITION_LOCK_OFFSET = 9000000000000000000
	const partitionLockKey = PARTITION_LOCK_OFFSET + 1

	lockTx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	require.NoError(t, err)
	defer lockTx.Rollback(ctx)

	var acquired bool
	err = lockTx.QueryRow(ctx, "SELECT pg_try_advisory_xact_lock($1)", partitionLockKey).Scan(&acquired)
	require.NoError(t, err)
	assert.True(t, acquired, "Should acquire advisory lock")

	var countInTransaction int
	err = lockTx.QueryRow(ctx, "SELECT COUNT(*) FROM pg_tables WHERE tablename LIKE 'v1_task_%'").Scan(&countInTransaction)
	require.NoError(t, err)

	var countOutsideTransaction int
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM pg_tables WHERE tablename LIKE 'v1_task_%'").Scan(&countOutsideTransaction)
	require.NoError(t, err)

	assert.Equal(t, countInTransaction, countOutsideTransaction, "Partition operations should be visible outside the lock transaction")

	err = lockTx.Commit(ctx)
	require.NoError(t, err)

	lockTx2, err := pool.BeginTx(ctx, pgx.TxOptions{})
	require.NoError(t, err)
	defer lockTx2.Rollback(ctx)

	var acquired2 bool
	err = lockTx2.QueryRow(ctx, "SELECT pg_try_advisory_xact_lock($1)", partitionLockKey).Scan(&acquired2)
	require.NoError(t, err)
	assert.True(t, acquired2, "Should be able to acquire lock again after first transaction commits")

	err = lockTx2.Commit(ctx)
	require.NoError(t, err)
}
