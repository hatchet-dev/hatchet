package ticker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// refreshCronSchedules acquires a list of cron schedules from the database and schedules any which are not
// already scheduled. This job runs in "singleton" mode, meaning that only one instance of this job will run at
// a time.
func (t *TickerImpl) refreshCronSchedules(ctx context.Context) func() {
	return func() {
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		// prevent multiple refreshes from running simultaneously
		if t.userCronRefreshLock.TryLock() {
			defer t.userCronRefreshLock.Unlock()

			t.l.Debug().Ctx(ctx).Msgf("ticker: polling cron schedules")

			crons, err := t.repov1.Ticker().PollCronSchedules(ctx, t.tickerId)

			if err != nil {
				t.l.Err(err).Ctx(ctx).Msg("could not poll cron schedules")
				return
			}

			// guard access to the userCronScheduler and userCronSchedulesToIds
			t.userCronSchedulerLock.Lock()
			defer t.userCronSchedulerLock.Unlock()

			newCronKeys := make(map[string]bool)

			for _, cron := range crons {
				cronKey := getCronKey(cron)

				newCronKeys[cronKey] = true

				t.l.Debug().Ctx(ctx).Msgf("ticker: handling cron %s", cronKey)

				// if the cron is already scheduled, skip
				if _, ok := t.userCronSchedulesToIds[cronKey]; ok {
					continue
				}

				// if the cron is not scheduled, schedule it
				if err := t.handleScheduleCron(ctx, cron); err != nil {
					t.l.Err(err).Ctx(ctx).Msg("could not schedule cron")
				}
			}

			// cancel any crons that are no longer assigned to this ticker
			for key := range t.userCronSchedulesToIds {
				if _, ok := newCronKeys[key]; !ok {
					if err := t.handleCancelCron(ctx, key); err != nil {
						t.l.Err(err).Ctx(ctx).Msg("could not cancel cron")
					}
				}
			}

		} else {
			t.l.Debug().Ctx(ctx).Msgf("ticker: skipping cron refresh")
		}

	}
}

func (t *TickerImpl) handleScheduleCron(ctx context.Context, cron *sqlcv1.PollCronSchedulesRow) error {
	t.l.Debug().Ctx(ctx).Msg("ticker: scheduling cron")

	tenantId := cron.TenantId
	workflowVersionId := cron.WorkflowVersionId
	cronParentId := cron.ParentId.String()

	var additionalMetadata map[string]interface{}

	if cron.AdditionalMetadata != nil {
		if err := json.Unmarshal(cron.AdditionalMetadata, &additionalMetadata); err != nil {
			return fmt.Errorf("could not unmarshal additional metadata: %w", err)
		}
	}

	cronUUID := uuid.New()

	var job gocron.Job
	var scheduledAtPtr atomic.Pointer[time.Time]

	// schedule the cron
	job, err := t.userCronScheduler.NewJob(
		// the gocron library accepts either 5 or 6 term crontabs when withSeconds is true
		gocron.CronJob(cron.Cron, true),
		gocron.NewTask(func() {
			scheduledAt := scheduledAtPtr.Load()
			if scheduledAt == nil {
				// this should be pretty rare in practice, since we're assigning the `scheduledAtPtr` immediately before
				// it triggers, but just here for nil safety
				now := time.Now().UTC()
				scheduledAt = &now
			}

			t.runCronWorkflow(
				tenantId, workflowVersionId, cron.Cron,
				cronParentId, &cron.Name.String, cron.Input,
				additionalMetadata, &cron.Priority,
				*scheduledAt,
			)()
		}),
		gocron.WithIdentifier(cronUUID),
		gocron.WithEventListeners(
			// this runs sync before the gocron task in the same goroutine,
			// and before gocron reschedules the job — so NextRun() here returns the
			// scheduled fire time for the current run, not the subsequent one.
			gocron.BeforeJobRuns(func(_ uuid.UUID, _ string) {
				if nextRun, err := job.NextRun(); err == nil && !nextRun.IsZero() {
					scheduledAtPtr.Store(&nextRun)
				}
			}),
		),
	)

	if err != nil {
		if errors.Is(err, gocron.ErrCronJobParse) || errors.Is(err, gocron.ErrCronJobInvalid) {
			deleteCronErr := t.repov1.WorkflowSchedules().DeleteInvalidCron(ctx, cron.ID)

			if deleteCronErr != nil {
				t.l.Error().Ctx(ctx).Err(deleteCronErr).Msg("could not delete invalid cron from database")
			}
		}

		return fmt.Errorf("could not schedule cron: %w", err)
	}

	// store the schedule in the cron map
	// NOTE: we already have a lock on the userCronSchedulerLock when we call this function, so we don't need to lock here
	t.userCronSchedulesToIds[getCronKey(cron)] = cronUUID.String()

	return nil
}

func (t *TickerImpl) runCronWorkflow(tenantId, workflowVersionId uuid.UUID, cron, cronParentId string, cronName *string, input []byte, additionalMetadata map[string]interface{}, priority *int32, scheduledAt time.Time) func() {
	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		t.l.Debug().Ctx(ctx).Msgf("ticker: running workflow %s", workflowVersionId)

		workflowVersion, err := t.repov1.Workflows().GetWorkflowVersionById(ctx, tenantId, workflowVersionId)

		if err != nil {
			t.l.Error().Ctx(ctx).Err(err).Msg("could not get workflow version")
			return
		}

		err = t.runCronWorkflowV1(ctx, tenantId, workflowVersion, cron, cronParentId, cronName, input, additionalMetadata, priority, scheduledAt)

		if err != nil {
			t.l.Error().Ctx(ctx).Err(err).Msg("could not run cron workflow")
		}
	}
}

func (t *TickerImpl) handleCancelCron(ctx context.Context, key string) error {
	t.l.Debug().Ctx(ctx).Msg("ticker: canceling cron")

	cronUUID, ok := t.userCronSchedulesToIds[key]

	if !ok {
		return nil
	}

	// Clean up the map entry if it exists
	delete(t.userCronSchedulesToIds, key)

	cronAsUUID, err := uuid.Parse(cronUUID)

	if err != nil {
		return fmt.Errorf("could not parse cron UUID: %w", err)
	}

	err = t.userCronScheduler.RemoveJob(cronAsUUID)

	if err != nil {
		return fmt.Errorf("could not remove job from scheduler: %w", err)
	}

	return nil
}

func getCronKey(cron *sqlcv1.PollCronSchedulesRow) string {
	workflowVersionId := cron.WorkflowVersionId.String()

	switch cron.Method {
	case sqlcv1.WorkflowTriggerCronRefMethodsAPI:
		return fmt.Sprintf("API-%s-%s-%s", workflowVersionId, cron.Cron, cron.Name.String)
	default:
		return fmt.Sprintf("DEFAULT-%s-%s", workflowVersionId, cron.Cron)
	}
}
