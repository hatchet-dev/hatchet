package workflows

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
)

func (t *WorkflowService) WorkflowCronGet(ctx echo.Context, request gen.WorkflowCronGetRequestObject) (gen.WorkflowCronGetResponseObject, error) {
	cronValue := ctx.Get("cron-workflow")
	if cronValue == nil {
		return gen.WorkflowCronGet404JSONResponse(
			apierrors.NewAPIErrors("Cron workflow not found."),
		), nil
	}

	cron, ok := cronValue.(*dbsqlc.GetCronWorkflowByIdRow)
	if !ok || cron == nil {
		return gen.WorkflowCronGet404JSONResponse(
			apierrors.NewAPIErrors("Cron workflow not found."),
		), nil
	}

	return gen.WorkflowCronGet200JSONResponse(
		*transformers.ToCronWorkflowDetailsFromSQLC(cron),
	), nil
}
