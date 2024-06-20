package slackapp

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"

	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
)

func (i *SlackAppService) SlackWebhookDelete(ctx echo.Context, req gen.SlackWebhookDeleteRequestObject) (gen.SlackWebhookDeleteResponseObject, error) {
	tenant := ctx.Get("tenant").(*db.TenantModel)
	slack := ctx.Get("slack").(*db.SlackAppWebhookModel)

	err := i.config.APIRepository.Slack().DeleteSlackWebhook(tenant.ID, slack.ID)

	if err != nil {
		return nil, err
	}

	return gen.SlackWebhookDelete204Response{}, nil
}
