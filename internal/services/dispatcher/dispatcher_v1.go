package dispatcher

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/hashicorp/go-multierror"
	"golang.org/x/sync/errgroup"

	msgqueuev1 "github.com/hatchet-dev/hatchet/internal/msgqueue/v1"
	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	tasktypesv1 "github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes/v1"
	"github.com/hatchet-dev/hatchet/internal/telemetry"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"
)

func (worker *subscribedWorker) StartTaskFromBulk(
	ctx context.Context,
	tenantId string,
	task *sqlcv1.V1Task,
) error {
	ctx, span := telemetry.NewSpan(ctx, "start-step-run-from-bulk") // nolint:ineffassign
	defer span.End()

	inputBytes := []byte{}

	if task.Input != nil {
		inputBytes = task.Input
	}

	action := populateAssignedAction(tenantId, task)

	action.ActionType = contracts.ActionType_START_STEP_RUN
	action.ActionPayload = string(inputBytes)

	worker.sendMu.Lock()
	defer worker.sendMu.Unlock()

	return worker.stream.Send(action)
}

func (worker *subscribedWorker) CancelTask(
	ctx context.Context,
	tenantId string,
	task *sqlcv1.V1Task,
) error {
	ctx, span := telemetry.NewSpan(ctx, "cancel-task") // nolint:ineffassign
	defer span.End()

	action := populateAssignedAction(tenantId, task)

	action.ActionType = contracts.ActionType_CANCEL_STEP_RUN

	worker.sendMu.Lock()
	defer worker.sendMu.Unlock()

	return worker.stream.Send(action)
}

func populateAssignedAction(tenantId string, task *sqlcv1.V1Task) *contracts.AssignedAction {
	action := &contracts.AssignedAction{
		TenantId:      tenantId,
		JobId:         sqlchelpers.UUIDToStr(task.StepID), // FIXME
		JobName:       task.StepReadableID,
		JobRunId:      sqlchelpers.UUIDToStr(task.ExternalID), // FIXME
		StepId:        sqlchelpers.UUIDToStr(task.StepID),
		StepRunId:     sqlchelpers.UUIDToStr(task.ExternalID),
		ActionId:      task.ActionID,
		StepName:      task.StepReadableID,
		WorkflowRunId: sqlchelpers.UUIDToStr(task.WorkflowRunID),
		RetryCount:    task.RetryCount,
	}

	if task.AdditionalMetadata != nil {
		metadataStr := string(task.AdditionalMetadata)
		action.AdditionalMetadata = &metadataStr
	}

	if task.ParentTaskExternalID.Valid {
		parentId := sqlchelpers.UUIDToStr(task.ParentTaskExternalID)
		action.ParentWorkflowRunId = &parentId
	}

	if task.ChildIndex.Valid {
		i := int32(task.ChildIndex.Int64) // nolint: gosec
		action.ChildWorkflowIndex = &i
	}

	if task.ChildKey.Valid {
		key := task.ChildKey.String
		action.ChildWorkflowKey = &key
	}

	return action
}

