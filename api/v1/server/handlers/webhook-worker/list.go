package webhookworker

import (
	"context"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/middleware/populator"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

func (i *WebhookWorkersService) WebhookList(ctx echo.Context, request gen.WebhookListRequestObject) (gen.WebhookListResponseObject, error) {
	tenant, err := populator.FromContext(ctx).GetTenant()
	if err != nil {
		return nil, err
	}
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	webhooks, err := i.config.EngineRepository.WebhookWorker().ListActiveWebhookWorkers(context.Background(), tenantId)

	if err != nil {
		return nil, err
	}

	rows := make([]gen.WebhookWorker, len(webhooks))

	for i := range webhooks {
		rows[i] = *transformers.ToWebhookWorker(webhooks[i])
	}

	return gen.WebhookList200JSONResponse(
		gen.WebhookWorkerListResponse{
			Rows: &rows,
		},
	), nil
}
