package dispatcher

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/go-multierror"
	"golang.org/x/sync/errgroup"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/internal/queueutils"
	tasktypesv1 "github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes/v1"
	"github.com/hatchet-dev/hatchet/internal/syncx"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/telemetry"

	schedulingv1 "github.com/hatchet-dev/hatchet/pkg/scheduling/v1"
)

func (d *DispatcherImpl) handleTaskBulkAssignedTask(ctx context.Context, msg *msgqueue.Message) error {
	ctx, span := telemetry.NewSpanWithCarrier(ctx, "task-assigned-bulk", msg.OtelCarrier)
	defer span.End()

	// we set a timeout of 25 seconds because we don't want to hold the semaphore for longer than the visibility timeout (30 seconds)
	// on the worker
	ctx, cancel := context.WithTimeout(ctx, 25*time.Second)

	msgs := msgqueue.JSONConvert[tasktypesv1.TaskAssignedBulkTaskPayload](msg.Payloads)
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

		taskIdToData, err := d.populateTaskData(ctx, requeue, msg.TenantID, taskIds)

		if err != nil {
			// we've already handled the requeue in populateTaskData, and we've logged the error, so we just continue
			continue
		}

		for workerId, taskIds := range innerMsg.WorkerIdToTaskIds {
			workerId := workerId

			outerEg.Go(func() error {
				return d.sendTasksToWorker(ctx, requeue, msg.TenantID, workerId, taskIds, taskIdToData)
			})
		}
	}

	// we spawn a goroutine to wait for the outer error group to finish and handle retries, because sending over the gRPC stream
	// can occasionally take a long time and we don't want to block the RabbitMQ queue processing
	go func() {
		defer cancel()

		outerErr := outerEg.Wait()

		if err := d.handleRetries(ctx, msg.TenantID, toRetry); err != nil {
			outerErr = multierror.Append(outerErr, fmt.Errorf("could not retry failed tasks: %w", err))
		}

		if outerErr != nil {
			d.l.Error().Err(outerErr).Msg("failed to handle task assigned bulk message")
		}
	}()

	return nil
}

func (d *DispatcherImpl) GetLocalWorkerIds() map[uuid.UUID]struct{} {
	workerIds := make(map[uuid.UUID]struct{})

	d.workers.Range(func(workerId uuid.UUID, value *syncx.Map[string, *subscribedWorker]) bool {
		workerIds[workerId] = struct{}{}

		return true
	})

	return workerIds
}

// Note: this is very similar to handleTaskBulkAssignedTask, with some differences in what's sync vs run in a goroutine
// In this method, we wait until all tasks have been sent to the worker before returning
func (d *DispatcherImpl) HandleLocalAssignments(ctx context.Context, tenantId, workerId uuid.UUID, tasks []*schedulingv1.AssignedItemWithTask) error {
	ctx, span := telemetry.NewSpan(ctx, "DispatcherImpl.HandleLocalAssignments")
	defer span.End()

	// we set a timeout of 25 seconds because we don't want to hold the semaphore for longer than the visibility timeout (30 seconds)
	// on the worker
	ctx, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()

	toRetry := []*sqlcv1.V1Task{}
	toRetryMu := sync.Mutex{}

	requeue := func(task *sqlcv1.V1Task) {
		toRetryMu.Lock()
		toRetry = append(toRetry, task)
		toRetryMu.Unlock()
	}

	// we already have payloads; no lookups necessary. we can just send them to the worker
	taskIdToData := make(map[int64]*V1TaskWithPayloadAndInvocationCount)
	taskIds := make([]int64, 0, len(tasks))

	getDurableInvocationCountOpts := make([]v1.IdInsertedAt, 0)

	for _, assigned := range tasks {
		taskIdToData[assigned.Task.ID] = &V1TaskWithPayloadAndInvocationCount{
			V1TaskWithPayload: assigned.Task,
		}
		taskIds = append(taskIds, assigned.Task.ID)

		if assigned.Task.IsDurable.Valid && assigned.Task.IsDurable.Bool {
			getDurableInvocationCountOpts = append(getDurableInvocationCountOpts, v1.IdInsertedAt{
				ID:         assigned.Task.ID,
				InsertedAt: assigned.Task.InsertedAt,
			})
		}
	}

	if len(getDurableInvocationCountOpts) > 0 {
		invocationCounts, err := d.repov1.DurableEvents().GetDurableTaskInvocationCounts(ctx, tenantId, getDurableInvocationCountOpts)

		if err != nil {
			d.l.Error().Err(err).Msgf("could not get durable task invocation counts for %d tasks", len(getDurableInvocationCountOpts))
		} else {
			for _, assigned := range tasks {
				if assigned.Task.IsDurable.Valid && assigned.Task.IsDurable.Bool {
					count := invocationCounts[v1.IdInsertedAt{
						ID:         assigned.Task.ID,
						InsertedAt: assigned.Task.InsertedAt,
					}]
					taskIdToData[assigned.Task.ID].InvocationCount = count
				}
			}
		}
	}

	// this is one of the core differences from handleTaskBulkAssignedTask: we run this synchronously
	// so that we continue to use an optimistic scheduling semaphore slot until all tasks have been sent
	// to the worker
	err := d.sendTasksToWorker(ctx, requeue, tenantId, workerId, taskIds, taskIdToData)

	if retryErr := d.handleRetries(ctx, tenantId, toRetry); retryErr != nil {
		err = multierror.Append(err, fmt.Errorf("could not retry failed tasks: %w", retryErr))
	}

	return err
}