func (d *DispatcherImpl) handleTaskBulkAssignedTask(ctx context.Context, msg *msgqueuev1.Message) error {
	ctx, span := telemetry.NewSpanWithCarrier(ctx, "task-assigned-bulk", msg.OtelCarrier)
	defer span.End()

	// we set a timeout of 25 seconds because we don't want to hold the semaphore for longer than the visibility timeout (30 seconds)
	// on the worker
	ctx, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()

	msgs := msgqueuev1.JSONConvert[tasktypesv1.TaskAssignedBulkTaskPayload](msg.Payloads)

	for _, innerMsg := range msgs {
		// load the step runs from the database
		taskIds := make([]int64, 0)

		for _, tasks := range innerMsg.WorkerIdToTaskIds {
			taskIds = append(taskIds, tasks...)
		}

		bulkDatas, err := d.repov1.Tasks().ListTasks(ctx, msg.TenantID, taskIds)

		if err != nil {
			return fmt.Errorf("could not bulk list step run data: %w", err)
		}

		taskIdToData := make(map[int64]*sqlcv1.V1Task)

		for _, task := range bulkDatas {
			taskIdToData[task.ID] = task
		}

		outerEg := errgroup.Group{}

		for workerId, stepRunIds := range innerMsg.WorkerIdToTaskIds {
			workerId := workerId

			outerEg.Go(func() error {
				d.l.Debug().Msgf("worker %s has %d step runs", workerId, len(stepRunIds))

				// get the worker for this task
				workers, err := d.workers.Get(workerId)

				if err != nil && !errors.Is(err, ErrWorkerNotFound) {
					return fmt.Errorf("could not get worker: %w", err)
				}

				innerEg := errgroup.Group{}

				// toRetry := []string{}
				// toRetryMu := sync.Mutex{}

				for _, stepRunId := range stepRunIds {
					stepRunId := stepRunId

					innerEg.Go(func() error {
						task := taskIdToData[stepRunId]

						// requeue := func() {
						// 	toRetryMu.Lock()
						// 	toRetry = append(toRetry, stepRunId)
						// 	toRetryMu.Unlock()
						// }

						// if we've reached the context deadline, this should be requeued
						if ctx.Err() != nil {
							// FIXME
							// requeue()
							return nil
						}

						// // if the step run has a job run in a non-running state, we should not send it to the worker
						// if repository.IsFinalJobRunStatus(stepRun.JobRunStatus) {
						// 	d.l.Debug().Msgf("job run %s is in a final state %s, ignoring", sqlchelpers.UUIDToStr(stepRun.JobRunId), string(stepRun.JobRunStatus))

						// 	// release the semaphore
						// 	return d.repo.StepRun().ReleaseStepRunSemaphore(ctx, metadata.TenantId, stepRunId, false)
						// }

						// // if the step run is in a final state, we should not send it to the worker
						// if repository.IsFinalStepRunStatus(stepRun.Status) {
						// 	d.l.Warn().Msgf("step run %s is in a final state %s, ignoring", stepRunId, string(stepRun.Status))

						// 	return d.repo.StepRun().ReleaseStepRunSemaphore(ctx, metadata.TenantId, stepRunId, false)
						// }

						var multiErr error
						var success bool

						for i, w := range workers {
							err := w.StartTaskFromBulk(ctx, msg.TenantID, task)

							if err != nil {
								d.l.Err(err).Msgf("could not send step run to worker (%d)", i)
								multiErr = multierror.Append(multiErr, fmt.Errorf("could not send step action to worker (%d): %w", i, err))
							} else {
								success = true
								break
							}
						}

						// now := time.Now().UTC()

						if success {
							// defer d.repo.StepRun().DeferredStepRunEvent(
							// 	metadata.TenantId,
							// 	repository.CreateStepRunEventOpts{
							// 		StepRunId:     sqlchelpers.UUIDToStr(stepRun.SRID),
							// 		EventMessage:  repository.StringPtr("Sent step run to the assigned worker"),
							// 		EventReason:   repository.StepRunEventReasonPtr(dbsqlc.StepRunEventReasonSENTTOWORKER),
							// 		EventSeverity: repository.StepRunEventSeverityPtr(dbsqlc.StepRunEventSeverityINFO),
							// 		Timestamp:     &now,
							// 		EventData:     map[string]interface{}{"worker_id": workerId},
							// 	},
							// )

							return nil
						}

						// defer d.repo.StepRun().DeferredStepRunEvent(
						// 	metadata.TenantId,
						// 	repository.CreateStepRunEventOpts{
						// 		StepRunId:     sqlchelpers.UUIDToStr(stepRun.SRID),
						// 		EventMessage:  repository.StringPtr("Could not send step run to assigned worker"),
						// 		EventReason:   repository.StepRunEventReasonPtr(dbsqlc.StepRunEventReasonREASSIGNED),
						// 		EventSeverity: repository.StepRunEventSeverityPtr(dbsqlc.StepRunEventSeverityWARNING),
						// 		Timestamp:     &now,
						// 		EventData:     map[string]interface{}{"worker_id": workerId},
						// 	},
						// )

						// requeue()
						// FIXME

						return multiErr
					})
				}

				innerErr := innerEg.Wait()

				// if len(toRetry) > 0 {
				// 	retryCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
				// 	defer cancel()

				// 	_, stepRunsToFail, err := d.repo.StepRun().InternalRetryStepRuns(retryCtx, metadata.TenantId, toRetry)

				// 	if err != nil {
				// 		innerErr = multierror.Append(innerErr, fmt.Errorf("could not requeue step runs: %w", err))
				// 	}

				// 	if len(stepRunsToFail) > 0 {
				// 		now := time.Now()

				// 		batchErr := queueutils.BatchConcurrent(50, stepRunsToFail, func(stepRuns []*dbsqlc.GetStepRunForEngineRow) error {
				// 			var innerBatchErr error

				// 			for _, stepRun := range stepRuns {
				// 				err := d.mq.SendMessage(
				// 					retryCtx,
				// 					msgqueuev1.JOB_PROCESSING_QUEUE,
				// 					tasktypesv1.StepRunFailedToTask(
				// 						stepRun,
				// 						"Could not send step run to worker",
				// 						&now,
				// 					),
				// 				)

				// 				if err != nil {
				// 					innerBatchErr = multierror.Append(innerBatchErr, err)
				// 				}
				// 			}

				// 			return innerBatchErr
				// 		})

				// 		if batchErr != nil {
				// 			innerErr = multierror.Append(innerErr, fmt.Errorf("could not fail step runs: %w", batchErr))
				// 		}
				// 	}
				// }

				return innerErr
			})
		}

		return outerEg.Wait()
	}

	return nil
}

