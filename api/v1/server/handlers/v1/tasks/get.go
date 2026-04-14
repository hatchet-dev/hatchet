package tasks

import (
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v4"

	v1handlers "github.com/hatchet-dev/hatchet/api/v1/server/handlers/v1"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"

	transformers "github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers/v1"
)

func (t *TasksService) V1TaskGet(ctx echo.Context, request gen.V1TaskGetRequestObject) (gen.V1TaskGetResponseObject, error) {
	taskInterface := ctx.Get("task")

	if taskInterface == nil {
		return gen.V1TaskGet404JSONResponse{
			Errors: []gen.APIError{
				{
					Description: "task not found",
				},
			},
		}, nil
	}

	task, ok := taskInterface.(*sqlcv1.V1TasksOlap)

	if !ok {
		return nil, echo.NewHTTPError(500, "Task type assertion failed")
	}

	tenant := ctx.Get("tenant").(*sqlcv1.Tenant)

	if ts := task.InsertedAt; ts.Valid && v1handlers.IsBeforeRetention(ts.Time, tenant.DataRetentionPeriod) {
		return nil, echo.NewHTTPError(http.StatusGone, "task is outside the data retention window")
	}

	attempt := request.Params.Attempt

	var retryCount *int

	if attempt != nil {
		count := *attempt - 1
		retryCount = &count
	}

	if retryCount != nil && *retryCount < 0 {
		return nil, echo.NewHTTPError(400, "Attempt must be greater than 0")
	}

	taskWithData, workflowRunExternalId, err := t.config.V1.OLAP().ReadTaskRunData(
		ctx.Request().Context(),
		task.TenantID,
		task.ID,
		task.InsertedAt,
		retryCount,
	)

	if err != nil {
		return nil, err
	}

	var workflowVersion *sqlcv1.GetWorkflowVersionByIdRow

	workflowVersion, _, _, _, _, _, err = t.config.V1.Workflows().GetWorkflowVersionWithTriggers(ctx.Request().Context(), task.TenantID, taskWithData.WorkflowVersionID)

	// a workflow version or the workflow itself may be deleted but we still want to return the task details
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	result := transformers.ToTask(taskWithData, workflowRunExternalId, workflowVersion)

	return gen.V1TaskGet200JSONResponse(
		result,
	), nil
}
