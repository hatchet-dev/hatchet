//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/repository/cache"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/validator"
)

var internalTenantId = uuid.MustParse("8d420720-ef03-41dc-9c73-1c93f276db97")

func newWorkflowTestRepository(pool *pgxpool.Pool) *workflowRepository {
	logger := zerolog.Nop()
	shared := &sharedRepository{
		pool:       pool,
		ddlPool:    pool,
		l:          &logger,
		queries:    sqlcv1.New(),
		v:          validator.NewDefaultValidator(),
		queueCache: cache.New(5 * time.Minute),
	}
	return &workflowRepository{sharedRepository: shared}
}

func countCronRefsForWorkflowName(ctx context.Context, t *testing.T, pool *pgxpool.Pool, workflowName string) int {
	t.Helper()
	var count int
	err := pool.QueryRow(ctx, `
		SELECT count(c.*)
		FROM "WorkflowTriggerCronRef" c
		JOIN "WorkflowTriggers" tr ON tr."id" = c."parentId"
		JOIN "WorkflowVersion" wv ON wv."id" = tr."workflowVersionId"
		JOIN "Workflow" w ON w."id" = wv."workflowId"
		WHERE w."name" = $1
		  AND w."deletedAt" IS NULL
	`, workflowName).Scan(&count)
	require.NoError(t, err)
	return count
}

// minimalWorkflowOpts returns valid workflow opts with a single step.
// Pass a unique description to force a new checksum on re-registration.
func minimalWorkflowOpts(name, description string, cronTriggers []string) *CreateWorkflowVersionOpts {
	desc := description
	return &CreateWorkflowVersionOpts{
		Name:         name,
		Description:  &desc,
		CronTriggers: cronTriggers,
		Tasks: []CreateStepOpts{
			{
				ReadableId: "step1",
				Action:     "integration:step1",
			},
		},
	}
}

func TestDefaultCronTriggersCleanedUpOnReregistration(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()
	repo := newWorkflowTestRepository(pool)

	const workflowName = "cron-cleanup-test"
	cronTriggers := []string{"0 * * * *", "30 * * * *"}

	// Register v1
	_, err := repo.PutWorkflowVersion(ctx, internalTenantId, minimalWorkflowOpts(workflowName, "v1", cronTriggers))
	require.NoError(t, err)

	count := countCronRefsForWorkflowName(ctx, t, pool, workflowName)
	assert.Equal(t, 2, count, "should have 2 cron refs after initial registration")

	// Register v2 — same crons, different description forces a new checksum and new version
	_, err = repo.PutWorkflowVersion(ctx, internalTenantId, minimalWorkflowOpts(workflowName, "v2", cronTriggers))
	require.NoError(t, err)

	count = countCronRefsForWorkflowName(ctx, t, pool, workflowName)
	assert.Equal(t, 2, count, "should still have 2 cron refs after re-registration — old DEFAULT rows must be deleted")
}

func TestDefaultCronsDontAccumulateAcrossMultipleVersions(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()
	repo := newWorkflowTestRepository(pool)

	const workflowName = "cron-no-accumulation-test"
	cronTriggers := []string{"0 0 * * *"}

	for i := 0; i < 5; i++ {
		desc := fmt.Sprintf("version-%d", i)
		_, err := repo.PutWorkflowVersion(ctx, internalTenantId, minimalWorkflowOpts(workflowName, desc, cronTriggers))
		require.NoError(t, err)
	}

	count := countCronRefsForWorkflowName(ctx, t, pool, workflowName)
	assert.Equal(t, 1, count, "should have exactly 1 cron ref regardless of how many versions were registered")
}

func TestOnlyDefaultCronsAreDeleted(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()
	repo := newWorkflowTestRepository(pool)

	const workflowName = "api-cron-method-test"

	// Register v1 with one DEFAULT cron
	v1, err := repo.PutWorkflowVersion(ctx, internalTenantId, minimalWorkflowOpts(workflowName, "v1", []string{"0 * * * *"}))
	require.NoError(t, err)

	// Insert an API-method cron directly against v1's WorkflowTriggers record
	queries := sqlcv1.New()
	var triggersId uuid.UUID
	err = pool.QueryRow(ctx,
		`SELECT "id" FROM "WorkflowTriggers" WHERE "workflowVersionId" = $1`,
		v1.WorkflowVersion.ID,
	).Scan(&triggersId)
	require.NoError(t, err)

	_, err = queries.CreateWorkflowTriggerCronRef(ctx, pool, sqlcv1.CreateWorkflowTriggerCronRefParams{
		Workflowtriggersid: triggersId,
		Crontrigger:        "15 * * * *",
		Method:             sqlcv1.NullWorkflowTriggerCronRefMethods{WorkflowTriggerCronRefMethods: sqlcv1.WorkflowTriggerCronRefMethodsAPI, Valid: true},
	})
	require.NoError(t, err)

	// Register v2 — this should delete v1's DEFAULT cron and migrate v1's API cron
	_, err = repo.PutWorkflowVersion(ctx, internalTenantId, minimalWorkflowOpts(workflowName, "v2", []string{"0 * * * *"}))
	require.NoError(t, err)

	// Only DEFAULT crons from the old version should be gone; API cron was migrated
	var defaultCount, apiCount int
	err = pool.QueryRow(ctx, `
		SELECT count(*) FILTER (WHERE c."method" = 'DEFAULT'),
		       count(*) FILTER (WHERE c."method" = 'API')
		FROM "WorkflowTriggerCronRef" c
		JOIN "WorkflowTriggers" tr ON tr."id" = c."parentId"
		JOIN "WorkflowVersion" wv ON wv."id" = tr."workflowVersionId"
		JOIN "Workflow" w ON w."id" = wv."workflowId"
		WHERE w."name" = $1
		  AND w."deletedAt" IS NULL
	`, workflowName).Scan(&defaultCount, &apiCount)
	require.NoError(t, err)

	assert.Equal(t, 1, defaultCount, "should have exactly 1 DEFAULT cron on the new version")
	assert.Equal(t, 1, apiCount, "API cron should be migrated to the new version, not deleted")
}
