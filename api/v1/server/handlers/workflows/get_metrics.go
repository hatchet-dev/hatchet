package workflows

import (
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
)

func (t *WorkflowService) WorkflowGetMetrics(ctx echo.Context, request gen.WorkflowGetMetricsRequestObject) (gen.WorkflowGetMetricsResponseObject, error) {
	tenant := ctx.Get("tenant").(*dbsqlc.Tenant)
	tenantId := tenant.ID.String()
	workflow := ctx.Get("workflow").(*dbsqlc.GetWorkflowByIdRow)

	opts := &repository.GetWorkflowMetricsOpts{}

	if request.Params.Status != nil {
		opts.Status = (*string)(request.Params.Status)
	}

	if request.Params.GroupKey != nil {
		opts.GroupKey = request.Params.GroupKey
	}

	metrics, err := t.config.APIRepository.Workflow().GetWorkflowMetrics(tenantId, workflow.Workflow.ID.String(), opts)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return gen.WorkflowGetMetrics404JSONResponse(
				apierrors.NewAPIErrors("workflow not found"),
			), nil
		}

		return nil, err
	}

	return gen.WorkflowGetMetrics200JSONResponse(gen.WorkflowMetrics{
		GroupKeyCount:     &metrics.GroupKeyCount,
		GroupKeyRunsCount: &metrics.GroupKeyRunsCount,
	}), nil
}
