package slackapp

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/middleware/populator"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"

	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

func (i *SlackAppService) SlackWebhookDelete(ctx echo.Context, req gen.SlackWebhookDeleteRequestObject) (gen.SlackWebhookDeleteResponseObject, error) {
	populator := populator.FromContext(ctx)

	tenant, err := populator.GetTenant()
	if err != nil {
		return nil, err
	}
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	slack, err := populator.GetSlackWebhook()
	if err != nil {
		return nil, err
	}

	err = i.config.APIRepository.Slack().DeleteSlackWebhook(ctx.Request().Context(), tenantId, sqlchelpers.UUIDToStr(slack.ID))

	if err != nil {
		return nil, err
	}

	return gen.SlackWebhookDelete204Response{}, nil
}
