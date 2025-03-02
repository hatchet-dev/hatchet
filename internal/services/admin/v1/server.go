package v1

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	msgqueue "github.com/hatchet-dev/hatchet/internal/msgqueue/v1"
	contracts "github.com/hatchet-dev/hatchet/internal/services/admin/contracts/v1"
	tasktypes "github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"
)

func (a *AdminServiceImpl) CancelTasks(ctx context.Context, req *contracts.CancelTasksRequest) (*contracts.CancelTasksResponse, error) {
	tenant := ctx.Value("tenant").(*dbsqlc.Tenant)

	tasksToCancel := []v1.TaskIdRetryCount{}

	// for tasks passed in directly, make sure the tenant id and retry counts match
	if len(req.Tasks) > 0 {
		taskIds := make([]int64, len(req.Tasks))
		retryCounts := make([]int32, len(req.Tasks))

		for i, task := range req.Tasks {
			taskIds[i] = task.TaskId
			retryCounts[i] = task.RetryCount
		}

		tasks, err := a.repo.Tasks().ListTasks(ctx, sqlchelpers.UUIDToStr(tenant.ID), taskIds)

		if err != nil {
			return nil, err
		}

		for i, task := range tasks {
			// we'd like to make sure the retry counts match so that there wasn't another replay or cancellation
			// which occurred concurrently. if there was, this is a no-op.
			if task.RetryCount == retryCounts[i] {
				tasksToCancel = append(tasksToCancel, v1.TaskIdRetryCount{
					Id:         task.ID,
					RetryCount: task.RetryCount,
				})
			}
		}
	}

	if req.Filter != nil {
		var (
			statuses = []sqlcv1.V1ReadableStatusOlap{
				sqlcv1.V1ReadableStatusOlapQUEUED,
				sqlcv1.V1ReadableStatusOlapRUNNING,
				sqlcv1.V1ReadableStatusOlapFAILED,
				sqlcv1.V1ReadableStatusOlapCOMPLETED,
				sqlcv1.V1ReadableStatusOlapCANCELLED,
			}
			since       = req.Filter.Since.AsTime()
			until       *time.Time
			workflowIds       = []uuid.UUID{}
			limit       int64 = 20000
			offset      int64
		)

		if len(req.Filter.Statuses) > 0 {
			statuses = []sqlcv1.V1ReadableStatusOlap{}

			for _, status := range req.Filter.Statuses {
				statuses = append(statuses, sqlcv1.V1ReadableStatusOlap(status))
			}
		}

		if len(req.Filter.WorkflowIds) > 0 {
			for _, id := range req.Filter.WorkflowIds {
				workflowIds = append(workflowIds, uuid.MustParse(id))
			}
		}

		if req.Filter.Until != nil {
			t := req.Filter.Until.AsTime()
			until = &t
		}

		var additionalMetadataFilters map[string]interface{}

		if len(req.Filter.AdditionalMetadata) > 0 {
			additionalMetadataFilters := make(map[string]interface{})
			for _, v := range req.Filter.AdditionalMetadata {
				kv_pairs := strings.Split(v, ":")
				if len(kv_pairs) == 2 {
					additionalMetadataFilters[kv_pairs[0]] = kv_pairs[1]
				} else {
					return nil, status.Errorf(codes.InvalidArgument, "invalid additional metadata filter: %s", v)
				}
			}
		}

		opts := v1.ListWorkflowRunOpts{
			CreatedAfter:       since,
			FinishedBefore:     until,
			Statuses:           statuses,
			WorkflowIds:        workflowIds,
			Limit:              limit,
			Offset:             offset,
			AdditionalMetadata: additionalMetadataFilters,
		}

		runs, _, err := a.repo.OLAP().ListWorkflowRuns(ctx, sqlchelpers.UUIDToStr(tenant.ID), opts)

		if err != nil {
			return nil, err
		}

		runExternalIds := make([]string, len(runs))

		for i, run := range runs {
			runExternalIds[i] = sqlchelpers.UUIDToStr(run.ExternalID)
		}

		tasks, err := a.repo.OLAP().ListTasksByExternalIds(ctx, sqlchelpers.UUIDToStr(tenant.ID), runExternalIds)

		if err != nil {
			return nil, err
		}

		for _, task := range tasks {
			tasksToCancel = append(tasksToCancel, v1.TaskIdRetryCount{
				Id:         task.ID,
				RetryCount: task.RetryCount,
			})
		}
	}

	// send the payload to the tasks controller, and send the list of tasks back to the client
	toCancel := tasktypes.CancelTasksPayload{
		Tasks: tasksToCancel,
	}

	msg, err := msgqueue.NewTenantMessage(
		sqlchelpers.UUIDToStr(tenant.ID),
		"cancel-tasks",
		false,
		true,
		toCancel,
	)

	if err != nil {
		return nil, err
	}

	err = a.mq.SendMessage(ctx, msgqueue.TASK_PROCESSING_QUEUE, msg)

	if err != nil {
		return nil, err
	}

	respCancelledTasks := []*contracts.TaskIdRetryCount{}

	for _, task := range tasksToCancel {
		respCancelledTasks = append(respCancelledTasks, &contracts.TaskIdRetryCount{
			TaskId:     task.Id,
			RetryCount: task.RetryCount,
		})
	}

	return &contracts.CancelTasksResponse{
		CancelledTasks: respCancelledTasks,
	}, nil
}

