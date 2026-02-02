package ticker

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func (t *TickerImpl) runPollSchedules(ctx context.Context) func() {
	return func() {
		ctx, cancel := context.WithTimeout(ctx, 1*time.Minute)
		defer cancel()

		t.l.Debug().Msgf("ticker: polling workflow schedules")

		scheduledWorkflows, err := t.repov1.Ticker().PollScheduledWorkflows(ctx, t.tickerId)

		if err != nil {
			t.l.Err(err).Msg("could not poll workflow schedules")
			return
		}

		existingSchedules := make(map[string]bool)

		t.scheduledWorkflows.Range(func(key string, _ context.CancelFunc) bool {
			existingSchedules[key] = false
			return true
		})

		for _, scheduledWorkflow := range scheduledWorkflows {
			workflowVersionId := scheduledWorkflow.WorkflowVersionId
			scheduledWorkflowId := scheduledWorkflow.ID

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

func (t *TickerImpl) handleScheduleWorkflow(ctx context.Context, scheduledWorkflow *sqlcv1.PollScheduledWorkflowsRow) error {
	t.l.Debug().Msg("ticker: scheduling workflow")

	// parse trigger time
	tenantId := scheduledWorkflow.TenantId
	workflowVersionId := scheduledWorkflow.WorkflowVersionId
	scheduledWorkflowId := scheduledWorkflow.ID
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

func (t *TickerImpl) runScheduledWorkflow(tenantId, workflowVersionId, scheduledWorkflowId uuid.UUID, scheduled *sqlcv1.PollScheduledWorkflowsRow) func() {
	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		t.l.Debug().Msgf("ticker: running workflow %s", workflowVersionId)

		workflowVersion, err := t.repov1.Workflows().GetWorkflowVersionById(ctx, tenantId, workflowVersionId)

		if err != nil {
			t.l.Err(err).Msg("could not get workflow version")
			return
		}

		err = t.runScheduledWorkflowV1(ctx, tenantId, workflowVersion, scheduledWorkflowId, scheduled)

		if err != nil {
			t.l.Error().Err(err).Msg("could not run scheduled workflow")
			return
		}
	}
}

func (t *TickerImpl) handleCancelWorkflow(ctx context.Context, key string) error {
	t.l.Debug().Msg("ticker: canceling scheduled workflow")

	// get the cancel function
	cancel, ok := t.scheduledWorkflows.Load(key)

	if !ok {
		return fmt.Errorf("could not find scheduled workflow with key %s", key)
	}

	defer t.scheduledWorkflows.Delete(key)

	// cancel the scheduled workflow
	cancel()

	return nil
}

func getScheduledWorkflowKey(workflowVersionId, scheduledWorkflowId uuid.UUID) string {
	return fmt.Sprintf("%s-%s", workflowVersionId.String(), scheduledWorkflowId.String())
}
