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
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

func (t *WorkflowService) WorkflowVersionGet(ctx echo.Context, request gen.WorkflowVersionGetRequestObject) (gen.WorkflowVersionGetResponseObject, error) {
	tenant := ctx.Get("tenant").(*dbsqlc.Tenant)
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)
	workflow := ctx.Get("workflow").(*dbsqlc.GetWorkflowByIdRow)

	var workflowVersionId string

	if request.Params.Version != nil {
		workflowVersionId = request.Params.Version.String()
	} else {
		row, err := t.config.APIRepository.Workflow().GetWorkflowById(
			ctx.Request().Context(),
			sqlchelpers.UUIDToStr(workflow.Workflow.ID),
		)

		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return gen.WorkflowVersionGet404JSONResponse(
					apierrors.NewAPIErrors("workflow not found"),
				), nil
			}

			return nil, err

		}

		workflowVersionId = sqlchelpers.UUIDToStr(row.WorkflowVersionId)
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
