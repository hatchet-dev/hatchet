package webhookworker

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
)

func (i *WebhookWorkersService) WebhookDelete(ctx echo.Context, request gen.WebhookDeleteRequestObject) (gen.WebhookDeleteResponseObject, error) {
	tenant := ctx.Get("tenant").(*db.TenantModel)
	webhook := ctx.Get("webhook").(*db.WebhookWorkerModel)

	err := i.config.EngineRepository.WebhookWorker().SoftDeleteWebhookWorker(ctx.Request().Context(), webhook.ID, tenant.ID)
	if err != nil {
		return nil, err
	}

	return gen.WebhookDelete200Response{}, nil
}