func (a *AdminServiceImpl) ReplayTasks(ctx context.Context, req *contracts.ReplayTasksRequest) (*contracts.ReplayTasksResponse, error) {
	tenant := ctx.Value("tenant").(*dbsqlc.Tenant)

	tasksToReplay := []v1.TaskIdRetryCount{}

	// for tasks passed in directly, make sure the tenant id and retry counts match
	if len(req.Tasks) > 0 {
		taskIds := make([]int64, len(req.Tasks))
		retryCounts := make([]int32, len(req.Tasks))

		for i, task := range req.Tasks {
			taskIds[i] = task.TaskId
			retryCounts[i] = task.RetryCount
		}

		tasks, err := a.repo.Tasks().ListTasks(ctx, sqlchelpers.UUIDToStr(tenant.ID), taskIds)

		if err != nil {
			return nil, err
		}

		for _, task := range tasks {
			tasksToReplay = append(tasksToReplay, v1.TaskIdRetryCount{
				Id:         task.ID,
				RetryCount: task.RetryCount,
			})
		}
	}

	if req.Filter != nil {
		var (
			statuses = []sqlcv1.V1ReadableStatusOlap{
				sqlcv1.V1ReadableStatusOlapQUEUED,
				sqlcv1.V1ReadableStatusOlapRUNNING,
				sqlcv1.V1ReadableStatusOlapFAILED,
				sqlcv1.V1ReadableStatusOlapCOMPLETED,
				sqlcv1.V1ReadableStatusOlapCANCELLED,
			}
			since       = req.Filter.Since.AsTime()
			until       *time.Time
			workflowIds       = []uuid.UUID{}
			limit       int64 = 20000
			offset      int64
		)

		if len(req.Filter.Statuses) > 0 {
			statuses = []sqlcv1.V1ReadableStatusOlap{}

			for _, status := range req.Filter.Statuses {
				statuses = append(statuses, sqlcv1.V1ReadableStatusOlap(status))
			}
		}

		if len(req.Filter.WorkflowIds) > 0 {
			for _, id := range req.Filter.WorkflowIds {
				workflowIds = append(workflowIds, uuid.MustParse(id))
			}
		}

		if req.Filter.Until != nil {
			t := req.Filter.Until.AsTime()
			until = &t
		}

		var additionalMetadataFilters map[string]interface{}

		if len(req.Filter.AdditionalMetadata) > 0 {
			additionalMetadataFilters := make(map[string]interface{})
			for _, v := range req.Filter.AdditionalMetadata {
				kv_pairs := strings.Split(v, ":")
				if len(kv_pairs) == 2 {
					additionalMetadataFilters[kv_pairs[0]] = kv_pairs[1]
				} else {
					return nil, status.Errorf(codes.InvalidArgument, "invalid additional metadata filter: %s", v)
				}
			}
		}

		opts := v1.ListWorkflowRunOpts{
			CreatedAfter:       since,
			FinishedBefore:     until,
			Statuses:           statuses,
			WorkflowIds:        workflowIds,
			Limit:              limit,
			Offset:             offset,
			AdditionalMetadata: additionalMetadataFilters,
		}

		runs, _, err := a.repo.OLAP().ListWorkflowRuns(ctx, sqlchelpers.UUIDToStr(tenant.ID), opts)

		if err != nil {
			return nil, err
		}

		runExternalIds := make([]string, len(runs))

		for i, run := range runs {
			runExternalIds[i] = sqlchelpers.UUIDToStr(run.ExternalID)
		}

		tasks, err := a.repo.OLAP().ListTasksByExternalIds(ctx, sqlchelpers.UUIDToStr(tenant.ID), runExternalIds)

		if err != nil {
			return nil, err
		}

		for _, task := range tasks {
			tasksToReplay = append(tasksToReplay, v1.TaskIdRetryCount{
				Id:         task.ID,
				RetryCount: task.RetryCount,
			})
		}
	}

	// send the payload to the tasks controller, and send the list of tasks back to the client
	toReplay := tasktypes.ReplayTasksPayload{
		Tasks: tasksToReplay,
	}

	msg, err := msgqueue.NewTenantMessage(
		sqlchelpers.UUIDToStr(tenant.ID),
		"replay-tasks",
		false,
		true,
		toReplay,
	)

	if err != nil {
		return nil, err
	}

	err = a.mq.SendMessage(ctx, msgqueue.TASK_PROCESSING_QUEUE, msg)

	if err != nil {
		return nil, err
	}

	respReplayedTasks := []*contracts.TaskIdRetryCount{}

	for _, task := range tasksToReplay {
		respReplayedTasks = append(respReplayedTasks, &contracts.TaskIdRetryCount{
			TaskId:     task.Id,
			RetryCount: task.RetryCount,
		})
	}

	return &contracts.ReplayTasksResponse{
		ReplayedTasks: respReplayedTasks,
	}, nil
}
