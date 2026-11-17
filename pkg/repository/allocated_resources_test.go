//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"

	"github.com/hatchet-dev/hatchet/pkg/enums"
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

		eventKeyWorkflowsCache: expirable.NewLRU(10000, func(key eventKeyTenantIdTuple, value []*sqlcv1.ListWorkflowsForEventsRow) {}, 5*time.Second),
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
		Sourcename:         enums.V1IncomingWebhookSourceNameGENERIC,
		Eventkeyexpression: "input.key",
		AuthConfig: AuthConfig{
			Type: enums.V1IncomingWebhookAuthTypeBASIC,
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

	lateRunID := uuid.New()
	_, err = pool.Exec(ctx, `
		INSERT INTO "WorkflowRun" ("id", "tenantId", "workflowVersionId")
		VALUES ($1, $2, $3)
	`, lateRunID, internalTenantId, wf.WorkflowVersion.ID)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `
		INSERT INTO "WorkflowRunTriggeredBy" ("id", "tenantId", "scheduledId", "parentId")
		VALUES ($1, $2, $3, $4)
	`, uuid.New(), internalTenantId, pending.ID, lateRunID)
	require.NoError(t, err)

	rows, err = schedules.CountAllocatedResourcesByTenant(ctx, []uuid.UUID{internalTenantId})
	require.NoError(t, err)
	require.Equal(t, int64(1), rows[0].ScheduledRunCount, "a fire after the workflow delete does not decrement the remaining schedule")
}

func TestCorrectAllocatedResourceCounts(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()
	workflows, schedules, _ := newAllocatedResourcesTestRepos(pool)

	_, err := workflows.PutWorkflowVersion(ctx, internalTenantId, minimalWorkflowOpts("allocated-correct", "v1", []string{"0 * * * *"}))
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `
		INSERT INTO tenant_allocated_resource_count (tenant_id, cron_count, scheduled_run_count)
		VALUES ($1, 39, -3)
	`, internalTenantId)
	require.NoError(t, err)

	rows, err := schedules.CountAllocatedResourcesByTenant(ctx, []uuid.UUID{internalTenantId})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, int64(40), rows[0].CronCount, "the filtered read returns the stored count")
	require.Equal(t, int64(0), rows[0].ScheduledRunCount, "a stored count below zero reads as zero")

	rows, err = schedules.CountAllocatedResourcesByTenant(ctx, nil)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, int64(1), rows[0].CronCount, "the unfiltered read corrects the stored count to the live rows")

	deltasAfterFirstCorrection := countAllocatedResourceDeltas(ctx, t, pool)

	_, err = schedules.CountAllocatedResourcesByTenant(ctx, nil)
	require.NoError(t, err)
	require.Equal(t, deltasAfterFirstCorrection, countAllocatedResourceDeltas(ctx, t, pool), "a correct stored count gets no further correction")

	require.NoError(t, schedules.CompactAllocatedResourceCounts(ctx))

	var cronCount, scheduledRunCount int64
	require.NoError(t, pool.QueryRow(ctx, `
		SELECT cron_count, scheduled_run_count
		FROM tenant_allocated_resource_count
		WHERE tenant_id = $1
	`, internalTenantId).Scan(&cronCount, &scheduledRunCount))
	require.Equal(t, int64(1), cronCount)
	require.Equal(t, int64(0), scheduledRunCount)
}

func TestCompactAllocatedResourceCounts(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()
	workflows, schedules, _ := newAllocatedResourcesTestRepos(pool)

	wf, err := workflows.PutWorkflowVersion(ctx, internalTenantId, minimalWorkflowOpts("allocated-compact", "v1", []string{"0 * * * *"}))
	require.NoError(t, err)

	created := make([]uuid.UUID, 0, 3)

	for i := range 3 {
		schedule, err := schedules.CreateScheduledWorkflow(ctx, internalTenantId, &CreateScheduledWorkflowRunForWorkflowOpts{
			WorkflowId:       wf.WorkflowVersion.WorkflowId,
			ScheduledTrigger: time.Now().UTC().Add(time.Duration(i+1) * time.Hour),
		})
		require.NoError(t, err)

		created = append(created, schedule.ID)
	}

	require.NoError(t, schedules.DeleteScheduledWorkflow(ctx, internalTenantId, created[0]))
	require.Equal(t, 5, countAllocatedResourceDeltas(ctx, t, pool), "each write appends one delta")

	before, err := schedules.CountAllocatedResourcesByTenant(ctx, []uuid.UUID{internalTenantId})
	require.NoError(t, err)
	require.Len(t, before, 1)
	require.Equal(t, int64(1), before[0].CronCount)
	require.Equal(t, int64(2), before[0].ScheduledRunCount)

	require.NoError(t, schedules.CompactAllocatedResourceCounts(ctx))
	require.Equal(t, 0, countAllocatedResourceDeltas(ctx, t, pool), "compaction folds every delta")

	after, err := schedules.CountAllocatedResourcesByTenant(ctx, []uuid.UUID{internalTenantId})
	require.NoError(t, err)
	require.Equal(t, before, after, "folding deltas into the count row leaves the count unchanged")

	require.NoError(t, schedules.DeleteScheduledWorkflow(ctx, internalTenantId, created[1]))

	after, err = schedules.CountAllocatedResourcesByTenant(ctx, []uuid.UUID{internalTenantId})
	require.NoError(t, err)
	require.Equal(t, int64(1), after[0].ScheduledRunCount, "a delta written after compaction adds to the folded count")
}

