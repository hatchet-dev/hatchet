package dispatcher

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/go-multierror"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/types/known/timestamppb"

	msgqueuev1 "github.com/hatchet-dev/hatchet/internal/msgqueue/v1"
	"github.com/hatchet-dev/hatchet/internal/queueutils"
	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	tasktypesv1 "github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/telemetry"
)

func (d *DispatcherImpl) handleTaskBulkAssignedTask(ctx context.Context, msg *msgqueuev1.Message) error {
	ctx, span := telemetry.NewSpanWithCarrier(ctx, "task-assigned-bulk", msg.OtelCarrier)
	defer span.End()

	// we set a timeout of 25 seconds because we don't want to hold the semaphore for longer than the visibility timeout (30 seconds)
	// on the worker
	ctx, cancel := context.WithTimeout(ctx, 25*time.Second)

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

		for _, batches := range innerMsg.WorkerBatches {
			for _, batch := range batches {
				taskIds = append(taskIds, batch.TaskIds...)
			}
		}

		bulkDatas, err := d.repov1.Tasks().ListTasks(ctx, msg.TenantID, taskIds)

		if err != nil {
			for _, task := range bulkDatas {
				if task != nil && task.Task != nil {
					requeue(task.Task)
				}
			}

			d.l.Error().Err(err).Msgf("could not bulk list step run data:")
			continue
		}

		tasksOnly := make([]*sqlcv1.V1Task, 0, len(bulkDatas))

		for _, task := range bulkDatas {
			if task != nil && task.Task != nil {
				tasksOnly = append(tasksOnly, task.Task)
			}
		}

		parentDataMap, err := d.repov1.Tasks().ListTaskParentOutputs(ctx, msg.TenantID, tasksOnly)

		if err != nil {
			for _, task := range bulkDatas {
				if task != nil && task.Task != nil {
					requeue(task.Task)
				}
			}

			d.l.Error().Err(err).Msgf("could not list parent data for %d tasks", len(bulkDatas))
			continue
		}

		retrievePayloadOpts := make([]v1.RetrievePayloadOpts, 0, len(tasksOnly))

		for _, taskWithRuntime := range bulkDatas {
			if taskWithRuntime == nil || taskWithRuntime.Task == nil {
				continue
			}

			task := taskWithRuntime.Task

			retrievePayloadOpts = append(retrievePayloadOpts, v1.RetrievePayloadOpts{
				Id:         task.ID,
				InsertedAt: task.InsertedAt,
				Type:       sqlcv1.V1PayloadTypeTASKINPUT,
				TenantId:   task.TenantID,
			})
		}

		inputs, err := d.repov1.Payloads().Retrieve(ctx, nil, retrievePayloadOpts...)

		if err != nil {
			d.l.Error().Err(err).Msgf("could not bulk retrieve inputs for %d tasks", len(bulkDatas))
			for _, taskWithRuntime := range bulkDatas {
				if taskWithRuntime != nil && taskWithRuntime.Task != nil {
					requeue(taskWithRuntime.Task)
				}
			}
		}

		// this is to avoid a nil pointer dereference in the code below
		if inputs == nil {
			inputs = make(map[v1.RetrievePayloadOpts][]byte)
		}

		for _, taskWithRuntime := range bulkDatas {
			if taskWithRuntime == nil || taskWithRuntime.Task == nil {
				continue
			}

			task := taskWithRuntime.Task
			input, ok := inputs[v1.RetrievePayloadOpts{
				Id:         task.ID,
				InsertedAt: task.InsertedAt,
				Type:       sqlcv1.V1PayloadTypeTASKINPUT,
				TenantId:   task.TenantID,
			}]

			if !ok {
				// If the input wasn't found in the payload store,
				// fall back to the input stored on the task itself.
				d.l.Error().Msgf("handleTaskBulkAssignedTask-1: task %s with ID %d and inserted_at %s has empty payload, falling back to input", task.ExternalID.String(), task.ID, task.InsertedAt.Time)
				input = task.Input
			}

			if parentData, ok := parentDataMap[task.ID]; ok {
				currInput := &v1.V1StepRunData{}

				if input != nil {
					err := json.Unmarshal(input, currInput)

					if err != nil {
						d.l.Warn().Err(err).Msg("failed to unmarshal input")
						continue
					}
				}

				readableIdToData := make(map[string]map[string]interface{})

				for _, outputEvent := range parentData {
					outputMap := make(map[string]interface{})

					if len(outputEvent.Output) > 0 {
						err := json.Unmarshal(outputEvent.Output, &outputMap)

						if err != nil {
							d.l.Warn().Err(err).Msg("failed to unmarshal output")
							continue
						}
					}

					readableIdToData[outputEvent.StepReadableID] = outputMap
				}

				currInput.Parents = readableIdToData

				inputs[v1.RetrievePayloadOpts{
					Id:         task.ID,
					InsertedAt: task.InsertedAt,
					Type:       sqlcv1.V1PayloadTypeTASKINPUT,
					TenantId:   task.TenantID,
				}] = currInput.Bytes()
			}
		}

		taskIdToData := make(map[int64]*v1.V1TaskWithPayload)

		for _, taskWithRuntime := range bulkDatas {
			if taskWithRuntime == nil || taskWithRuntime.Task == nil {
				continue
			}

			task := taskWithRuntime.Task
			input, ok := inputs[v1.RetrievePayloadOpts{
				Id:         task.ID,
				InsertedAt: task.InsertedAt,
				Type:       sqlcv1.V1PayloadTypeTASKINPUT,
				TenantId:   task.TenantID,
			}]

			if !ok {
				// If the input wasn't found in the payload store,
				// fall back to the input stored on the task itself.
				d.l.Error().Msgf("handleTaskBulkAssignedTask-2: task %s witth id %d and inserted_at %s has empty payload, falling back to input", task.ExternalID.String(), task.ID, task.InsertedAt.Time)
				input = task.Input
			}

			taskIdToData[task.ID] = &v1.V1TaskWithPayload{
				V1Task:  task,
				Runtime: taskWithRuntime.Runtime,
				Payload: input,
			}
		}

		for workerId, batches := range innerMsg.WorkerBatches {
			workerId := workerId

			outerEg.Go(func() error {
				totalTasks := 0

				for _, batch := range batches {
					totalTasks += len(batch.TaskIds)
				}

				d.l.Debug().Msgf("worker %s has %d step runs", workerId, totalTasks)

				// get the worker for this task
				workers, err := d.workers.Get(workerId)

				if err != nil && !errors.Is(err, ErrWorkerNotFound) {
					return fmt.Errorf("could not get worker: %w", err)
				}

				innerEg := errgroup.Group{}

				for _, batch := range batches {
					// If this is a batched assignment and includes start metadata, emit START_BATCH
					// before sending any of the individual tasks for the batch.
					if batch.BatchID != "" && batch.StartBatch != nil {
						if err := d.sendBatchStartFromPayload(ctx, msg.TenantID, batch.StartBatch); err != nil {
							// Don't fail the whole batch assignment; tasks will be requeued via the normal path below
							// if they cannot be sent. This just logs and continues.
							d.l.Warn().Err(err).Msgf("could not send embedded batch start for batch %s", batch.BatchID)
						}
					}

					for _, taskId := range batch.TaskIds {
						innerEg.Go(func() error {
							task := taskIdToData[taskId]

							// if we've reached the context deadline, this should be requeued
							if ctx.Err() != nil {
								requeue(task.V1Task)
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
								msg, err := tasktypesv1.MonitoringEventMessageFromInternal(
									task.TenantID.String(),
									tasktypesv1.CreateMonitoringEventPayload{
										TaskId:         task.ID,
										RetryCount:     task.RetryCount,
										WorkerId:       &workerId,
										EventType:      sqlcv1.V1EventTypeOlapSENTTOWORKER,
										EventTimestamp: time.Now().UTC(),
										EventMessage:   "Sent task run to the assigned worker",
									},
								)

								if err != nil {
									multiErr = multierror.Append(
										multiErr,
										fmt.Errorf("could not create monitoring event for task %d: %w", task.ID, err),
									)
								} else {
									defer d.pubBuffer.Pub(ctx, msgqueuev1.OLAP_QUEUE, msg, false)
								}

								return nil
							}

							requeue(task.V1Task)

							return multiErr
						})
					}
				}

				return innerEg.Wait()
			})
		}
	}

	// we spawn a goroutine to wait for the outer error group to finish and handle retries, because sending over the gRPC stream
	// can occasionally take a long time and we don't want to block the RabbitMQ queue processing
	go func() {
		defer cancel()

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

		if outerErr != nil {
			d.l.Error().Err(outerErr).Msg("failed to handle task assigned bulk message")
		}
	}()

	return nil
}

