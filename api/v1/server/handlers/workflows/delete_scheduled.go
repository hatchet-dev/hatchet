package workflows

import (
	"context"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

func (t *WorkflowService) WorkflowScheduledDelete(ctx echo.Context, request gen.WorkflowScheduledDeleteRequestObject) (gen.WorkflowScheduledDeleteResponseObject, error) {
	scheduled := ctx.Get("scheduled-workflow-run").(*dbsqlc.ListScheduledWorkflowsRow)

	// Only allow deleting scheduled runs created via API.
	if scheduled.Method != dbsqlc.WorkflowTriggerScheduledRefMethodsAPI {
		return gen.WorkflowScheduledDelete403JSONResponse(gen.APIError{
			Description: "Cannot delete scheduled run created via code definition.",
		}), nil
	}

	dbCtx, cancel := context.WithTimeout(ctx.Request().Context(), 30*time.Second)
	defer cancel()

	err := t.config.APIRepository.WorkflowRun().DeleteScheduledWorkflow(dbCtx, sqlchelpers.UUIDToStr(scheduled.TenantId), request.ScheduledWorkflowRun.String())

	if err != nil {
		return nil, err
	}

	return gen.WorkflowScheduledDelete204Response{}, nil
}
