package workflows

import (
	"errors"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/middleware/populator"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
)

func (t *WorkflowService) WorkflowCronGet(ctx echo.Context, request gen.WorkflowCronGetRequestObject) (gen.WorkflowCronGetResponseObject, error) {
	cronValue, err := populator.FromContext(ctx).GetCronWorkflow()

	if err != nil {
		if errors.Is(err, populator.ErrNotFound) {
			return gen.WorkflowCronGet404JSONResponse(
				apierrors.NewAPIErrors("Cron workflow not found."),
			), nil
		}

		return nil, err
	}

	return gen.WorkflowCronGet200JSONResponse(
		*transformers.ToCronWorkflowDetailsFromSQLC(cronValue),
	), nil
}