func (d *DispatcherImpl) sendBatchStartFromPayload(ctx context.Context, msgTenantId string, payload *tasktypesv1.StartBatchTaskPayload) error {
	if payload == nil {
		return nil
	}

	tenantId := payload.TenantId
	if tenantId == "" {
		tenantId = msgTenantId
	}

	if payload.BatchId == "" {
		return fmt.Errorf("batch start payload missing batch id")
	}

	if payload.ActionId == "" {
		return fmt.Errorf("batch start payload missing action id for batch %s", payload.BatchId)
	}

	workers, err := d.workers.Get(payload.WorkerId)
	if err != nil {
		if errors.Is(err, ErrWorkerNotFound) {
			// If the worker isn't connected, ignore (the tasks will be retried separately).
			return nil
		}
		return fmt.Errorf("could not get worker for batch %s: %w", payload.BatchId, err)
	}

	triggerTime := payload.TriggerTime
	if triggerTime.IsZero() {
		triggerTime = time.Now().UTC()
	}

	expectedSize := int32(payload.ExpectedSize)
	if expectedSize < 0 {
		expectedSize = 0
	}

	batchID := payload.BatchId
	batchStart := &contracts.BatchStartPayload{
		TriggerTime:  timestamppb.New(triggerTime),
		ExpectedSize: expectedSize,
	}

	if payload.TriggerReason != "" {
		batchStart.TriggerReason = payload.TriggerReason
	}

	action := &contracts.AssignedAction{
		TenantId:   tenantId,
		ActionType: contracts.ActionType_START_BATCH,
		ActionId:   payload.ActionId,
		BatchStart: batchStart,
	}

	action.BatchId = &batchID

	if strings.TrimSpace(payload.BatchKey) != "" {
		key := strings.TrimSpace(payload.BatchKey)
		action.BatchKey = &key
	}

	var sendErr error
	var success bool

	for i, w := range workers {
		if err := w.StartBatch(ctx, action); err != nil {
			sendErr = multierror.Append(sendErr, fmt.Errorf("could not send batch start to worker %s (%d): %w", payload.WorkerId, i, err))
		} else {
			success = true
			break
		}
	}

	if !success {
		return sendErr
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

	taskIdsToRetryCounts := make(map[int64][]int32)

	for _, innerMsg := range msgs {
		taskIdsToRetryCounts[innerMsg.TaskId] = append(taskIdsToRetryCounts[innerMsg.TaskId], innerMsg.RetryCount)
	}

	taskIds := make([]int64, 0)
	for taskId := range taskIdsToRetryCounts {
		taskIds = append(taskIds, taskId)
	}

	bulkDatas, err := d.repov1.Tasks().ListTasks(ctx, msg.TenantID, taskIds)

	if err != nil {
		return fmt.Errorf("could not list tasks: %w", err)
	}

	taskIdsToTasks := make(map[int64]*v1.TaskWithRuntime)

	for _, taskWithRuntime := range bulkDatas {
		if taskWithRuntime == nil || taskWithRuntime.Task == nil {
			continue
		}

		taskIdsToTasks[taskWithRuntime.Task.ID] = taskWithRuntime
	}

	// group by worker id
	workerIdToTasks := make(map[string][]*sqlcv1.V1Task)

	for _, msg := range msgs {
		if _, ok := workerIdToTasks[msg.WorkerId]; !ok {
			workerIdToTasks[msg.WorkerId] = []*sqlcv1.V1Task{}
		}

		taskWithRuntime, ok := taskIdsToTasks[msg.TaskId]

		if !ok {
			d.l.Warn().Msgf("task %d not found", msg.TaskId)
			continue
		}

		if taskWithRuntime == nil {
			continue
		}

		workerIdToTasks[msg.WorkerId] = append(workerIdToTasks[msg.WorkerId], taskWithRuntime.Task)
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
				retryCounts, ok := taskIdsToRetryCounts[task.ID]

				if !ok {
					d.l.Warn().Msgf("task %d not found in retry counts", task.ID)
					continue
				}

				for _, retryCount := range retryCounts {
					err = w.CancelTask(ctx, msg.TenantID, task, retryCount)

					if err != nil {
						multiErr = multierror.Append(multiErr, fmt.Errorf("could not send job to worker: %w", err))
					}
				}
			}
		}
	}

	return multiErr
}

func (d *DispatcherImpl) handleBatchStartTask(ctx context.Context, msg *msgqueuev1.Message) error {
	ctx, span := telemetry.NewSpanWithCarrier(ctx, "batch-start", msg.OtelCarrier)
	defer span.End()

	payloads := msgqueuev1.JSONConvert[tasktypesv1.StartBatchTaskPayload](msg.Payloads)

	var result error

	for _, payload := range payloads {
		if err := d.sendBatchStartFromPayload(ctx, msg.TenantID, payload); err != nil {
			if errors.Is(err, ErrWorkerNotFound) {
				continue
			}
			result = multierror.Append(result, err)
		}
	}

	return result
}
