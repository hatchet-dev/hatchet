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

func (t *TickerImpl) runPollSchedules(ctx context.Context) func() {
	return func() {
		t.l.Debug().Msgf("ticker: polling workflow schedules")

		scheduledWorkflows, err := t.repo.Ticker().PollScheduledWorkflows(t.tickerId)

		if err != nil {
			t.l.Err(err).Msg("could not poll workflow schedules")
			return
		}

		existingSchedules := make(map[string]bool)

		t.scheduledWorkflows.Range(func(key, value interface{}) bool {
			existingSchedules[key.(string)] = false
			return true
		})

		for _, scheduledWorkflow := range scheduledWorkflows {
			workflowVersionId := sqlchelpers.UUIDToStr(scheduledWorkflow.WorkflowVersionId)
			scheduledWorkflowId := sqlchelpers.UUIDToStr(scheduledWorkflow.ID)

			t.l.Debug().Msgf("ticker: handling scheduled workflow %s for version %s", scheduledWorkflowId, workflowVersionId)

			// if the cron is already scheduled, mark it as existing
			if _, ok := existingSchedules[getScheduledWorkflowKey(workflowVersionId, scheduledWorkflowId)]; ok {
				existingSchedules[getScheduledWorkflowKey(workflowVersionId, scheduledWorkflowId)] = true
				continue
			}

			// if the cron is not scheduled, schedule it
			if err := t.handleScheduleWorkflow(ctx, scheduledWorkflow); err != nil {
				t.l.Err(err).Msg("could not schedule cron")
			}

			existingSchedules[getScheduledWorkflowKey(workflowVersionId, scheduledWorkflowId)] = true
		}

		// cancel any crons that are no longer assigned to this ticker
		for key, exists := range existingSchedules {
			if !exists {
				if err := t.handleCancelWorkflow(ctx, key); err != nil {
					t.l.Err(err).Msg("could not cancel workflow")
				}
			}
		}
	}
}

func (t *TickerImpl) handleScheduleWorkflow(ctx context.Context, scheduledWorkflow *dbsqlc.PollScheduledWorkflowsRow) error {
	t.l.Debug().Msg("ticker: scheduling workflow")

	// create a new scheduler
	s, err := gocron.NewScheduler(gocron.WithLocation(time.UTC))

	if err != nil {
		return fmt.Errorf("could not create scheduler: %w", err)
	}

	// parse trigger time
	tenantId := sqlchelpers.UUIDToStr(scheduledWorkflow.TenantId)
	workflowVersionId := sqlchelpers.UUIDToStr(scheduledWorkflow.WorkflowVersionId)
	scheduledWorkflowId := sqlchelpers.UUIDToStr(scheduledWorkflow.ID)
	triggerAt := scheduledWorkflow.TriggerAt.Time

	// if start is in the past, run now
	if triggerAt.Before(time.Now()) {
		t.l.Debug().Msg("ticker: trigger time is in the past, running now")

		t.runScheduledWorkflow(ctx, tenantId, workflowVersionId, scheduledWorkflowId, scheduledWorkflow.Input)()
		return nil
	}

	// schedule the workflow
	_, err = t.s.NewJob(
		gocron.OneTimeJob(
			gocron.OneTimeJobStartDateTime(triggerAt),
		),
		gocron.NewTask(
			t.runScheduledWorkflow(ctx, tenantId, workflowVersionId, scheduledWorkflowId, scheduledWorkflow.Input),
		),
	)

	if err != nil {
		return fmt.Errorf("could not schedule workflow: %w", err)
	}

	// store the schedule in the cron map
	t.scheduledWorkflows.Store(getScheduledWorkflowKey(workflowVersionId, scheduledWorkflowId), s)

	s.Start()

	return nil
}

func (t *TickerImpl) runScheduledWorkflow(ctx context.Context, tenantId, workflowVersionId, scheduledWorkflowId string, input []byte) func() {
	return func() {
		t.l.Debug().Msgf("ticker: running workflow %s", workflowVersionId)

		workflowVersion, err := t.repo.Workflow().GetWorkflowVersionById(tenantId, workflowVersionId)

		if err != nil {
			t.l.Err(err).Msg("could not get workflow version")
			return
		}

		// create a new workflow run in the database
		createOpts, err := repository.GetCreateWorkflowRunOptsFromSchedule(scheduledWorkflowId, workflowVersion, input)

		if err != nil {
			t.l.Err(err).Msg("could not get create workflow run opts")
			return
		}

		workflowRunId, err := t.repo.WorkflowRun().CreateNewWorkflowRun(ctx, tenantId, createOpts)

		if err != nil {
			t.l.Err(err).Msg("could not create workflow run")
			return
		}

		jobRuns, err := t.repo.JobRun().ListJobRunsForWorkflowRun(tenantId, workflowRunId)

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

		// get the scheduler
		schedulerVal, ok := t.scheduledWorkflows.Load(getScheduledWorkflowKey(workflowVersionId, scheduledWorkflowId))

		if !ok {
			t.l.Error().Msgf("could not find scheduled workflow %s", workflowVersionId)
			return
		}

		defer t.scheduledWorkflows.Delete(workflowVersionId)

		scheduler := schedulerVal.(gocron.Scheduler)

		// cancel the schedule
		err = scheduler.Shutdown()

		if err != nil {
			t.l.Err(err).Msg("could not cancel scheduler")
			return
		}
	}
}

func (t *TickerImpl) handleCancelWorkflow(ctx context.Context, key string) error {
	t.l.Debug().Msg("ticker: canceling scheduled workflow")

	// get the scheduler
	schedulerVal, ok := t.scheduledWorkflows.Load(key)

	if !ok {
		return fmt.Errorf("could not find scheduled workflow with key %s", key)
	}

	defer t.scheduledWorkflows.Delete(key)

	scheduler := schedulerVal.(gocron.Scheduler)

	// cancel the schedule
	return scheduler.Shutdown()
}

func getScheduledWorkflowKey(workflowVersionId, scheduledWorkflowId string) string {
	return fmt.Sprintf("%s-%s", workflowVersionId, scheduledWorkflowId)
}
