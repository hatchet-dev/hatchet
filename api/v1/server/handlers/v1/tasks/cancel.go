package tasks

import (
	"context"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/internal/services/admin/contracts/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

func (t *TasksService) V1TaskCancel(ctx echo.Context, request gen.V1TaskCancelRequestObject) (gen.V1TaskCancelResponseObject, error) {
	tenant := ctx.Get("tenant").(*dbsqlc.Tenant)

	var taskIdRetryCounts []*contracts.TaskIdRetryCount
	var err error

	grpcReq := &contracts.CancelTasksRequest{}

	if request.Body.ExternalIds != nil {
		taskIdRetryCounts, err = t.populateTasksFromExternalIds(ctx.Request().Context(), tenant, *request.Body.ExternalIds)

		if err != nil {
			return nil, err
		}

		grpcReq.Tasks = taskIdRetryCounts
	}

	if request.Body.Filter != nil {
		filter := &contracts.CancelTasksRequestFilter{
			Since: timestamppb.New(request.Body.Filter.Since),
		}

		if request.Body.Filter.Until != nil {
			filter.Until = timestamppb.New(*request.Body.Filter.Until)
		}

		if request.Body.Filter.Statuses != nil {
			filter.Statuses = make([]string, len(*request.Body.Filter.Statuses))

			for i, status := range *request.Body.Filter.Statuses {
				filter.Statuses[i] = string(status)
			}
		}

		if request.Body.Filter.WorkflowIds != nil {
			filter.WorkflowIds = make([]string, len(*request.Body.Filter.WorkflowIds))

			for i, id := range *request.Body.Filter.WorkflowIds {
				filter.WorkflowIds[i] = id.String()
			}
		}

		if request.Body.Filter.AdditionalMetadata != nil {
			filter.AdditionalMetadata = make([]string, len(*request.Body.Filter.AdditionalMetadata))

			copy(filter.AdditionalMetadata, *request.Body.Filter.AdditionalMetadata)
		}

		grpcReq.Filter = filter
	}

	_, err = t.proxyCancel.Do(
		ctx.Request().Context(),
		tenant,
		grpcReq,
	)

	if err != nil {
		return nil, err
	}

	return gen.V1TaskCancel200Response{}, nil
}

func (t *TasksService) populateTasksFromExternalIds(ctx context.Context, tenant *dbsqlc.Tenant, externalIds []uuid.UUID) ([]*contracts.TaskIdRetryCount, error) {
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	externalIdStrings := make([]string, len(externalIds))

	for i, id := range externalIds {
		externalIdStrings[i] = id.String()
	}

	// get the task ids from the external ids
	tasks, err := t.config.V1.OLAP().ListTasksByExternalIds(ctx, tenantId, externalIdStrings)

	if err != nil {
		return nil, err
	}

	// construct the list of tasks to cancel
	tasksToCancel := make([]*contracts.TaskIdRetryCount, len(tasks))

	for i, task := range tasks {
		tasksToCancel[i] = &contracts.TaskIdRetryCount{
			TaskId:     task.ID,
			RetryCount: task.RetryCount,
		}
	}

	return tasksToCancel, nil
}
