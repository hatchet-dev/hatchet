package workflows

import (
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/middleware/populator"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

func (t *WorkflowService) WorkflowGetMetrics(ctx echo.Context, request gen.WorkflowGetMetricsRequestObject) (gen.WorkflowGetMetricsResponseObject, error) {
	populator := populator.FromContext(ctx)

	tenant, err := populator.GetTenant()
	if err != nil {
		return nil, err
	}
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)
	workflow, err := populator.GetWorkflow()
	if err != nil {
		return nil, err
	}

	opts := &repository.GetWorkflowMetricsOpts{}

	if request.Params.Status != nil {
		opts.Status = (*string)(request.Params.Status)
	}

	if request.Params.GroupKey != nil {
		opts.GroupKey = request.Params.GroupKey
	}

	metrics, err := t.config.APIRepository.Workflow().GetWorkflowMetrics(tenantId, sqlchelpers.UUIDToStr(workflow.Workflow.ID), opts)

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
