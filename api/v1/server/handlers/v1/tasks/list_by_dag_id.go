package tasks

import (
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"

	transformers "github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers/v1"
)

func (t *TasksService) V1DagListTasks(ctx echo.Context, request gen.V1DagListTasksRequestObject) (gen.V1DagListTasksResponseObject, error) {
	tenantId := request.Params.Tenant.String()
	dagIds := request.Params.DagIds

	pguuids := make([]uuid.UUID, 0)
	for _, dagId := range dagIds {
		pguuids = append(pguuids, dagId)
	}

	tasks, taskIdToDagExternalId, err := t.config.V1.OLAP().ListTasksByDAGId(
		ctx.Request().Context(),
		tenantId,
		pguuids,
		false,
	)

	if err != nil {
		return nil, err
	}

	result := transformers.ToDagChildren(tasks, taskIdToDagExternalId)

	// Search for api errors to see how we handle errors in other cases
	return gen.V1DagListTasks200JSONResponse(
		result,
	), nil
}
