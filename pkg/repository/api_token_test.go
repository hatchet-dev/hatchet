//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/validator"
	"github.com/stretchr/testify/require"
)

func TestAPITokenReadOnlyPersistence(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()
	ctx := context.Background()
	repo := newAPITokenRepository(&sharedRepository{pool: pool, queries: sqlcv1.New(), v: validator.NewDefaultValidator()}, time.Minute)
	for _, readOnly := range []bool{false, true} {
		row, err := repo.CreateAPIToken(ctx, &CreateAPITokenOpts{ID: uuid.New(), ExpiresAt: time.Now().Add(time.Hour), ReadOnly: readOnly})
		require.NoError(t, err)
		require.Equal(t, readOnly, row.ReadOnly)
		loaded, err := repo.GetAPITokenById(ctx, row.ID)
		require.NoError(t, err)
		require.Equal(t, readOnly, loaded.ReadOnly)
		cached, err := repo.GetAPITokenById(ctx, row.ID)
		require.NoError(t, err)
		require.Equal(t, readOnly, cached.ReadOnly)
	}
	id := uuid.New()
	_, err := pool.Exec(ctx, `INSERT INTO "APIToken" ("id") VALUES ($1)`, id)
	require.NoError(t, err)
	loaded, err := repo.GetAPITokenById(ctx, id)
	require.NoError(t, err)
	require.False(t, loaded.ReadOnly)
}
