//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultTaskActivityGauge(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	t.Cleanup(cleanup)

	ctx := context.Background()

	// createTaskRepository uses a 24 hour retention period
	repo := createTaskRepository(pool)

	active, stale, neverActive, noQueues := uuid.New(), uuid.New(), uuid.New(), uuid.New()

	_, err := pool.Exec(ctx, `
		INSERT INTO v1_queue (tenant_id, name, last_active)
		VALUES
			($1, 'old', NOW() - INTERVAL '10 days'),
			($1, 'recent', NOW() - INTERVAL '1 hour'),
			($2, 'old', NOW() - INTERVAL '10 days'),
			($2, 'just-outside', NOW() - INTERVAL '25 hours'),
			($3, 'never-active', NULL)
	`, active, stale, neverActive)
	require.NoError(t, err)

	for _, tc := range []struct {
		name     string
		tenantId uuid.UUID
		want     int
	}{
		{"one queue inside the retention period", active, 1},
		{"every queue outside the retention period", stale, 0},
		{"only a queue with no last_active", neverActive, 0},
		{"no queues", noQueues, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := repo.DefaultTaskActivityGauge(ctx, tc.tenantId.String())
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}

	t.Run("index from the migration is valid", func(t *testing.T) {
		var isValid bool

		err := pool.QueryRow(ctx, `
			SELECT i.indisvalid
			FROM pg_index i
			JOIN pg_class c ON c.oid = i.indexrelid
			WHERE c.relname = 'v1_queue_tenant_id_last_active_idx'
		`).Scan(&isValid)
		require.NoError(t, err, "v1_queue_tenant_id_last_active_idx should exist")
		assert.True(t, isValid)
	})
}
