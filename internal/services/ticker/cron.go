package ticker

import (
	"context"
	"fmt"
	"time"

	"github.com/go-co-op/gocron/v2"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/internal/repository"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes"
)

func (t *TickerImpl) handleScheduleCron(ctx context.Context, task *msgqueue.Message) error {
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
	s, err := gocron.NewScheduler(gocron.WithLocation(time.UTC))

	if err != nil {
		return fmt.Errorf("could not create scheduler: %w", err)
	}

	// schedule the cron
	_, err = t.s.NewJob(
		gocron.CronJob(payload.Cron, false),
		gocron.NewTask(
			t.runCronWorkflow(ctx, metadata.TenantId, &payload, workflowVersion),
		),
	)

	if err != nil {
		return fmt.Errorf("could not schedule cron: %w", err)
	}

	// store the schedule in the cron map
	t.crons.Store(getCronKey(payload.WorkflowVersionId, payload.Cron), s)

	s.Start()

	return nil
}

func (t *TickerImpl) runCronWorkflow(ctx context.Context, tenantId string, payload *tasktypes.ScheduleCronTaskPayload, workflowVersion *db.WorkflowVersionModel) func() {
	return func() {
		t.l.Debug().Msgf("ticker: running workflow %s", payload.WorkflowVersionId)

		// create a new workflow run in the database
		createOpts, err := repository.GetCreateWorkflowRunOptsFromCron(payload.Cron, payload.CronParentId, workflowVersion)

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
			err = t.mq.AddMessage(
				context.Background(),
				msgqueue.JOB_PROCESSING_QUEUE,
				tasktypes.JobRunQueuedToTask(jobRun.Job(), &jobRunCp),
			)

			if err != nil {
				t.l.Err(err).Msg("could not add job run queued task")
				continue
			}
		}
	}
}

func (t *TickerImpl) handleCancelCron(ctx context.Context, task *msgqueue.Message) error {
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
	schedulerVal, ok := t.crons.Load(getCronKey(payload.WorkflowVersionId, payload.Cron))

	if !ok {
		return fmt.Errorf("could not find cron %s with schedule %s", payload.WorkflowVersionId, payload.Cron)
	}

	defer t.crons.Delete(getCronKey(payload.WorkflowVersionId, payload.Cron))

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
