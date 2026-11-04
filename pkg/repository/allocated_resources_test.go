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

	_ = pending
}
