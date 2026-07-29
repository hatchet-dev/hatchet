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
	if err := migrate.RunMigrations(ctx); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}
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
		ddlPool: pool,
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
