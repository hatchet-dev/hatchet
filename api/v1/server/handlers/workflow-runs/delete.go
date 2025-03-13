package workflowruns

import (
	"fmt"

	"github.com/hashicorp/go-multierror"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/labstack/echo/v4"
)

func (t *WorkflowRunsService) WorkflowRunDelete(ctx echo.Context, request gen.WorkflowRunDeleteRequestObject) (gen.WorkflowRunDeleteResponseObject, error) {
	tenant := ctx.Get("tenant").(*dbsqlc.Tenant)
	runIds := request.Body.WorkflowRunIds
	var uuidStrings []string
	var returnErr error

	for _, runIds := range runIds {
		uuidStrings = append(uuidStrings, runIds.String())
	}

	_, err := t.config.EngineRepository.WorkflowRun().SoftDeleteSelectedWorkflowRuns(ctx.Request().Context(), tenant.ID, uuidStrings)

	if err != nil {
		returnErr = multierror.Append(err, fmt.Errorf("failed to delete workflow runs"))
		return nil, returnErr
	}

	// Create a new instance of gen.WorkflowRunCancel200JSONResponse and assign canceledWorkflowRunUUIDs to its WorkflowRunIds field
	response := gen.WorkflowRunDelete200JSONResponse{}

	// Return the response
	return response, nil
}
