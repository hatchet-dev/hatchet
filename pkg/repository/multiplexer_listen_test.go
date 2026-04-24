//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// setupPostgresPlain spins up a fresh Postgres 15.6 container and returns a
// pgxpool configured with the given MaxConns. Unlike setupPostgresWithMigration,
// it does NOT run hatchet migrations — these tests only need a raw connection
// to exercise pgxpool bookkeeping.
func setupPostgresPlain(t *testing.T, maxConns int32) (*pgxpool.Pool, func()) {
	t.Helper()

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

	config, err := pgxpool.ParseConfig(connStr)
	require.NoError(t, err)
	config.MaxConns = maxConns
	config.MinConns = 0

	pool, err := pgxpool.NewWithConfig(ctx, config)
	require.NoError(t, err)

	err = pool.Ping(ctx)
	require.NoError(t, err)

	cleanup := func() {
		pool.Close()
		_ = postgresContainer.Terminate(ctx)
	}

	return pool, cleanup
}

// TestAcquireListenerConn_ReleasesPoolSlotImmediately verifies that
// acquireListenerConn returns a raw *pgx.Conn detached from the pool, so that
// pgxpool's "acquired" bookkeeping drops back to zero right after the call.
//
// Regression guard for #3694: the previous implementation returned
// poolConn.Conn() without hijacking, letting the *pgxpool.Conn wrapper fall
// out of scope without Release() — the slot stayed permanently counted as
// acquired even though the raw conn would later be closed by pgxlisten.
func TestAcquireListenerConn_ReleasesPoolSlotImmediately(t *testing.T) {
	pool, cleanup := setupPostgresPlain(t, 5)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	require.Equal(t, int32(0), pool.Stat().AcquiredConns(),
		"pool should start with zero acquired conns")

	raw, err := acquireListenerConn(ctx, pool)
	require.NoError(t, err)
	require.NotNil(t, raw)
	defer raw.Close(ctx)

	require.Equal(t, int32(0), pool.Stat().AcquiredConns(),
		"Hijack should transfer the conn out of the pool so AcquiredConns drops to zero")
}

// TestAcquireListenerConn_SurvivesReconnectCyclePastPoolLimit simulates the
// exact scenario reported in #3694: pgxlisten repeatedly calls Connect after
// each server-side reconnect, and each returned conn is eventually closed.
// A slot-leak bug would exhaust the pool within MaxConns iterations because
// the orphaned *pgxpool.Conn wrappers would never release their slots.
//
// With acquireListenerConn's Hijack, each call is independent of pool state:
// we run many more iterations than MaxConns and assert the pool never becomes
// exhausted and AcquiredConns returns to zero after each cycle.
func TestAcquireListenerConn_SurvivesReconnectCyclePastPoolLimit(t *testing.T) {
	const maxConns int32 = 3
	const iterations = int(maxConns) * 4 // well past MaxConns

	pool, cleanup := setupPostgresPlain(t, maxConns)
	defer cleanup()

	for i := 0; i < iterations; i++ {
		// Each iteration uses its own short timeout so a would-be slot leak
		// surfaces as a deadline-exceeded error rather than hanging the test.
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		raw, err := acquireListenerConn(ctx, pool)
		require.NoErrorf(t, err, "iteration %d acquire should succeed; slot leak would starve the pool here", i)
		require.NotNil(t, raw)

		require.Equalf(t, int32(0), pool.Stat().AcquiredConns(),
			"iteration %d: AcquiredConns must be zero after Hijack", i)

		// Simulate pgxlisten's `defer conn.Close(ctx)` when Listen exits.
		err = raw.Close(ctx)
		require.NoError(t, err, "iteration %d close should not error", i)
		cancel()
	}

	require.Equal(t, int32(0), pool.Stat().AcquiredConns(),
		"pool should end the test with zero acquired conns")
}
