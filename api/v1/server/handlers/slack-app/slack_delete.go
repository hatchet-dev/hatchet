package slackapp

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"

	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
)

func (i *SlackAppService) SlackWebhookDelete(ctx echo.Context, req gen.SlackWebhookDeleteRequestObject) (gen.SlackWebhookDeleteResponseObject, error) {
	tenant := ctx.Get("tenant").(*dbsqlc.Tenant)
	tenantId := tenant.ID.String()
	slack := ctx.Get("slack").(*dbsqlc.SlackAppWebhook)

	err := i.config.APIRepository.Slack().DeleteSlackWebhook(ctx.Request().Context(), tenantId, slack.ID.String())

	if err != nil {
		return nil, err
	}

	return gen.SlackWebhookDelete204Response{}, nil
}
