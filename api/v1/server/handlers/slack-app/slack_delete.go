package slackapp

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"

	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

func (i *SlackAppService) SlackWebhookDelete(ctx echo.Context, req gen.SlackWebhookDeleteRequestObject) (gen.SlackWebhookDeleteResponseObject, error) {
	tenant := ctx.Get("tenant").(*dbsqlc.Tenant)
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)
	slack := ctx.Get("slack").(*dbsqlc.SlackAppWebhook)

	err := i.config.APIRepository.Slack().DeleteSlackWebhook(ctx.Request().Context(), tenantId, sqlchelpers.UUIDToStr(slack.ID))

	if err != nil {
		return nil, err
	}

	return gen.SlackWebhookDelete204Response{}, nil
}
