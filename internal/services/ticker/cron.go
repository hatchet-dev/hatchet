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
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
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
			cronKey := getCronKey(cron)

			t.l.Debug().Msgf("ticker: handling cron %s", cronKey)

			// if the cron is already scheduled, mark it as existing
			if _, ok := existingCrons[cronKey]; ok {
				existingCrons[cronKey] = true
				continue
			}

			// if the cron is not scheduled, schedule it
			if err := t.handleScheduleCron(ctx, cron); err != nil {
				t.l.Err(err).Msg("could not schedule cron")
			}

			existingCrons[cronKey] = true
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

	var additionalMetadata map[string]interface{}

	if cron.AdditionalMetadata != nil {
		if err := json.Unmarshal(cron.AdditionalMetadata, &additionalMetadata); err != nil {
			return fmt.Errorf("could not unmarshal additional metadata: %w", err)
		}
	}

	// schedule the cron
	_, err = s.NewJob(
		gocron.CronJob(cron.Cron, false),
		gocron.NewTask(
			t.runCronWorkflow(tenantId, workflowVersionId, cron.Cron, cronParentId, &cron.Name.String, cron.Input, additionalMetadata),
		),
	)

	if err != nil {
		return fmt.Errorf("could not schedule cron: %w", err)
	}

	// store the schedule in the cron map
	t.crons.Store(getCronKey(cron), s)

	s.Start()

	return nil
}

func (t *TickerImpl) runCronWorkflow(tenantId, workflowVersionId, cron, cronParentId string, cronName *string, input []byte, additionalMetadata map[string]interface{}) func() {
	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		t.l.Debug().Msgf("ticker: running workflow %s", workflowVersionId)

		workflowVersion, err := t.repo.Workflow().GetWorkflowVersionById(ctx, tenantId, workflowVersionId)

		if err != nil {
			t.l.Err(err).Msg("could not get workflow version")
			return
		}
		// create a new workflow run in the database
		createOpts, err := repository.GetCreateWorkflowRunOptsFromCron(cron, cronParentId, cronName, workflowVersion, input, additionalMetadata)

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

	scheduler, ok := schedulerVal.(gocron.Scheduler)

	if !ok {
		return fmt.Errorf("could not cast scheduler")
	}

	// cancel the cron
	if err := scheduler.Shutdown(); err != nil {
		return fmt.Errorf("could not cancel cron: %w", err)
	}

	return nil
}

func getCronKey(cron *dbsqlc.PollCronSchedulesRow) string {
	workflowVersionId := sqlchelpers.UUIDToStr(cron.WorkflowVersionId)

	switch cron.Method {
	case dbsqlc.WorkflowTriggerCronRefMethodsAPI:
		return fmt.Sprintf("API-%s-%s-%s", workflowVersionId, cron.Cron, cron.Name.String)
	default:
		return fmt.Sprintf("DEFAULT-%s-%s", workflowVersionId, cron.Cron)
	}
}
