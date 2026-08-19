package workflows

import (
	"fmt"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func (t *WorkflowService) WorkflowUpdate(echoCtx echo.Context, request gen.WorkflowUpdateRequestObject) (gen.WorkflowUpdateResponseObject, error) {
	tenant := echoCtx.Get("tenant").(*sqlcv1.Tenant)
	workflow := echoCtx.Get("workflow").(*sqlcv1.GetWorkflowByIdRow)
	ctx := echoCtx.Request().Context()

	var result *sqlcv1.Workflow
	var err error

	if request.Body == nil {
		return gen.WorkflowUpdate400JSONResponse(apierrors.NewAPIErrors("the request body must be non-null")), nil
	}

	if request.Body.Pause != nil {
		result, err = t.applyPauseWorkflowRequest(ctx, tenant.ID, workflow.Workflow.ID, *request.Body.Pause)

		if err != nil {
			t.config.Logger.Err(err).Msg("failed to update workflow pause state")
			return nil, fmt.Errorf("failed to update workflow")
		}
	}

	return gen.WorkflowUpdate200JSONResponse(
		*transformers.ToWorkflowFromSQLC(result),
	), nil
}
