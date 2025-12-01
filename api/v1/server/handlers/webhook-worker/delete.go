package webhookworker

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
)

func (i *WebhookWorkersService) WebhookDelete(ctx echo.Context, request gen.WebhookDeleteRequestObject) (gen.WebhookDeleteResponseObject, error) {
	tenant := ctx.Get("tenant").(*dbsqlc.Tenant)
	tenantId := tenant.ID.String()
	webhook := ctx.Get("webhook").(*dbsqlc.WebhookWorker)

	err := i.config.EngineRepository.WebhookWorker().SoftDeleteWebhookWorker(ctx.Request().Context(), webhook.ID.String(), tenantId)
	if err != nil {
		return nil, err
	}

	return gen.WebhookDelete200Response{}, nil
}
