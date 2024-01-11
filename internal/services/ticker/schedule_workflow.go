package ticker

import (
	"context"
	"fmt"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/hatchet-dev/hatchet/internal/repository"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes"
	"github.com/hatchet-dev/hatchet/internal/taskqueue"
)

func (t *TickerImpl) handleScheduleWorkflow(ctx context.Context, task *taskqueue.Task) error {
	t.l.Debug().Msg("ticker: scheduling workflow")

	payload := tasktypes.ScheduleWorkflowTaskPayload{}
	metadata := tasktypes.ScheduleWorkflowTaskMetadata{}

	err := t.dv.DecodeAndValidate(task.Payload, &payload)

	if err != nil {
		return fmt.Errorf("could not decode ticker task payload: %w", err)
	}

	err = t.dv.DecodeAndValidate(task.Metadata, &metadata)

	if err != nil {
		return fmt.Errorf("could not decode ticker task metadata: %w", err)
	}

	workflowVersion, err := t.repo.Workflow().GetWorkflowVersionById(metadata.TenantId, payload.WorkflowVersionId)

	if err != nil {
		return fmt.Errorf("could not get workflow version: %w", err)
	}

	// create a new scheduler
	s, err := gocron.NewScheduler(gocron.WithLocation(time.UTC))

	if err != nil {
		return fmt.Errorf("could not create scheduler: %w", err)
	}

	// parse trigger time
	triggerAt, err := time.Parse(time.RFC3339, payload.TriggerAt)

	if err != nil {
		return fmt.Errorf("could not parse started at: %w", err)
	}

	// schedule the workflow
	_, err = t.s.NewJob(
		gocron.OneTimeJob(
			gocron.OneTimeJobStartDateTime(triggerAt),
		),
		gocron.NewTask(
			t.runScheduledWorkflow(ctx, metadata.TenantId, payload.ScheduledWorkflowId, workflowVersion),
		),
	)

	if err != nil {
		return fmt.Errorf("could not schedule workflow: %w", err)
	}

	// store the schedule in the cron map
	t.scheduledWorkflows.Store(getScheduledWorkflowKey(workflowVersion.ID, payload.ScheduledWorkflowId), s)

	s.Start()

	return nil
}

func (t *TickerImpl) runScheduledWorkflow(ctx context.Context, tenantId, scheduledWorkflowId string, workflowVersion *db.WorkflowVersionModel) func() {
	return func() {
		t.l.Debug().Msgf("ticker: running workflow %s", workflowVersion.ID)

		// create a new workflow run in the database
		createOpts, err := repository.GetCreateWorkflowRunOptsFromSchedule(scheduledWorkflowId, workflowVersion)

		if err != nil {
			t.l.Err(err).Msg("could not get create workflow run opts")
			return
		}

		workflowRun, err := t.repo.WorkflowRun().CreateNewWorkflowRun(ctx, tenantId, createOpts)

		if err != nil {
			t.l.Err(err).Msg("could not create workflow run")
			return
		}

		for _, jobRun := range workflowRun.JobRuns() {
			jobRunCp := jobRun
			err = t.tq.AddTask(
				context.Background(),
				taskqueue.JOB_PROCESSING_QUEUE,
				tasktypes.JobRunQueuedToTask(jobRun.Job(), &jobRunCp),
			)

			if err != nil {
				t.l.Err(err).Msg("could not add job run queued task")
				continue
			}
		}

		// get the scheduler
		schedulerVal, ok := t.scheduledWorkflows.Load(getScheduledWorkflowKey(workflowVersion.ID, scheduledWorkflowId))

		if !ok {
			// return fmt.Errorf("could not find scheduled workflow %s", payload.WorkflowVersionId)
			t.l.Error().Msgf("could not find scheduled workflow %s", workflowVersion.ID)
			return
		}

		defer t.scheduledWorkflows.Delete(workflowVersion.ID)

		scheduler := schedulerVal.(gocron.Scheduler)

		// cancel the schedule
		err = scheduler.Shutdown()

		if err != nil {
			t.l.Err(err).Msg("could not cancel scheduler")
			return
		}
	}
}

func (t *TickerImpl) handleCancelWorkflow(ctx context.Context, task *taskqueue.Task) error {
	t.l.Debug().Msg("ticker: canceling scheduled workflow")

	payload := tasktypes.CancelWorkflowTaskPayload{}
	metadata := tasktypes.CancelWorkflowTaskMetadata{}

	err := t.dv.DecodeAndValidate(task.Payload, &payload)

	if err != nil {
		return fmt.Errorf("could not decode ticker task payload: %w", err)
	}

	err = t.dv.DecodeAndValidate(task.Metadata, &metadata)

	if err != nil {
		return fmt.Errorf("could not decode ticker task metadata: %w", err)
	}

	// get the scheduler
	schedulerVal, ok := t.scheduledWorkflows.Load(getScheduledWorkflowKey(payload.WorkflowVersionId, payload.ScheduledWorkflowId))

	if !ok {
		return fmt.Errorf("could not find scheduled workflow %s", payload.WorkflowVersionId)
	}

	defer t.scheduledWorkflows.Delete(getScheduledWorkflowKey(payload.WorkflowVersionId, payload.ScheduledWorkflowId))

	schedulerP := schedulerVal.(*gocron.Scheduler)
	scheduler := *schedulerP

	// cancel the schedule
	return scheduler.Shutdown()
}

func getScheduledWorkflowKey(workflowVersionId, scheduledWorkflowId string) string {
	return fmt.Sprintf("%s-%s", workflowVersionId, scheduledWorkflowId)
}
