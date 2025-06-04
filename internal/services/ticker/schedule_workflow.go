package ticker

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
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

	// parse trigger time
	tenantId := sqlchelpers.UUIDToStr(scheduledWorkflow.TenantId)
	workflowVersionId := sqlchelpers.UUIDToStr(scheduledWorkflow.WorkflowVersionId)
	scheduledWorkflowId := sqlchelpers.UUIDToStr(scheduledWorkflow.ID)
	triggerAt := scheduledWorkflow.TriggerAt.Time

	key := getScheduledWorkflowKey(workflowVersionId, scheduledWorkflowId)

	if _, exists := t.scheduledWorkflows.Load(key); exists {
		return fmt.Errorf("workflow %s already scheduled", key)
	}

	// if start is in the past, run now
	if triggerAt.Before(time.Now()) {
		t.l.Debug().Msg("ticker: trigger time is in the past, running now")

		t.runScheduledWorkflow(tenantId, workflowVersionId, scheduledWorkflowId, scheduledWorkflow)()
		return nil
	}

	duration := time.Until(triggerAt)

	// create a cancellation context for this specific scheduled workflow
	cancelCtx, cancel := context.WithTimeout(context.Background(), duration*2)

	// store the cancel function so we can cancel this scheduled workflow later
	t.scheduledWorkflows.Store(key, cancel)

	// start a goroutine to wait for the trigger time
	go func() {
		defer cancel()
		defer t.scheduledWorkflows.Delete(key)

		timer := time.After(duration)

		select {
		case <-cancelCtx.Done():
			t.l.Debug().Msgf("ticker: scheduled workflow %s was cancelled", key)
			return
		case <-timer:
			t.runScheduledWorkflow(tenantId, workflowVersionId, scheduledWorkflowId, scheduledWorkflow)()
		}
	}()

	return nil
}

func (t *TickerImpl) runScheduledWorkflow(tenantId, workflowVersionId, scheduledWorkflowId string, scheduled *dbsqlc.PollScheduledWorkflowsRow) func() {
	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		t.l.Debug().Msgf("ticker: running workflow %s", workflowVersionId)

		tenant, err := t.repo.Tenant().GetTenantByID(ctx, tenantId)

		if err != nil {
			t.l.Error().Err(err).Msg("could not get tenant")
			return
		}

		workflowVersion, err := t.repo.Workflow().GetWorkflowVersionById(ctx, tenantId, workflowVersionId)

		if err != nil {
			t.l.Err(err).Msg("could not get workflow version")
			return
		}

		switch tenant.Version {
		case dbsqlc.TenantMajorEngineVersionV0:
			err = t.runScheduledWorkflowV0(ctx, tenantId, workflowVersion, scheduledWorkflowId, scheduled)
		case dbsqlc.TenantMajorEngineVersionV1:
			err = t.runScheduledWorkflowV1(ctx, tenantId, workflowVersion, scheduledWorkflowId, scheduled)
		default:
			t.l.Error().Msgf("unsupported tenant major engine version %s", tenant.Version)
			return
		}

		if err != nil {
			t.l.Error().Err(err).Msg("could not run scheduled workflow")
			return
		}
	}
}

func (t *TickerImpl) runScheduledWorkflowV0(ctx context.Context, tenantId string, workflowVersion *dbsqlc.GetWorkflowVersionForEngineRow, scheduledWorkflowId string, scheduled *dbsqlc.PollScheduledWorkflowsRow) error {
	fs := make([]repository.CreateWorkflowRunOpt, 0)

	var additionalMetadata map[string]interface{}

	if scheduled.AdditionalMetadata != nil {
		err := json.Unmarshal(scheduled.AdditionalMetadata, &additionalMetadata)
		if err != nil {
			return fmt.Errorf("could not unmarshal additional metadata: %w", err)
		}
	}

	if scheduled.ParentWorkflowRunId.Valid {
		var childKey *string

		if scheduled.ChildKey.Valid {
			childKey = &scheduled.ChildKey.String
		}

		parent, err := t.repo.WorkflowRun().GetWorkflowRunById(ctx, tenantId, sqlchelpers.UUIDToStr(scheduled.ParentWorkflowRunId))

		if err != nil {
			return fmt.Errorf("could not get parent workflow run: %w", err)
		}

		var parentAdditionalMeta map[string]interface{}

		if parent.WorkflowRun.AdditionalMetadata != nil {
			err := json.Unmarshal(parent.WorkflowRun.AdditionalMetadata, &parentAdditionalMeta)
			if err != nil {
				return fmt.Errorf("could not unmarshal parent additional metadata: %w", err)
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
		return fmt.Errorf("could not get create workflow run opts: %w", err)
	}

	workflowRun, err := t.repo.WorkflowRun().CreateNewWorkflowRun(ctx, tenantId, createOpts)

	if err != nil {
		return fmt.Errorf("could not create workflow run: %w", err)
	}

	workflowRunId := sqlchelpers.UUIDToStr(workflowRun.ID)

	err = t.mq.AddMessage(
		context.Background(),
		msgqueue.WORKFLOW_PROCESSING_QUEUE,
		tasktypes.WorkflowRunQueuedToTask(tenantId, workflowRunId),
	)

	if err != nil {
		return fmt.Errorf("could not add workflow run queued task: %w", err)
	}

	return nil
}

func (t *TickerImpl) handleCancelWorkflow(ctx context.Context, key string) error {
	t.l.Debug().Msg("ticker: canceling scheduled workflow")

	// get the cancel function
	cancelVal, ok := t.scheduledWorkflows.Load(key)

	if !ok {
		return fmt.Errorf("could not find scheduled workflow with key %s", key)
	}

	defer t.scheduledWorkflows.Delete(key)

	cancel, ok := cancelVal.(context.CancelFunc)

	if !ok {
		return fmt.Errorf("could not cast cancel function")
	}

	// cancel the scheduled workflow
	cancel()

	return nil
}

func getScheduledWorkflowKey(workflowVersionId, scheduledWorkflowId string) string {
	return fmt.Sprintf("%s-%s", workflowVersionId, scheduledWorkflowId)
}
