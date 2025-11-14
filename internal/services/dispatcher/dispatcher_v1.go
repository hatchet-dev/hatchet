package dispatcher

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/hashicorp/go-multierror"
	"go.opentelemetry.io/otel/codes"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"

	msgqueuev1 "github.com/hatchet-dev/hatchet/internal/msgqueue/v1"
	"github.com/hatchet-dev/hatchet/internal/queueutils"
	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	tasktypesv1 "github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/telemetry"
)

func (worker *subscribedWorker) StartTaskFromBulk(
	ctx context.Context,
	tenantId string,
	task *v1.V1TaskWithPayload,
) error {
	ctx, span := telemetry.NewSpan(ctx, "start-step-run-from-bulk") // nolint:ineffassign
	defer span.End()

	inputBytes := []byte{}

	if task.Payload != nil {
		inputBytes = task.Payload
	}

	action := populateAssignedAction(tenantId, task.V1Task, task.RetryCount)

	action.ActionType = contracts.ActionType_START_STEP_RUN
	action.ActionPayload = string(inputBytes)

	return worker.sendToWorker(ctx, action)
}

func (worker *subscribedWorker) sendToWorker(
	ctx context.Context,
	action *contracts.AssignedAction,
) error {
	ctx, span := telemetry.NewSpan(ctx, "send-to-worker") // nolint:ineffassign
	defer span.End()

	telemetry.WithAttributes(
		span,
		telemetry.AttributeKV{
			Key:   "worker.id",
			Value: worker.workerId,
		},
	)

	telemetry.WithAttributes(
		span,
		telemetry.AttributeKV{
			Key:   "payload.size_bytes",
			Value: len(action.ActionPayload),
		},
	)

	_, encodeSpan := telemetry.NewSpan(ctx, "encode-action")

	msg := &grpc.PreparedMsg{}
	err := msg.Encode(worker.stream, action)

	if err != nil {
		encodeSpan.RecordError(err)
		encodeSpan.End()
		return fmt.Errorf("could not encode action: %w", err)
	}

	encodeSpan.End()

	lockBegin := time.Now()

	_, lockSpan := telemetry.NewSpan(ctx, "acquire-worker-stream-lock")

	worker.sendMu.Lock()
	defer worker.sendMu.Unlock()

	lockSpan.End()

	telemetry.WithAttributes(span, telemetry.AttributeKV{
		Key:   "lock.duration_ms",
		Value: time.Since(lockBegin).Milliseconds(),
	})

	_, streamSpan := telemetry.NewSpan(ctx, "send-worker-stream")
	defer streamSpan.End()

	sendMsgBegin := time.Now()

	err = worker.stream.SendMsg(msg)

	if err != nil {
		span.RecordError(err)
	}

	if time.Since(sendMsgBegin) > 50*time.Millisecond {
		span.SetStatus(codes.Error, "flow control detected")
		span.RecordError(fmt.Errorf("send took too long, we may be in flow control: %s", time.Since(sendMsgBegin)))
	}

	return err
}

func (worker *subscribedWorker) CancelTask(
	ctx context.Context,
	tenantId string,
	task *sqlcv1.V1Task,
	retryCount int32,
) error {
	ctx, span := telemetry.NewSpan(ctx, "cancel-task") // nolint:ineffassign
	defer span.End()

	action := populateAssignedAction(tenantId, task, retryCount)

	action.ActionType = contracts.ActionType_CANCEL_STEP_RUN

	worker.sendMu.Lock()
	defer worker.sendMu.Unlock()

	return worker.stream.Send(action)
}

func populateAssignedAction(tenantID string, task *sqlcv1.V1Task, retryCount int32) *contracts.AssignedAction {
	workflowId := sqlchelpers.UUIDToStr(task.WorkflowID)
	workflowVersionId := sqlchelpers.UUIDToStr(task.WorkflowVersionID)

	action := &contracts.AssignedAction{
		TenantId:          tenantID,
		JobId:             sqlchelpers.UUIDToStr(task.StepID), // FIXME
		JobName:           task.StepReadableID,
		JobRunId:          sqlchelpers.UUIDToStr(task.ExternalID), // FIXME
		StepId:            sqlchelpers.UUIDToStr(task.StepID),
		StepRunId:         sqlchelpers.UUIDToStr(task.ExternalID),
		ActionId:          task.ActionID,
		StepName:          task.StepReadableID,
		WorkflowRunId:     sqlchelpers.UUIDToStr(task.WorkflowRunID),
		RetryCount:        retryCount,
		Priority:          task.Priority.Int32,
		WorkflowId:        &workflowId,
		WorkflowVersionId: &workflowVersionId,
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

		retrievePayloadOpts := make([]v1.RetrievePayloadOpts, len(bulkDatas))

		for i, task := range bulkDatas {
			retrievePayloadOpts[i] = v1.RetrievePayloadOpts{
				Id:         task.ID,
				InsertedAt: task.InsertedAt,
				Type:       sqlcv1.V1PayloadTypeTASKINPUT,
				TenantId:   task.TenantID,
			}
		}

		inputs, err := d.repov1.Payloads().Retrieve(ctx, nil, retrievePayloadOpts...)

		if err != nil {
			d.l.Error().Err(err).Msgf("could not bulk retrieve inputs for %d tasks", len(bulkDatas))

			for _, task := range bulkDatas {
				requeue(task)
			}
		}

		// this is to avoid a nil pointer dereference in the code below
		if inputs == nil {
			inputs = make(map[v1.RetrievePayloadOpts][]byte)
		}

		for _, task := range bulkDatas {
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

		for _, task := range bulkDatas {
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
				Payload: input,
			}
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
							return nil
						}

						requeue(task.V1Task)

						return multiErr
					})
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

		if !ok {
			d.l.Warn().Msgf("task %d not found in retry counts", msg.TaskId)
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
