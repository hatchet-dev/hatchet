package tasks

import (
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers/v2"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
)

func (t *TasksService) V2TaskList(ctx echo.Context, request gen.V2TaskListRequestObject) (gen.V2TaskListResponseObject, error) {
	tenant := ctx.Get("tenant").(*db.TenantModel)

	var (
		statuses = []gen.V2TaskStatus{
			gen.V2TaskStatusCANCELLED,
			gen.V2TaskStatusCOMPLETED,
			gen.V2TaskStatusFAILED,
			gen.V2TaskStatusQUEUED,
			gen.V2TaskStatusRUNNING,
		}
		since             = request.Params.Since
		workflowIds       = []uuid.UUID{}
		limit       int64 = 50
		offset      int64 = 0
		workerId    *uuid.UUID
	)

	if request.Params.Statuses != nil {
		if len(*request.Params.Statuses) > 0 {
			statuses = *request.Params.Statuses
		}
	}

	if request.Params.Limit != nil {
		limit = *request.Params.Limit
	}

	if request.Params.Offset != nil {
		offset = *request.Params.Offset
	}

	if request.Params.WorkflowIds != nil {
		workflowIds = *request.Params.WorkflowIds
	}

	if request.Params.WorkerId != nil {
		workerId = request.Params.WorkerId
	}

	tasks, total, err := t.config.EngineRepository.OLAP().ListTaskRuns(
		ctx.Request().Context(),
		tenant.ID,
		repository.ListTaskRunOpts{
			CreatedAfter: since,
			Statuses:     statuses,
			WorkflowIds:  workflowIds,
			WorkerId:     workerId,
			Limit:        limit,
			Offset:       offset,
		},
	)

	if err != nil {
		return nil, err
	}

	result := transformers.ToTaskSummaryMany(tasks, total, limit, offset)

	// Search for api errors to see how we handle errors in other cases
	return gen.V2TaskList200JSONResponse(
		result,
	), nil
}
