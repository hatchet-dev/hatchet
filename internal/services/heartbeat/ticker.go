package heartbeat

import (
	"context"
	"fmt"
	"time"

	"github.com/hatchet-dev/hatchet/internal/datautils"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes"
	"github.com/hatchet-dev/hatchet/internal/taskqueue"
)

func (t *HeartbeaterImpl) removeStaleTickers(ctx context.Context) func() {
	return func() {
		t.l.Debug().Msg("removing old tickers")

		removeTicker := func(tickerId string, getValidTickerId func() string) error {
			// unfortunate, but this includes a separate db call outside of the main tx. need to ensure this
			// does not lock the table
			ticker, err := t.repo.Ticker().GetTickerById(tickerId)

			if err != nil {
				return fmt.Errorf("could not get ticker %s: %w", tickerId, err)
			}

			// reschedule crons
			err = t.rescheduleCrons(ticker, getValidTickerId())

			if err != nil {
				return fmt.Errorf("could not reschedule crons for ticker %s: %w", ticker.ID, err)
			}

			// reschedule schedules
			err = t.rescheduleWorkflows(ticker, getValidTickerId())

			if err != nil {
				return fmt.Errorf("could not reschedule workflows for ticker %s: %w", ticker.ID, err)
			}

			// send a task to the job processing queue that the ticker is removed
			err = t.tq.AddTask(
				ctx,
				taskqueue.JOB_PROCESSING_QUEUE,
				tickerRemoved(ticker.ID),
			)

			if err != nil {
				t.l.Err(err).Msg("could not add ticker removed task")
				return err
			}

			return nil
		}

		err := t.repo.Ticker().UpdateStaleTickers(removeTicker)

		if err != nil {
			t.l.Err(err).Msgf("could not delete tickers")
		}
	}
}

func tickerRemoved(tickerId string) *taskqueue.Task {
	payload, _ := datautils.ToJSONMap(tasktypes.RemoveTickerTaskPayload{
		TickerId: tickerId,
	})

	metadata, _ := datautils.ToJSONMap(tasktypes.RemoveTickerTaskMetadata{})

	return &taskqueue.Task{
		ID:       "ticker-removed",
		Queue:    taskqueue.JOB_PROCESSING_QUEUE,
		Payload:  payload,
		Metadata: metadata,
	}
}

func (t *HeartbeaterImpl) rescheduleCrons(ticker *db.TickerModel, validTickerId string) error {
	for _, cronTrigger := range ticker.Crons() {
		cronTriggerCp := cronTrigger

		_, err := t.repo.Ticker().AddCron(
			validTickerId,
			&cronTriggerCp,
		)

		if err != nil {
			return err
		}

		task, err := cronScheduleTask(validTickerId, &cronTriggerCp, cronTriggerCp.Parent().Workflow())

		if err != nil {
			return err
		}

		// send to task queue
		err = t.tq.AddTask(
			context.TODO(),
			taskqueue.QueueTypeFromTickerID(ticker.ID),
			task,
		)

		if err != nil {
			return err
		}
	}

	return nil
}

func (t *HeartbeaterImpl) rescheduleWorkflows(ticker *db.TickerModel, validTickerId string) error {
	for _, schedule := range ticker.Scheduled() {
		scheduleCp := schedule

		// only send to task queue if the trigger is in the future
		if scheduleCp.TriggerAt.After(time.Now().UTC()) {

			_, err := t.repo.Ticker().AddScheduledWorkflow(
				validTickerId,
				&scheduleCp,
			)

			if err != nil {
				return err
			}

			task, err := workflowScheduleTask(validTickerId, &scheduleCp, scheduleCp.Parent())

			if err != nil {
				return err
			}

			// send to task queue
			err = t.tq.AddTask(
				context.TODO(),
				taskqueue.QueueTypeFromTickerID(ticker.ID),
				task,
			)

			if err != nil {
				return err
			}
		}
	}

	return nil
}

func cronScheduleTask(tickerId string, cronTriggerRef *db.WorkflowTriggerCronRefModel, workflowVersion *db.WorkflowVersionModel) (*taskqueue.Task, error) {
	payload, _ := datautils.ToJSONMap(tasktypes.ScheduleCronTaskPayload{
		CronParentId:      cronTriggerRef.ParentID,
		Cron:              cronTriggerRef.Cron,
		WorkflowVersionId: workflowVersion.ID,
	})

	metadata, _ := datautils.ToJSONMap(tasktypes.ScheduleCronTaskMetadata{
		TenantId: workflowVersion.Workflow().TenantID,
	})

	return &taskqueue.Task{
		ID:       "schedule-cron",
		Queue:    taskqueue.QueueTypeFromTickerID(tickerId),
		Payload:  payload,
		Metadata: metadata,
	}, nil
}

func workflowScheduleTask(tickerId string, workflowTriggerRef *db.WorkflowTriggerScheduledRefModel, workflowVersion *db.WorkflowVersionModel) (*taskqueue.Task, error) {
	payload, _ := datautils.ToJSONMap(tasktypes.ScheduleWorkflowTaskPayload{
		ScheduledWorkflowId: workflowTriggerRef.ID,
		TriggerAt:           workflowTriggerRef.TriggerAt.Format(time.RFC3339),
		WorkflowVersionId:   workflowVersion.ID,
	})

	metadata, _ := datautils.ToJSONMap(tasktypes.ScheduleWorkflowTaskMetadata{
		TenantId: workflowVersion.Workflow().TenantID,
	})

	return &taskqueue.Task{
		ID:       "schedule-workflow",
		Queue:    taskqueue.QueueTypeFromTickerID(tickerId),
		Payload:  payload,
		Metadata: metadata,
	}, nil
}