func (d *DispatcherImpl) handleTaskCancelled(ctx context.Context, msg *msgqueuev1.Message) error {
	ctx, span := telemetry.NewSpanWithCarrier(ctx, "tasks-cancelled", msg.OtelCarrier)
	defer span.End()

	// we set a timeout of 25 seconds because we don't want to hold the semaphore for longer than the visibility timeout (30 seconds)
	// on the worker
	ctx, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()

	msgs := msgqueuev1.JSONConvert[tasktypesv1.SignalTaskCancelledPayload](msg.Payloads)

	taskIds := make([]int64, 0)

	for _, innerMsg := range msgs {
		taskIds = append(taskIds, innerMsg.TaskId)
	}

	tasks, err := d.repov1.Tasks().ListTasks(ctx, msg.TenantID, taskIds)

	if err != nil {
		return fmt.Errorf("could not list tasks: %w", err)
	}

	taskIdsToTasks := make(map[int64]*sqlcv1.V1Task)

	for _, task := range tasks {
		taskIdsToTasks[task.ID] = task
	}

	// group by worker id
	workerIdToTasks := make(map[string][]*sqlcv1.V1Task)

	for _, msg := range msgs {
		if _, ok := workerIdToTasks[msg.WorkerId]; !ok {
			workerIdToTasks[msg.WorkerId] = []*sqlcv1.V1Task{}
		}

		task, ok := taskIdsToTasks[msg.TaskId]

		if !ok {
			d.l.Warn().Msgf("task %d not found", msg.TaskId)
			continue
		}

		workerIdToTasks[msg.WorkerId] = append(workerIdToTasks[msg.WorkerId], task)
	}

	var multiErr error

	for workerId, tasks := range workerIdToTasks {
		// get the worker for this task
		workers, err := d.workers.Get(workerId)

		if err != nil && !errors.Is(err, ErrWorkerNotFound) {
			return fmt.Errorf("could not get worker: %w", err)
		} else if errors.Is(err, ErrWorkerNotFound) {
			// if the worker is not found, we can ignore this task
			d.l.Debug().Msgf("worker %s not found, ignoring task", workerId)
			continue
		}

		for _, w := range workers {
			for _, task := range tasks {
				err = w.CancelTask(ctx, msg.TenantID, task)

				if err != nil {
					multiErr = multierror.Append(multiErr, fmt.Errorf("could not send job to worker: %w", err))
				}
			}
		}
	}

	return multiErr
}
