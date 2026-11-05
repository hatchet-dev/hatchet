//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/repository/cache"
	"github.com/hatchet-dev/hatchet/pkg/repository/fairpool"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/validator"
)

func newAllocatedResourcesTestRepos(pool *pgxpool.Pool) (*workflowRepository, *workflowScheduleRepository, *webhookRepository) {
	logger := zerolog.Nop()
	shared := &sharedRepository{
		pool:       fairpool.Ungated(pool),
		ddlPool:    pool,
		l:          &logger,
		queries:    sqlcv1.New(),
		v:          validator.NewDefaultValidator(),
		queueCache: cache.New(5 * time.Minute),
	}
	return &workflowRepository{sharedRepository: shared},
		&workflowScheduleRepository{sharedRepository: shared},
		&webhookRepository{sharedRepository: shared}
}

func TestCountAllocatedResourcesByTenant(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()
	workflows, schedules, webhooks := newAllocatedResourcesTestRepos(pool)

	wf, err := workflows.PutWorkflowVersion(ctx, internalTenantId, minimalWorkflowOpts("allocated-resources-count", "v1", []string{"0 * * * *"}))
	require.NoError(t, err)

	apiCron, err := schedules.CreateCronWorkflow(ctx, internalTenantId, &CreateCronWorkflowTriggerOpts{
		WorkflowId: wf.WorkflowVersion.WorkflowId,
		Name:       "api-cron",
		Cron:       "15 * * * *",
	})
	require.NoError(t, err)

	disabled := false
	require.NoError(t, schedules.UpdateCronWorkflow(ctx, internalTenantId, apiCron.CronId, &UpdateCronOpts{Enabled: &disabled}))

	pending, err := schedules.CreateScheduledWorkflow(ctx, internalTenantId, &CreateScheduledWorkflowRunForWorkflowOpts{
		WorkflowId:       wf.WorkflowVersion.WorkflowId,
		ScheduledTrigger: time.Now().UTC().Add(time.Hour),
	})
	require.NoError(t, err)

	fired, err := schedules.CreateScheduledWorkflow(ctx, internalTenantId, &CreateScheduledWorkflowRunForWorkflowOpts{
		WorkflowId:       wf.WorkflowVersion.WorkflowId,
		ScheduledTrigger: time.Now().UTC().Add(2 * time.Hour),
	})
	require.NoError(t, err)

	runID := uuid.New()
	_, err = pool.Exec(ctx, `
		INSERT INTO "WorkflowRun" ("id", "tenantId", "workflowVersionId")
		VALUES ($1, $2, $3)
	`, runID, internalTenantId, wf.WorkflowVersion.ID)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `
		INSERT INTO "WorkflowRunTriggeredBy" ("id", "tenantId", "scheduledId", "parentId")
		VALUES ($1, $2, $3, $4)
	`, uuid.New(), internalTenantId, fired.ID, runID)
	require.NoError(t, err)

	_, err = webhooks.CreateWebhook(ctx, internalTenantId, CreateWebhookOpts{
		Name:               "allocated-webhook",
		Sourcename:         sqlcv1.V1IncomingWebhookSourceNameGENERIC,
		Eventkeyexpression: "input.key",
		AuthConfig: AuthConfig{
			Type: sqlcv1.V1IncomingWebhookAuthTypeBASIC,
			BasicAuth: &BasicAuthCredentials{
				Username:          "user",
				EncryptedPassword: []byte("secret"),
			},
		},
	})
	require.NoError(t, err)

	rows, err := schedules.CountAllocatedResourcesByTenant(ctx, []uuid.UUID{internalTenantId})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, internalTenantId, rows[0].TenantID)
	require.Equal(t, int64(1), rows[0].CronCount, "enabled DEFAULT cron counted; disabled API cron excluded")
	require.Equal(t, int64(1), rows[0].ScheduledRunCount, "pending schedule counted; fired schedule excluded")
	require.Equal(t, int64(1), rows[0].WebhookCount)

	require.NoError(t, schedules.DeleteScheduledWorkflow(ctx, internalTenantId, fired.ID))
	rows, err = schedules.CountAllocatedResourcesByTenant(ctx, []uuid.UUID{internalTenantId})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, int64(1), rows[0].ScheduledRunCount, "deleting an already-fired schedule leaves the pending one")

	require.NoError(t, schedules.DeleteScheduledWorkflow(ctx, internalTenantId, pending.ID))
	rows, err = schedules.CountAllocatedResourcesByTenant(ctx, []uuid.UUID{internalTenantId})
	require.NoError(t, err)
	require.Equal(t, int64(0), rows[0].ScheduledRunCount, "deleting the pending schedule drops it from the count")
}

func TestAllocatedCronCountSurvivesReregistration(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()
	workflows, schedules, _ := newAllocatedResourcesTestRepos(pool)

	_, err := workflows.PutWorkflowVersion(ctx, internalTenantId, minimalWorkflowOpts("allocated-reregister", "v1", []string{"0 * * * *"}))
	require.NoError(t, err)

	wf, err := workflows.GetWorkflowByName(ctx, internalTenantId, "allocated-reregister")
	require.NoError(t, err)

	_, err = schedules.CreateCronWorkflow(ctx, internalTenantId, &CreateCronWorkflowTriggerOpts{
		WorkflowId: wf.ID,
		Name:       "api-cron",
		Cron:       "15 * * * *",
	})
	require.NoError(t, err)

	_, err = workflows.PutWorkflowVersion(ctx, internalTenantId, minimalWorkflowOpts("allocated-reregister", "v2", []string{"0 * * * *"}))
	require.NoError(t, err)

	rows, err := schedules.CountAllocatedResourcesByTenant(ctx, []uuid.UUID{internalTenantId})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, int64(2), rows[0].CronCount, "the moved API cron and the new DEFAULT cron are each counted once")
}

