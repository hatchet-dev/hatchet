package repository

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// Two registrations that put the same new workflow name at once both succeed: the insert
// yields to the winner and the other continues as a new version of the row it created.
func TestPutWorkflowVersionConcurrentCreateOfTheSameName(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()
	repo := newWorkflowTestRepository(pool)

	const puts = 8

	var wg sync.WaitGroup
	errs := make([]error, puts)

	for i := 0; i < puts; i++ {
		wg.Add(1)

		go func(i int) {
			defer wg.Done()

			_, errs[i] = repo.PutWorkflowVersion(ctx, internalTenantId, minimalWorkflowOpts("concurrent-create", "v1", nil))
		}(i)
	}

	wg.Wait()

	for i, err := range errs {
		require.NoError(t, err, "put %d", i)
	}

	wf, err := sqlcv1.New().GetWorkflowByName(ctx, pool, sqlcv1.GetWorkflowByNameParams{
		Tenantid: internalTenantId,
		Name:     "concurrent-create",
	})
	require.NoError(t, err)
	require.NotEqual(t, "", wf.ID.String())

	var count int
	require.NoError(t, pool.QueryRow(ctx, `SELECT count(*) FROM "Workflow" WHERE "tenantId" = $1 AND name = $2`, internalTenantId, "concurrent-create").Scan(&count))
	require.Equal(t, 1, count)
}
