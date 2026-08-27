//go:build !e2e && !load && !rampup && !integration

package migrations

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func setupContainerWithStatusesOlap(t *testing.T, image string) *sql.DB {
	t.Helper()

	ctx := context.Background()

	container, err := postgres.Run(ctx,
		image,
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

	t.Cleanup(func() {
		_ = container.Terminate(ctx)
	})

	connStr, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	db, err := sql.Open("pgx", connStr)
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = db.Close()
	})

	_, err = db.ExecContext(ctx, `
		CREATE TABLE v1_statuses_olap (
			external_id UUID NOT NULL,
			inserted_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
			tenant_id UUID NOT NULL,
			workflow_id UUID NOT NULL,
			readable_status TEXT NOT NULL,
			PRIMARY KEY (external_id, inserted_at)
		)
	`)
	require.NoError(t, err)

	return db
}

func requireIndexState(t *testing.T, db *sql.DB, wantExists bool, wantValid bool) {
	t.Helper()

	exists, valid, err := statusesOlapIndexState(context.Background(), db)
	require.NoError(t, err)
	assert.Equal(t, wantExists, exists, "index existence")

	if wantExists {
		assert.Equal(t, wantValid, valid, "index validity")
	}
}

func TestV10149_PlainPostgres(t *testing.T) {
	db := setupContainerWithStatusesOlap(t, "postgres:15.6")
	ctx := context.Background()

	require.NoError(t, upV10149(ctx, db))
	requireIndexState(t, db, true, true)

	// applying again is a no-op
	require.NoError(t, upV10149(ctx, db))
	requireIndexState(t, db, true, true)

	// an invalid leftover from a failed concurrent build is dropped and rebuilt
	_, err := db.ExecContext(ctx, `
		UPDATE pg_index SET indisvalid = false
		WHERE indexrelid = 'v1_statuses_olap_tenant_inserted_at_idx'::regclass
	`)
	require.NoError(t, err)
	requireIndexState(t, db, true, false)

	require.NoError(t, upV10149(ctx, db))
	requireIndexState(t, db, true, true)

	require.NoError(t, downV10149(ctx, db))
	requireIndexState(t, db, false, false)
}

func TestV10149_TimescaleHypertable(t *testing.T) {
	db := setupContainerWithStatusesOlap(t, "timescale/timescaledb:2.21.2-pg15")
	ctx := context.Background()

	_, err := db.ExecContext(ctx, `SELECT create_hypertable('v1_statuses_olap', 'inserted_at')`)
	require.NoError(t, err)

	isHypertable, err := statusesOlapIsHypertable(ctx, db)
	require.NoError(t, err)
	require.True(t, isHypertable)

	// CREATE INDEX CONCURRENTLY is not supported on hypertables; the migration
	// must take the transaction_per_chunk branch
	require.NoError(t, upV10149(ctx, db))
	requireIndexState(t, db, true, true)

	// applying again is a no-op
	require.NoError(t, upV10149(ctx, db))
	requireIndexState(t, db, true, true)

	require.NoError(t, downV10149(ctx, db))
	requireIndexState(t, db, false, false)
}
