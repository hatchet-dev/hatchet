package workflows

import (
	"encoding/json"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/internal/services/ticker"
	"github.com/hatchet-dev/hatchet/pkg/analytics"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func (t *WorkflowService) WorkflowCronTrigger(ctx echo.Context, request gen.WorkflowCronTriggerRequestObject) (gen.WorkflowCronTriggerResponseObject, error) {
	tenant := ctx.Get("tenant").(*sqlcv1.Tenant)
	cron := ctx.Get("cron-workflow").(*sqlcv1.ListCronWorkflowsRow)

	var additionalMetadata map[string]any
	if len(cron.AdditionalMetadata) > 0 {
		if err := json.Unmarshal(cron.AdditionalMetadata, &additionalMetadata); err != nil {
			return nil, err
		}
	}

	priority := cron.Priority

	var cronName *string
	if cron.Name.Valid {
		cronName = &cron.Name.String
	}

	externalId, err := ticker.RunCronWorkflow(
		ctx.Request().Context(),
		t.config.MessageQueueV1,
		tenant.ID,
		cron.Cron,
		cron.WorkflowName,
		cronName,
		cron.Input,
		additionalMetadata,
		&priority,
		time.Now().UTC(),
	)
	if err != nil || externalId == nil {
		return gen.WorkflowCronTrigger500JSONResponse(apierrors.NewAPIErrors("Failed to trigger cron workflow.")), nil
	}

	t.config.Analytics.Enqueue(
		ctx.Request().Context(),
		analytics.WorkflowRun, analytics.Create,
		externalId.String(),
		nil,
	)

	return gen.WorkflowCronTrigger200JSONResponse(gen.TriggerRunResult{
		ExternalId: *externalId,
	}), nil
}
