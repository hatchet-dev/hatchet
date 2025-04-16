package dispatcher

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/hashicorp/go-multierror"
	"golang.org/x/sync/errgroup"

	msgqueuev1 "github.com/hatchet-dev/hatchet/internal/msgqueue/v1"
	"github.com/hatchet-dev/hatchet/internal/queueutils"
	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	tasktypesv1 "github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes/v1"
	"github.com/hatchet-dev/hatchet/internal/telemetry"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository/v1"
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
		Priority:      task.Priority.Int32,
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
	outerEg := errgroup.Group{}

	toRetry := []*sqlcv1.V1Task{}
	toRetryMu := sync.Mutex{}

	requeue := func(task *sqlcv1.V1Task) {
		toRetryMu.Lock()
		toRetry = append(toRetry, task)
		toRetryMu.Unlock()
	}

	for _, innerMsg := range msgs {
		// load the step runs from the database
		taskIds := make([]int64, 0)

		for _, tasks := range innerMsg.WorkerIdToTaskIds {
			taskIds = append(taskIds, tasks...)
		}

		bulkDatas, err := d.repov1.Tasks().ListTasks(ctx, msg.TenantID, taskIds)

		if err != nil {
			for _, task := range bulkDatas {
				requeue(task)
			}

			d.l.Error().Err(err).Msgf("could not bulk list step run data:")
			continue
		}

		parentDataMap, err := d.repov1.Tasks().ListTaskParentOutputs(ctx, msg.TenantID, bulkDatas)

		if err != nil {
			for _, task := range bulkDatas {
				requeue(task)
			}

			d.l.Error().Err(err).Msgf("could not list parent data for %d tasks", len(bulkDatas))
			continue
		}

		for _, task := range bulkDatas {
			if parentData, ok := parentDataMap[task.ID]; ok {
				currInput := &v1.V1StepRunData{}

				if task.Input != nil {
					err := json.Unmarshal(task.Input, currInput)

					if err != nil {
						d.l.Warn().Err(err).Msg("failed to unmarshal input")
						continue
					}
				}

				readableIdToData := make(map[string]map[string]interface{})

				for _, outputEvent := range parentData {
					outputMap := make(map[string]interface{})

					if outputEvent.Output != nil {
						err := json.Unmarshal(outputEvent.Output, &outputMap)

						if err != nil {
							d.l.Warn().Err(err).Msg("failed to unmarshal output")
							continue
						}
					}

					readableIdToData[outputEvent.StepReadableID] = outputMap
				}

				currInput.Parents = readableIdToData

				task.Input = currInput.Bytes()
			}
		}

		taskIdToData := make(map[int64]*sqlcv1.V1Task)

		for _, task := range bulkDatas {
			taskIdToData[task.ID] = task
		}

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

				for _, stepRunId := range stepRunIds {
					stepRunId := stepRunId

					innerEg.Go(func() error {
						task := taskIdToData[stepRunId]

						// if we've reached the context deadline, this should be requeued
						if ctx.Err() != nil {
							requeue(task)
							return nil
						}

						var multiErr error
						var success bool

						for i, w := range workers {
							err := w.StartTaskFromBulk(ctx, msg.TenantID, task)

							if err != nil {
								multiErr = multierror.Append(
									multiErr,
									fmt.Errorf("could not send action for task %s to worker %s (%d / %d): %w", sqlchelpers.UUIDToStr(task.ExternalID), workerId, i+1, len(workers), err),
								)
							} else {
								success = true
								break
							}
						}

						if success {
							return nil
						}

						requeue(task)

						return multiErr
					})
				}

				return innerEg.Wait()
			})
		}
	}

	outerErr := outerEg.Wait()

	if len(toRetry) > 0 {
		retryCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		retryGroup := errgroup.Group{}

		for _, _task := range toRetry {
			tenantId := msg.TenantID
			task := _task

			retryGroup.Go(func() error {
				msg, err := tasktypesv1.FailedTaskMessage(
					tenantId,
					task.ID,
					task.InsertedAt,
					sqlchelpers.UUIDToStr(task.ExternalID),
					sqlchelpers.UUIDToStr(task.WorkflowRunID),
					task.RetryCount,
					false,
					"Could not send task to worker",
					false,
				)

				if err != nil {
					return fmt.Errorf("could not create failed task message: %w", err)
				}

				queueutils.SleepWithExponentialBackoff(100*time.Millisecond, 5*time.Second, int(task.InternalRetryCount))

				err = d.mqv1.SendMessage(retryCtx, msgqueuev1.TASK_PROCESSING_QUEUE, msg)

				if err != nil {
					return fmt.Errorf("could not send failed task message: %w", err)
				}

				return nil
			})
		}

		if err := retryGroup.Wait(); err != nil {
			outerErr = multierror.Append(outerErr, fmt.Errorf("could not retry failed tasks: %w", err))
		}
	}

	return outerErr
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
