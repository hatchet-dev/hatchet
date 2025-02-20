package ticker

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-co-op/gocron/v2"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/sqlchelpers"
)

func (t *TickerImpl) runPollSchedules(ctx context.Context) func() {
	return func() {
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		t.l.Debug().Msgf("ticker: polling workflow schedules")

		scheduledWorkflows, err := t.repo.Ticker().PollScheduledWorkflows(ctx, t.tickerId)

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

		t.runScheduledWorkflow(tenantId, workflowVersionId, scheduledWorkflowId, scheduledWorkflow)()
		return nil
	}

	// schedule the workflow
	_, err = s.NewJob(
		gocron.OneTimeJob(
			gocron.OneTimeJobStartDateTime(triggerAt),
		),
		gocron.NewTask(
			t.runScheduledWorkflow(tenantId, workflowVersionId, scheduledWorkflowId, scheduledWorkflow),
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

func (t *TickerImpl) runScheduledWorkflow(tenantId, workflowVersionId, scheduledWorkflowId string, scheduled *dbsqlc.PollScheduledWorkflowsRow) func() {
	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		t.l.Debug().Msgf("ticker: running workflow %s", workflowVersionId)

		workflowVersion, err := t.repo.Workflow().GetWorkflowVersionById(ctx, tenantId, workflowVersionId)

		if err != nil {
			t.l.Err(err).Msg("could not get workflow version")
			return
		}

		fs := make([]repository.CreateWorkflowRunOpt, 0)

		var additionalMetadata map[string]interface{}

		if scheduled.AdditionalMetadata != nil {
			err := json.Unmarshal(scheduled.AdditionalMetadata, &additionalMetadata)
			if err != nil {
				t.l.Err(err).Msg("could not unmarshal additional metadata")
				return
			}
		}

		if scheduled.ParentWorkflowRunId.Valid {
			var childKey *string

			if scheduled.ChildKey.Valid {
				childKey = &scheduled.ChildKey.String
			}

			parent, err := t.repo.WorkflowRun().GetWorkflowRunById(ctx, tenantId, sqlchelpers.UUIDToStr(scheduled.ParentWorkflowRunId))

			if err != nil {
				t.l.Err(err).Msg("could not get parent workflow run")
				return
			}

			var parentAdditionalMeta map[string]interface{}

			if parent.WorkflowRun.AdditionalMetadata != nil {
				err := json.Unmarshal(parent.WorkflowRun.AdditionalMetadata, &parentAdditionalMeta)
				if err != nil {
					t.l.Err(err).Msg("could not unmarshal parent additional metadata")
					return
				}
			}

			fs = append(fs, repository.WithParent(
				sqlchelpers.UUIDToStr(scheduled.ParentWorkflowRunId),
				sqlchelpers.UUIDToStr(scheduled.ParentStepRunId),
				int(scheduled.ChildIndex.Int32),
				childKey,
				additionalMetadata,
				parentAdditionalMeta,
			))
		}

		// create a new workflow run in the database
		createOpts, err := repository.GetCreateWorkflowRunOptsFromSchedule(
			scheduledWorkflowId,
			workflowVersion,
			scheduled.Input,
			additionalMetadata,
			fs...,
		)

		if err != nil {
			t.l.Err(err).Msg("could not get create workflow run opts")
			return
		}

		workflowRun, err := t.repo.WorkflowRun().CreateNewWorkflowRun(ctx, tenantId, createOpts)

		if err != nil {
			t.l.Err(err).Msg("could not create workflow run")
			return
		}

		workflowRunId := sqlchelpers.UUIDToStr(workflowRun.ID)

		err = t.mq.AddMessage(
			context.Background(),
			msgqueue.WORKFLOW_PROCESSING_QUEUE,
			tasktypes.WorkflowRunQueuedToTask(tenantId, workflowRunId),
		)

		if err != nil {
			t.l.Err(err).Msg("could not add workflow run queued task")
			return
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

	scheduler, ok := schedulerVal.(gocron.Scheduler)

	if !ok {
		return fmt.Errorf("could not cast scheduler")
	}

	// cancel the schedule
	return scheduler.Shutdown()
}

func getScheduledWorkflowKey(workflowVersionId, scheduledWorkflowId string) string {
	return fmt.Sprintf("%s-%s", workflowVersionId, scheduledWorkflowId)
}
