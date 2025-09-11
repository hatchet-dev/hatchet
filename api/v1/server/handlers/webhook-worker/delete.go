package webhookworker

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

func (i *WebhookWorkersService) WebhookDelete(ctx echo.Context, request gen.WebhookDeleteRequestObject) (gen.WebhookDeleteResponseObject, error) {
	tenant := ctx.Get("tenant").(*dbsqlc.Tenant)
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)
	webhook := ctx.Get("webhook").(*dbsqlc.WebhookWorker)

	err := i.config.EngineRepository.WebhookWorker().SoftDeleteWebhookWorker(ctx.Request().Context(), sqlchelpers.UUIDToStr(webhook.ID), tenantId)
	if err != nil {
		return nil, err
	}

	return gen.WebhookDelete200Response{}, nil
}
