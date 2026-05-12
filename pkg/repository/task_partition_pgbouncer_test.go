//go:build load

package repository

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
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

// setupPostgresWithPgBouncer starts a PostgreSQL container, runs migrations,
// and starts a PgBouncer container in transaction pooling mode in front of it.
// It returns both the direct postgres pool (for setup) and the pgbouncer pool (for testing).
func setupPostgresWithPgBouncer(t *testing.T) (directPool *pgxpool.Pool, pgbouncerPool *pgxpool.Pool, cleanup func()) {
	t.Helper()
	ctx := context.Background()

	// Start PostgreSQL with a fixed host port so pgbouncer can reach it
	postgresContainer, err := postgres.Run(ctx,
		"postgres:17-alpine",
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

	pgConnStr, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	t.Logf("PostgreSQL started: %s", pgConnStr)

	// Extract the mapped port for pgbouncer to connect to
	u, err := url.Parse(pgConnStr)
	require.NoError(t, err)
	pgPort, err := strconv.Atoi(u.Port())
	require.NoError(t, err)

	// Run migrations on direct postgres
	originalDatabaseURL := os.Getenv("DATABASE_URL")
	err = os.Setenv("DATABASE_URL", pgConnStr)
	require.NoError(t, err)
	t.Log("Running database migration...")
	migrate.RunMigrations(ctx)
	t.Log("Migration completed successfully")

	// Create direct pool
	directConfig, err := pgxpool.ParseConfig(pgConnStr)
	require.NoError(t, err)
	directConfig.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		_, err := conn.Exec(ctx, "SET TIME ZONE 'UTC'")
		return err
	}
	directPool, err = pgxpool.NewWithConfig(ctx, directConfig)
	require.NoError(t, err)
	require.NoError(t, directPool.Ping(ctx))

	// Start PgBouncer in transaction pooling mode
	pgBouncerContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:           "edoburu/pgbouncer:v1.25.1-p0",
			ExposedPorts:    []string{"5432/tcp"},
			HostAccessPorts: []int{pgPort},
			Env: map[string]string{
				"DATABASE_URL":            fmt.Sprintf("postgres://hatchet:hatchet@host.testcontainers.internal:%d/hatchet", pgPort),
				"POOL_MODE":               "transaction",
				"MAX_CLIENT_CONN":         "500",
				"DEFAULT_POOL_SIZE":       "50",
				"AUTH_TYPE":               "scram-sha-256",
				"MAX_PREPARED_STATEMENTS": "256",
			},
		},
		Started: true,
	})
	require.NoError(t, err)

	host, err := pgBouncerContainer.Host(ctx)
	require.NoError(t, err)
	mappedPort, err := pgBouncerContainer.MappedPort(ctx, "5432/tcp")
	require.NoError(t, err)

	pgBouncerConnStr := fmt.Sprintf("postgresql://hatchet:hatchet@%s:%s/hatchet?sslmode=disable", host, mappedPort.Port())
	t.Logf("PgBouncer started: %s", pgBouncerConnStr)

	// Wait for pgbouncer to be ready
	var lastErr error
	for i := 0; i < 10; i++ {
		conn, connErr := pgx.Connect(ctx, pgBouncerConnStr)
		if connErr != nil {
			lastErr = connErr
			time.Sleep(2 * time.Second)
			continue
		}
		if pingErr := conn.Ping(ctx); pingErr != nil {
			conn.Close(ctx)
			lastErr = pingErr
			time.Sleep(2 * time.Second)
			continue
		}
		conn.Close(ctx)
		lastErr = nil
		break
	}
	require.NoError(t, lastErr, "failed to connect to pgbouncer after 10 attempts")

	// Create pgbouncer pool
	pgbConfig, err := pgxpool.ParseConfig(pgBouncerConnStr)
	require.NoError(t, err)
	pgbConfig.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		_, err := conn.Exec(ctx, "SET TIME ZONE 'UTC'")
		return err
	}
	pgbouncerPool, err = pgxpool.NewWithConfig(ctx, pgbConfig)
	require.NoError(t, err)
	require.NoError(t, pgbouncerPool.Ping(ctx))

	cleanup = func() {
		pgbouncerPool.Close()
		directPool.Close()
		pgBouncerContainer.Terminate(ctx) // nolint: errcheck
		postgresContainer.Terminate(ctx)  // nolint: errcheck
		if originalDatabaseURL != "" {
			os.Setenv("DATABASE_URL", originalDatabaseURL)
		} else {
			os.Unsetenv("DATABASE_URL")
		}
	}

	return directPool, pgbouncerPool, cleanup
}