func TestDeleteInvalidCronDropsAllocatedCount(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()
	workflows, schedules, _ := newAllocatedResourcesTestRepos(pool)

	wf, err := workflows.PutWorkflowVersion(ctx, internalTenantId, minimalWorkflowOpts("allocated-invalid-cron", "v1", []string{"0 * * * *"}))
	require.NoError(t, err)

	apiCron, err := schedules.CreateCronWorkflow(ctx, internalTenantId, &CreateCronWorkflowTriggerOpts{
		WorkflowId: wf.WorkflowVersion.WorkflowId,
		Name:       "api-cron",
		Cron:       "15 * * * *",
	})
	require.NoError(t, err)

	require.NoError(t, schedules.DeleteInvalidCron(ctx, apiCron.CronId))

	rows, err := schedules.CountAllocatedResourcesByTenant(ctx, []uuid.UUID{internalTenantId})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, int64(1), rows[0].CronCount, "deleting the invalid API cron leaves the DEFAULT cron")
}

func TestDeleteWorkflowDropsAllocatedCounts(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()
	workflows, schedules, _ := newAllocatedResourcesTestRepos(pool)

	wf, err := workflows.PutWorkflowVersion(ctx, internalTenantId, minimalWorkflowOpts("allocated-delete-workflow", "v1", []string{"0 * * * *"}))
	require.NoError(t, err)

	kept, err := workflows.PutWorkflowVersion(ctx, internalTenantId, minimalWorkflowOpts("allocated-delete-workflow-kept", "v1", []string{"30 * * * *"}))
	require.NoError(t, err)

	_, err = schedules.CreateScheduledWorkflow(ctx, internalTenantId, &CreateScheduledWorkflowRunForWorkflowOpts{
		WorkflowId:       kept.WorkflowVersion.WorkflowId,
		ScheduledTrigger: time.Now().UTC().Add(3 * time.Hour),
	})
	require.NoError(t, err)

	pending, err := schedules.CreateScheduledWorkflow(ctx, internalTenantId, &CreateScheduledWorkflowRunForWorkflowOpts{
		WorkflowId:       wf.WorkflowVersion.WorkflowId,
		ScheduledTrigger: time.Now().UTC().Add(time.Hour),
	})
	require.NoError(t, err)

	fired, err := schedules.CreateScheduledWorkflow(ctx, internalTenantId, &CreateScheduledWorkflowRunForWorkflowOpts{
		WorkflowId:       wf.WorkflowVersion.WorkflowId,
		ScheduledTrigger: time.Now().UTC().Add(2 * time.Hour),
	})
	require.NoError(t, err)

	runID := uuid.New()
	_, err = pool.Exec(ctx, `
		INSERT INTO "WorkflowRun" ("id", "tenantId", "workflowVersionId")
		VALUES ($1, $2, $3)
	`, runID, internalTenantId, wf.WorkflowVersion.ID)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `
		INSERT INTO "WorkflowRunTriggeredBy" ("id", "tenantId", "scheduledId", "parentId")
		VALUES ($1, $2, $3, $4)
	`, uuid.New(), internalTenantId, fired.ID, runID)
	require.NoError(t, err)

	_, err = workflows.DeleteWorkflow(ctx, internalTenantId, wf.WorkflowVersion.WorkflowId)
	require.NoError(t, err)

	rows, err := schedules.CountAllocatedResourcesByTenant(ctx, []uuid.UUID{internalTenantId})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, int64(1), rows[0].CronCount, "the deleted workflow's cron is gone and the other workflow's cron remains")
	require.Equal(t, int64(1), rows[0].ScheduledRunCount, "the deleted workflow's pending schedule is gone and the other schedule remains")

	_, err = pool.Exec(ctx, `
		INSERT INTO "WorkflowRunTriggeredBy" ("id", "tenantId", "scheduledId", "parentId")
		VALUES ($1, $2, $3, $4)
	`, uuid.New(), internalTenantId, pending.ID, runID)
	require.NoError(t, err)

	rows, err = schedules.CountAllocatedResourcesByTenant(ctx, []uuid.UUID{internalTenantId})
	require.NoError(t, err)
	require.Equal(t, int64(1), rows[0].ScheduledRunCount, "a fire after the workflow delete does not decrement the remaining schedule")
}

func TestRebuildAllocatedResourceCounts(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()
	workflows, schedules, _ := newAllocatedResourcesTestRepos(pool)

	_, err := workflows.PutWorkflowVersion(ctx, internalTenantId, minimalWorkflowOpts("allocated-rebuild", "v1", []string{"0 * * * *"}))
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `
		UPDATE tenant_allocated_resource_count
		SET cron_count = 40
		WHERE tenant_id = $1
	`, internalTenantId)
	require.NoError(t, err)

	rows, err := schedules.CountAllocatedResourcesByTenant(ctx, nil)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, int64(1), rows[0].CronCount, "the unfiltered read replaces the stored count from live rows")
}