// The stored schedule count equals the pending schedules on live workflows while schedule
// creates and deletes, a workflow delete, version registrations, and compaction race each
// other. The second pass also races corrections, which must not change a count that is right.
func TestAllocatedScheduleCountMatchesLiveRowsUnderConcurrency(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()
	workflows, schedules, _ := newAllocatedResourcesTestRepos(pool)

	for _, withCorrections := range []bool{false, true} {
		t.Run(fmt.Sprintf("corrections=%t", withCorrections), func(t *testing.T) {
			keptName := fmt.Sprintf("allocated-concurrent-kept-%t", withCorrections)

			kept, err := workflows.PutWorkflowVersion(ctx, internalTenantId, minimalWorkflowOpts(keptName, "v0", nil))
			require.NoError(t, err)

			deleted, err := workflows.PutWorkflowVersion(ctx, internalTenantId, minimalWorkflowOpts(fmt.Sprintf("allocated-concurrent-deleted-%t", withCorrections), "v0", nil))
			require.NoError(t, err)

			backgroundCtx, stopBackground := context.WithCancel(ctx)
			defer stopBackground()

			var background sync.WaitGroup

			runInBackground := func(f func() error) {
				background.Add(1)
				go func() {
					defer background.Done()
					for backgroundCtx.Err() == nil {
						if err := f(); err != nil && backgroundCtx.Err() == nil {
							t.Errorf("background work failed: %v", err)
							return
						}
					}
				}()
			}

			runInBackground(func() error {
				return schedules.CompactAllocatedResourceCounts(backgroundCtx)
			})

			if withCorrections {
				runInBackground(func() error {
					_, err := schedules.CountAllocatedResourcesByTenant(backgroundCtx, nil)
					return err
				})
			}

			var writers errgroup.Group

			for range 8 {
				writers.Go(func() error {
					for i := range 25 {
						schedule, err := schedules.CreateScheduledWorkflow(ctx, internalTenantId, &CreateScheduledWorkflowRunForWorkflowOpts{
							WorkflowId:       kept.WorkflowVersion.WorkflowId,
							ScheduledTrigger: time.Now().UTC().Add(time.Hour),
						})
						if err != nil {
							return fmt.Errorf("creating a schedule: %w", err)
						}

						if i%2 == 0 {
							if err := schedules.DeleteScheduledWorkflow(ctx, internalTenantId, schedule.ID); err != nil {
								return fmt.Errorf("deleting a schedule: %w", err)
							}
						}

						_, err = schedules.CreateScheduledWorkflow(ctx, internalTenantId, &CreateScheduledWorkflowRunForWorkflowOpts{
							WorkflowId:       deleted.WorkflowVersion.WorkflowId,
							ScheduledTrigger: time.Now().UTC().Add(time.Hour),
						})
						if err != nil && !errors.Is(err, errNoLiveWorkflowVersion) {
							return fmt.Errorf("creating a schedule on the workflow being deleted: %w", err)
						}
					}

					return nil
				})
			}

			writers.Go(func() error {
				for i := range 5 {
					if _, err := workflows.PutWorkflowVersion(ctx, internalTenantId, minimalWorkflowOpts(keptName, fmt.Sprintf("v%d", i+1), nil)); err != nil {
						return fmt.Errorf("registering a version: %w", err)
					}
				}

				return nil
			})

			writers.Go(func() error {
				time.Sleep(100 * time.Millisecond)

				if _, err := workflows.DeleteWorkflow(ctx, internalTenantId, deleted.WorkflowVersion.WorkflowId); err != nil {
					return fmt.Errorf("deleting the workflow: %w", err)
				}

				return nil
			})

			writesDone := make(chan error, 1)

			go func() {
				writesDone <- writers.Wait()
			}()

			checks := 0

			for writing := true; writing; checks++ {
				select {
				case err := <-writesDone:
					require.NoError(t, err)
					writing = false
				default:
				}

				requireScheduleCountMatchesLiveRows(ctx, t, pool, schedules)
			}

			t.Logf("compared the stored schedule count with the live rows %d times during writes", checks)

			stopBackground()
			background.Wait()

			require.NoError(t, schedules.CompactAllocatedResourceCounts(ctx))
			require.Equal(t, 0, countAllocatedResourceDeltas(ctx, t, pool))
			requireScheduleCountMatchesLiveRows(ctx, t, pool, schedules)
		})
	}
}

func countAllocatedResourceDeltas(ctx context.Context, t *testing.T, pool *pgxpool.Pool) int {
	t.Helper()

	var count int
	require.NoError(t, pool.QueryRow(ctx, `SELECT COUNT(*) FROM tenant_allocated_resource_count_delta`).Scan(&count))

	return count
}

// requireScheduleCountMatchesLiveRows reads the stored count and the live rows in one snapshot,
// so writes that commit between the two reads do not show up as a difference.
func requireScheduleCountMatchesLiveRows(ctx context.Context, t *testing.T, pool *pgxpool.Pool, schedules *workflowScheduleRepository) {
	t.Helper()

	tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	require.NoError(t, err)

	defer func() { _ = tx.Rollback(ctx) }()

	rows, err := schedules.queries.CountAllocatedResourcesByTenant(ctx, tx, []uuid.UUID{internalTenantId})
	require.NoError(t, err)

	var stored int64

	if len(rows) > 0 {
		stored = rows[0].ScheduledRunCount
	}

	var live int64
	require.NoError(t, tx.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM "WorkflowTriggerScheduledRef" s
		JOIN "WorkflowVersion" v ON s."parentId" = v."id"
		JOIN "Workflow" w ON v."workflowId" = w."id"
		WHERE
			w."tenantId" = $1
			AND s."deletedAt" IS NULL
			AND v."deletedAt" IS NULL
			AND w."deletedAt" IS NULL
	`, internalTenantId).Scan(&live))

	require.Equal(t, live, stored, "the stored schedule count differs from the pending schedules on live workflows")
}