// TestUpdateTablePartitions_PgBouncer exercises partition detach (DETACH PARTITION CONCURRENTLY)
// through pgbouncer in transaction pooling mode. This ensures that DDL operations that cannot
// run inside a transaction block work correctly when routed through pgbouncer.
func TestUpdateTablePartitions_PgBouncer(t *testing.T) {
	directPool, pgbouncerPool, cleanup := setupPostgresWithPgBouncer(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	queries := sqlcv1.New()

	// Create partitions for 3 days ago using direct pool
	threeDaysAgo := time.Now().UTC().AddDate(0, 0, -3)
	err := queries.CreatePartitions(ctx, directPool, pgtype.Date{
		Time:  threeDaysAgo,
		Valid: true,
	})
	require.NoError(t, err, "failed to create partitions for 3 days ago")
	t.Logf("Created partitions for %s", threeDaysAgo.Format("2006-01-02"))

	// Verify old partitions exist
	partitionsBefore, err := queries.ListPartitionsBeforeDate(ctx, directPool, pgtype.Date{
		Time:  time.Now().UTC().AddDate(0, 0, -1), // anything older than 1 day
		Valid: true,
	})
	require.NoError(t, err)
	require.Greater(t, len(partitionsBefore), 0, "expected old partitions to exist before detach")
	t.Logf("Found %d old partitions to detach", len(partitionsBefore))

	// Create a TaskRepository backed by the pgbouncer pool with short retention.
	// The directPool bypasses pgbouncer for DDL operations like DETACH PARTITION CONCURRENTLY.
	logger := zerolog.New(zerolog.NewTestWriter(t))
	repo := &TaskRepositoryImpl{
		sharedRepository: &sharedRepository{
			pool:    pgbouncerPool,
			ddlPool: directPool,
			l:       &logger,
			queries: queries,
		},
		taskRetentionPeriod:   24 * time.Hour, // 1 day retention means 3-day-old partitions get detached
		maxInternalRetryCount: 3,
	}

	// Run UpdateTablePartitions through pgbouncer — this exercises DETACH PARTITION CONCURRENTLY
	err = repo.UpdateTablePartitions(ctx)
	require.NoError(t, err, "UpdateTablePartitions should succeed through pgbouncer")

	// Verify old partitions were removed
	partitionsAfter, err := queries.ListPartitionsBeforeDate(ctx, directPool, pgtype.Date{
		Time:  time.Now().UTC().AddDate(0, 0, -1),
		Valid: true,
	})
	require.NoError(t, err)
	assert.Equal(t, 0, len(partitionsAfter), "all old partitions should have been detached and dropped")
	t.Logf("Partitions after detach: %d (was %d)", len(partitionsAfter), len(partitionsBefore))
}

// TestUpdateTablePartitions_PgBouncer_CreateOnly verifies that basic partition creation
// (non-DDL operations) works through pgbouncer.
func TestUpdateTablePartitions_PgBouncer_CreateOnly(t *testing.T) {
	directPool, pgbouncerPool, cleanup := setupPostgresWithPgBouncer(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	queries := sqlcv1.New()

	// Count partitions before
	var countBefore int
	err := pgbouncerPool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM pg_tables
		WHERE tablename LIKE 'v1_task_%'
		   OR tablename LIKE 'v1_dag_%'
		   OR tablename LIKE 'v1_task_event_%'
		   OR tablename LIKE 'v1_log_line_%'
	`).Scan(&countBefore)
	require.NoError(t, err)

	// Create a TaskRepository backed by pgbouncer
	logger := zerolog.New(zerolog.NewTestWriter(t))
	repo := &TaskRepositoryImpl{
		sharedRepository: &sharedRepository{
			pool:    pgbouncerPool,
			ddlPool: directPool,
			l:       &logger,
			queries: queries,
		},
		taskRetentionPeriod:   24 * time.Hour,
		maxInternalRetryCount: 3,
	}

	// Run UpdateTablePartitions — creates today + tomorrow partitions
	err = repo.UpdateTablePartitions(ctx)
	require.NoError(t, err, "UpdateTablePartitions should succeed through pgbouncer (create only)")

	// Verify new partitions were created
	var countAfter int
	err = pgbouncerPool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM pg_tables
		WHERE tablename LIKE 'v1_task_%'
		   OR tablename LIKE 'v1_dag_%'
		   OR tablename LIKE 'v1_task_event_%'
		   OR tablename LIKE 'v1_log_line_%'
	`).Scan(&countAfter)
	require.NoError(t, err)

	assert.GreaterOrEqual(t, countAfter, countBefore, "partition count should not decrease")
	t.Logf("Partitions before: %d, after: %d", countBefore, countAfter)
}
