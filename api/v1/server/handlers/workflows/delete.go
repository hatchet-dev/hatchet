package workflows

import (
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/analytics"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func (t *WorkflowService) WorkflowDelete(ctx echo.Context, request gen.WorkflowDeleteRequestObject) (gen.WorkflowDeleteResponseObject, error) {
	user, _ := ctx.Get("user").(*sqlcv1.User)
	tenant := ctx.Get("tenant").(*sqlcv1.Tenant)
	tenantId := tenant.ID
	workflow := ctx.Get("workflow").(*sqlcv1.GetWorkflowByIdRow)

	_, err := t.config.V1.Workflows().DeleteWorkflow(ctx.Request().Context(), tenantId, workflow.Workflow.ID)

	if err != nil {
		return nil, err
	}

	var userID *uuid.UUID
	if user != nil {
		userID = &user.ID
	}
	t.config.Analytics.Enqueue(
		ctx.Request().Context(),
		analytics.Workflow, analytics.Delete,
		userID,
		&tenantId,
		workflow.Workflow.ID.String(),
		nil,
	)

	return gen.WorkflowDelete204Response{}, nil
}
