package workflows

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func (t *WorkflowService) WorkflowVersionGet(ctx echo.Context, request gen.WorkflowVersionGetRequestObject) (gen.WorkflowVersionGetResponseObject, error) {
	tenant := ctx.Get("tenant").(*sqlcv1.Tenant)
	tenantId := tenant.ID
	workflow := ctx.Get("workflow").(*sqlcv1.GetWorkflowByIdRow)

	var workflowVersionId uuid.UUID

	if request.Params.Version != nil {
		workflowVersionId = *request.Params.Version
	} else {
		row, err := t.config.V1.Workflows().GetWorkflowById(
			ctx.Request().Context(),
			workflow.Workflow.ID,
		)

		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return gen.WorkflowVersionGet404JSONResponse(
					apierrors.NewAPIErrors("workflow not found"),
				), nil
			}

			return nil, err

		}

		workflowVersionId = *row.WorkflowVersionId
	}

	row, crons, events, scheduleT, stepConcurrency, workflowConcurrency, err := t.config.V1.Workflows().GetWorkflowVersionWithTriggers(ctx.Request().Context(), tenantId, workflowVersionId)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return gen.WorkflowVersionGet404JSONResponse(
				apierrors.NewAPIErrors("version not found"),
			), nil
		}

		return nil, fmt.Errorf("error fetching version: %s", err)
	}

	// The tenant filter in the lookup prevents cross-tenant reads; this keeps a caller from
	// reading a version of one workflow through another workflow's path within the tenant.
	if row.WorkflowVersion.WorkflowId != workflow.Workflow.ID {
		return gen.WorkflowVersionGet404JSONResponse(
			apierrors.NewAPIErrors("version not found"),
		), nil
	}

	resp := transformers.ToWorkflowVersion(
		&row.WorkflowVersion,
		&workflow.Workflow,
		workflowConcurrency,
		crons,
		events,
		scheduleT,
		stepConcurrency,
	)

	return gen.WorkflowVersionGet200JSONResponse(*resp), nil
}