type V1TaskWithPayloadAndInvocationCount struct {
	*v1.V1TaskWithPayload
	InvocationCount *int32 // only used for durable tasks
}

func (d *DispatcherImpl) populateTaskData(
	ctx context.Context,
	requeue func(task *sqlcv1.V1Task),
	tenantId uuid.UUID,
	taskIds []int64,
) (map[int64]*V1TaskWithPayloadAndInvocationCount, error) {
	bulkDatas, err := d.repov1.Tasks().ListTasks(ctx, tenantId, taskIds)

	if err != nil {
		for _, task := range bulkDatas {
			requeue(task)
		}

		d.l.Error().Err(err).Msgf("could not bulk list step run data:")
		return nil, err
	}

	getInvocationCountOpts := make([]v1.IdInsertedAt, 0)

	for _, task := range bulkDatas {
		if task.IsDurable.Valid && task.IsDurable.Bool {
			getInvocationCountOpts = append(getInvocationCountOpts, v1.IdInsertedAt{
				ID:         task.ID,
				InsertedAt: task.InsertedAt,
			})
		}
	}

	invocationCounts := make(map[v1.IdInsertedAt]*int32)

	if len(getInvocationCountOpts) > 0 {
		invocationCounts, err = d.repov1.DurableEvents().GetDurableTaskInvocationCounts(ctx, tenantId, getInvocationCountOpts)

		if err != nil {
			for _, task := range bulkDatas {
				requeue(task)
			}

			d.l.Error().Err(err).Msgf("could not get durable task invocation counts for %d tasks", len(getInvocationCountOpts))
			return nil, err
		}
	}

	parentDataMap, err := d.repov1.Tasks().ListTaskParentOutputs(ctx, tenantId, bulkDatas)

	if err != nil {
		for _, task := range bulkDatas {
			requeue(task)
		}

		d.l.Error().Err(err).Msgf("could not list parent data for %d tasks", len(bulkDatas))
		return nil, err
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

	// FIXME: we should differentiate between a retryable error and a non-retryable error here;
	// for example, if we're hitting an S3 rate limit for payloads that exist in S3, we should retry;
	// however, if the payloads simply don't exist, we should fail the tasks instead of requeuing them.
	// The tasks will eventually fail but the extra retries are wasteful.
	if err != nil {
		for _, task := range bulkDatas {
			requeue(task)
		}

		d.l.Error().Err(err).Msgf("could not bulk retrieve inputs for %d tasks", len(bulkDatas))
		return nil, err
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

	taskIdToData := make(map[int64]*V1TaskWithPayloadAndInvocationCount)

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
			input = task.Input
		}

		invocationCount := invocationCounts[v1.IdInsertedAt{
			ID:         task.ID,
			InsertedAt: task.InsertedAt,
		}]

		taskIdToData[task.ID] = &V1TaskWithPayloadAndInvocationCount{
			&v1.V1TaskWithPayload{
				V1Task:  task,
				Payload: input,
			},
			invocationCount,
		}
	}

	return taskIdToData, nil
}

