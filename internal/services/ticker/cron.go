package ticker

import (
	"context"
	"fmt"
	"time"

	"github.com/go-co-op/gocron"
	"github.com/hatchet-dev/hatchet/internal/repository"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes"
	"github.com/hatchet-dev/hatchet/internal/taskqueue"
)

func (t *TickerImpl) handleScheduleCron(ctx context.Context, task *taskqueue.Task) error {
	t.l.Debug().Msg("ticker: scheduling cron")

	payload := tasktypes.ScheduleCronTaskPayload{}
	metadata := tasktypes.ScheduleCronTaskMetadata{}

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
	s := gocron.NewScheduler(time.UTC)

	// schedule the cron
	_, err = s.Cron(payload.Cron).Do(t.runWorkflow(ctx, metadata.TenantId, &payload, workflowVersion))

	if err != nil {
		return fmt.Errorf("could not schedule cron: %w", err)
	}

	// store the schedule in the cron map
	t.crons.Store(payload.WorkflowVersionId, s)

	s.StartAsync()

	return nil
}

func (t *TickerImpl) runWorkflow(ctx context.Context, tenantId string, payload *tasktypes.ScheduleCronTaskPayload, workflowVersion *db.WorkflowVersionModel) func() {
	return func() {
		t.l.Debug().Msgf("ticker: running workflow %s", payload.WorkflowVersionId)

		// create a new workflow run in the database
		createOpts, err := repository.GetCreateWorkflowRunOptsFromCron(payload.Cron, payload.CronParentId, workflowVersion)

		if err != nil {
			t.l.Err(err).Msg("could not get create workflow run opts")
			return
		}

		workflowRun, err := t.repo.WorkflowRun().CreateNewWorkflowRun(tenantId, createOpts)

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
	}
}

func (t *TickerImpl) handleCancelCron(ctx context.Context, task *taskqueue.Task) error {
	t.l.Debug().Msg("ticker: canceling cron")

	payload := tasktypes.CancelCronTaskPayload{}
	metadata := tasktypes.CancelCronTaskMetadata{}

	err := t.dv.DecodeAndValidate(task.Payload, &payload)

	if err != nil {
		return fmt.Errorf("could not decode ticker task payload: %w", err)
	}

	err = t.dv.DecodeAndValidate(task.Metadata, &metadata)

	if err != nil {
		return fmt.Errorf("could not decode ticker task metadata: %w", err)
	}

	// get the scheduler
	schedulerVal, ok := t.crons.Load(payload.WorkflowVersionId)

	if !ok {
		return fmt.Errorf("could not find cron %s", payload.WorkflowVersionId)
	}

	scheduler := schedulerVal.(*gocron.Scheduler)

	// cancel the cron
	scheduler.Clear()

	return nil
}
