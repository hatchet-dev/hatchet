package workflows

import (
	"context"
	"encoding/json"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/internal/services/ticker"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func (t *WorkflowService) WorkflowCronTrigger(ctx echo.Context, request gen.WorkflowCronTriggerRequestObject) (gen.WorkflowCronTriggerResponseObject, error) {
	tenant := ctx.Get("tenant").(*sqlcv1.Tenant)

	dbCtx, cancel := context.WithTimeout(ctx.Request().Context(), 30*time.Second)
	defer cancel()

	cron, err := t.config.V1.WorkflowSchedules().GetCronWorkflow(dbCtx, tenant.ID, request.CronWorkflow)
	if err != nil {
		return nil, err
	}

	if cron == nil {
		return gen.WorkflowCronTrigger404JSONResponse(apierrors.NewAPIErrors("Cron workflow not found.")), nil
	}

	var additionalMetadata map[string]interface{}
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
		return gen.WorkflowCronTrigger400JSONResponse(apierrors.NewAPIErrors("Failed to trigger cron workflow.")), nil
	}

	return gen.WorkflowCronTrigger200JSONResponse(gen.TriggerRunResult{
		ExternalId: *externalId,
	}), nil
}
