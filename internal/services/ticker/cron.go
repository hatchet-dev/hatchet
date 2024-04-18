package ticker

import (
	"context"
	"fmt"
	"time"

	"github.com/go-co-op/gocron/v2"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/internal/repository"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/sqlchelpers"
	"github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes"
)

func (t *TickerImpl) runPollCronSchedules(ctx context.Context) func() {
	return func() {
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		t.l.Debug().Msgf("ticker: polling cron schedules")

		crons, err := t.repo.Ticker().PollCronSchedules(ctx, t.tickerId)

		if err != nil {
			t.l.Err(err).Msg("could not poll cron schedules")
			return
		}

		existingCrons := make(map[string]bool)

		t.crons.Range(func(key, value interface{}) bool {
			existingCrons[key.(string)] = false
			return true
		})

		for _, cron := range crons {
			workflowVersionId := sqlchelpers.UUIDToStr(cron.WorkflowVersionId)

			t.l.Debug().Msgf("ticker: handling cron %s for version %s", cron.Cron, workflowVersionId)

			// if the cron is already scheduled, mark it as existing
			if _, ok := existingCrons[getCronKey(workflowVersionId, cron.Cron)]; ok {
				existingCrons[getCronKey(workflowVersionId, cron.Cron)] = true
				continue
			}

			// if the cron is not scheduled, schedule it
			if err := t.handleScheduleCron(ctx, cron); err != nil {
				t.l.Err(err).Msg("could not schedule cron")
			}

			existingCrons[getCronKey(workflowVersionId, cron.Cron)] = true
		}

		// cancel any crons that are no longer assigned to this ticker
		for key, exists := range existingCrons {
			if !exists {
				if err := t.handleCancelCron(ctx, key); err != nil {
					t.l.Err(err).Msg("could not cancel cron")
				}
			}
		}
	}
}

func (t *TickerImpl) handleScheduleCron(ctx context.Context, cron *dbsqlc.PollCronSchedulesRow) error {
	t.l.Debug().Msg("ticker: scheduling cron")

	// create a new scheduler
	s, err := gocron.NewScheduler(gocron.WithLocation(time.UTC))

	if err != nil {
		return fmt.Errorf("could not create scheduler: %w", err)
	}

	tenantId := sqlchelpers.UUIDToStr(cron.TenantId)
	workflowVersionId := sqlchelpers.UUIDToStr(cron.WorkflowVersionId)
	cronParentId := sqlchelpers.UUIDToStr(cron.ParentId)

	// schedule the cron
	_, err = t.s.NewJob(
		gocron.CronJob(cron.Cron, false),
		gocron.NewTask(
			t.runCronWorkflow(ctx, tenantId, workflowVersionId, cron.Cron, cronParentId, cron.Input),
		),
	)

	if err != nil {
		return fmt.Errorf("could not schedule cron: %w", err)
	}

	// store the schedule in the cron map
	t.crons.Store(getCronKey(workflowVersionId, cron.Cron), s)

	s.Start()

	return nil
}

func (t *TickerImpl) runCronWorkflow(ctx context.Context, tenantId, workflowVersionId, cron, cronParentId string, input []byte) func() {
	return func() {
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		t.l.Debug().Msgf("ticker: running workflow %s", workflowVersionId)

		workflowVersion, err := t.repo.Workflow().GetWorkflowVersionById(ctx, tenantId, workflowVersionId)

		if err != nil {
			t.l.Err(err).Msg("could not get workflow version")
			return
		}

		// create a new workflow run in the database
		createOpts, err := repository.GetCreateWorkflowRunOptsFromCron(cron, cronParentId, workflowVersion, input)

		if err != nil {
			t.l.Err(err).Msg("could not get create workflow run opts")
			return
		}

		workflowRunId, err := t.repo.WorkflowRun().CreateNewWorkflowRun(ctx, tenantId, createOpts)

		if err != nil {
			t.l.Err(err).Msg("could not create workflow run")
			return
		}

		jobRuns, err := t.repo.JobRun().ListJobRunsForWorkflowRun(ctx, tenantId, workflowRunId)

		if err != nil {
			t.l.Err(err).Msg("could not list job runs for workflow run")
			return
		}

		for _, jobRunId := range jobRuns {
			jobRunStr := sqlchelpers.UUIDToStr(jobRunId)

			err = t.mq.AddMessage(
				context.Background(),
				msgqueue.JOB_PROCESSING_QUEUE,
				tasktypes.JobRunQueuedToTask(tenantId, jobRunStr),
			)

			if err != nil {
				t.l.Err(err).Msg("could not add job run queued task")
				continue
			}
		}
	}
}

func (t *TickerImpl) handleCancelCron(ctx context.Context, key string) error {
	t.l.Debug().Msg("ticker: canceling cron")

	// get the scheduler
	schedulerVal, ok := t.crons.Load(key)

	if !ok {
		return fmt.Errorf("could not find cron with key %s ", key)
	}

	defer t.crons.Delete(key)

	scheduler := schedulerVal.(gocron.Scheduler)

	// cancel the cron
	if err := scheduler.Shutdown(); err != nil {
		return fmt.Errorf("could not cancel cron: %w", err)
	}

	return nil
}

func getCronKey(workflowVersionId, schedule string) string {
	return fmt.Sprintf("%s-%s", workflowVersionId, schedule)
}
