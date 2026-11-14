package workflows

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/analytics"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func (t *WorkflowService) WorkflowDelete(ctx echo.Context, request gen.WorkflowDeleteRequestObject) (gen.WorkflowDeleteResponseObject, error) {
	tenant := ctx.Get("tenant").(*sqlcv1.Tenant)
	tenantId := tenant.ID
	workflow := ctx.Get("workflow").(*sqlcv1.GetWorkflowByIdRow)

	if err := repository.CheckWorkflowOwner(&workflow.Workflow, analytics.OperatorIDFromContext(ctx.Request().Context())); err != nil {
		return gen.WorkflowDelete403JSONResponse(apierrors.NewAPIErrors(err.Error())), nil
	}

	_, err := t.config.V1.Workflows().DeleteWorkflow(ctx.Request().Context(), tenantId, workflow.Workflow.ID)

	if err != nil {
		return nil, err
	}

	t.config.Analytics.Enqueue(
		ctx.Request().Context(),
		analytics.Workflow, analytics.Delete,
		workflow.Workflow.ID.String(),
		nil,
	)

	return gen.WorkflowDelete204Response{}, nil
}
