package slackapp

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func (i *SlackAppService) SlackWebhookDelete(ctx echo.Context, req gen.SlackWebhookDeleteRequestObject) (gen.SlackWebhookDeleteResponseObject, error) {
	tenant := ctx.Get("tenant").(*sqlcv1.Tenant)
	tenantId := tenant.ID
	slack := ctx.Get("slack").(*sqlcv1.SlackAppWebhook)

	err := i.config.V1.Slack().DeleteSlackWebhook(ctx.Request().Context(), tenantId, slack.ID.String())

	if err != nil {
		return nil, err
	}

	return gen.SlackWebhookDelete204Response{}, nil
}
