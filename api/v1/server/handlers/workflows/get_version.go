package workflows

import (
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
)

func (t *WorkflowService) WorkflowVersionGet(ctx echo.Context, request gen.WorkflowVersionGetRequestObject) (gen.WorkflowVersionGetResponseObject, error) {
	tenant := ctx.Get("tenant").(*dbsqlc.Tenant)
	tenantId := tenant.ID.String()
	workflow := ctx.Get("workflow").(*dbsqlc.GetWorkflowByIdRow)

	var workflowVersionId string

	if request.Params.Version != nil {
		workflowVersionId = request.Params.Version.String()
	} else {
		row, err := t.config.APIRepository.Workflow().GetWorkflowById(
			ctx.Request().Context(),
			workflow.Workflow.ID.String(),
		)

		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return gen.WorkflowVersionGet404JSONResponse(
					apierrors.NewAPIErrors("workflow not found"),
				), nil
			}

			return nil, err

		}

		workflowVersionId = row.WorkflowVersionId.String()
	}

	row, crons, events, scheduleT, err := t.config.APIRepository.Workflow().GetWorkflowVersionById(tenantId, workflowVersionId)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return gen.WorkflowVersionGet404JSONResponse(
				apierrors.NewAPIErrors("version not found"),
			), nil
		}

		return nil, fmt.Errorf("error fetching version: %s", err)
	}

	resp := transformers.ToWorkflowVersion(
		&row.WorkflowVersion,
		&workflow.Workflow,
		&transformers.WorkflowConcurrency{
			ID:                    row.ConcurrencyId,
			GetConcurrencyGroupId: row.ConcurrencyGroupId,
			MaxRuns:               row.ConcurrencyMaxRuns,
			LimitStrategy:         row.ConcurrencyLimitStrategy,
		},
		crons,
		events,
		scheduleT,
	)

	return gen.WorkflowVersionGet200JSONResponse(*resp), nil
}
