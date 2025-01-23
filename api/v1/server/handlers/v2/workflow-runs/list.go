package v2workflowruns

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
)

func (t *V2WorkflowRunsService) V2WorkflowRunsList(ctx echo.Context, request gen.V2WorkflowRunsListRequestObject) (gen.V2WorkflowRunsListResponseObject, error) {
	tenant := ctx.Get("tenant").(*db.TenantModel)

	// Replace this with OLAPRepository
	input, err := t.config.EngineRepository.WorkflowRun().GetWorkflowRunInputData(request.Tenant.String(), request.WorkflowRun.String())

	if err != nil {
		return nil, err
	}

	// Make transformer to transform clickhouse query result into this json object

	// Search for api errors to see how we handle errors in other cases
	return gen.V2WorkflowRunsList200JSONResponse(
		input,
	), nil
}