func (d *DispatcherImpl) sendTasksToWorker(
	ctx context.Context,
	requeue func(task *sqlcv1.V1Task),
	tenantId, workerId uuid.UUID,
	taskIds []int64,
	tasks map[int64]*V1TaskWithPayloadAndInvocationCount,
) error {
	// get the worker for this task
	workers, err := d.workers.Get(workerId)

	if err != nil && !errors.Is(err, ErrWorkerNotFound) {
		return fmt.Errorf("could not get worker: %w", err)
	}

	innerEg := errgroup.Group{}

	for _, taskId := range taskIds {
		task, ok := tasks[taskId]

		if !ok {
			d.l.Error().Msgf("task %d not found in task data map", taskId)
			continue
		}

		innerEg.Go(func() error {
			// if we've reached the context deadline, this should be requeued
			if ctx.Err() != nil {
				requeue(task.V1Task)
				return nil
			}

			var multiErr error
			var success bool

			for i, w := range workers {
				err := w.StartTaskFromBulk(ctx, tenantId, task.V1TaskWithPayload, task.InvocationCount)

				if err != nil {
					multiErr = multierror.Append(
						multiErr,
						fmt.Errorf("could not send action for task %s to worker %s (%d / %d): %w", task.ExternalID.String(), workerId, i+1, len(workers), err),
					)
				} else {
					success = true
					break
				}
			}

			if success {
				msg, err := tasktypesv1.MonitoringEventMessageFromInternal(
					task.TenantID,
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
					d.l.Error().Err(err).Int64("task_id", task.ID).Msg("could not create monitoring event")
				} else {
					defer func() {
						if err := d.pubBuffer.Pub(ctx, msgqueue.OLAP_QUEUE, msg, false); err != nil {
							d.l.Error().Err(err).Msg("could not publish monitoring event")
						}
					}()
				}

				return nil
			}

			requeue(task.V1Task)

			return multiErr
		})
	}

	return innerEg.Wait()
}

func (d *DispatcherImpl) handleRetries(
	ctx context.Context,
	tenantId uuid.UUID,
	toRetry []*sqlcv1.V1Task,
) error {
	if len(toRetry) == 0 {
		return nil
	}

	retryCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	retryGroup := errgroup.Group{}

	for _, _task := range toRetry {
		tenantId := tenantId
		task := _task

		retryGroup.Go(func() error {
			msg, err := tasktypesv1.FailedTaskMessage(
				tenantId,
				task.ID,
				task.InsertedAt,
				task.ExternalID,
				task.WorkflowRunID,
				task.RetryCount,
				false,
				"Could not send task to worker",
				false,
			)

			if err != nil {
				return fmt.Errorf("could not create failed task message: %w", err)
			}

			queueutils.SleepWithExponentialBackoff(100*time.Millisecond, 5*time.Second, int(task.InternalRetryCount))

			err = d.mqv1.SendMessage(retryCtx, msgqueue.TASK_PROCESSING_QUEUE, msg)

			if err != nil {
				return fmt.Errorf("could not send failed task message: %w", err)
			}

			return nil
		})
	}

	return retryGroup.Wait()
}

func (d *DispatcherImpl) handleTaskCancelled(ctx context.Context, msg *msgqueue.Message) error {
	ctx, span := telemetry.NewSpanWithCarrier(ctx, "tasks-cancelled", msg.OtelCarrier)
	defer span.End()

	// we set a timeout of 25 seconds because we don't want to hold the semaphore for longer than the visibility timeout (30 seconds)
	// on the worker
	ctx, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()

	msgs := msgqueue.JSONConvert[tasktypesv1.SignalTaskCancelledPayload](msg.Payloads)

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
	workerIdToTasks := make(map[uuid.UUID][]*sqlcv1.V1Task)

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
