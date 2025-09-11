package slackapp

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

func (s *SlackAppService) SlackWebhookList(ctx echo.Context, req gen.SlackWebhookListRequestObject) (gen.SlackWebhookListResponseObject, error) {
	tenant := ctx.Get("tenant").(*dbsqlc.Tenant)
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	webhooks, err := s.config.APIRepository.Slack().ListSlackWebhooks(ctx.Request().Context(), tenantId)

	if err != nil {
		return nil, err
	}

	rows := make([]gen.SlackWebhook, len(webhooks))

	for i := range webhooks {
		rows[i] = *transformers.ToSlackWebhook(webhooks[i])
	}

	return gen.SlackWebhookList200JSONResponse(
		gen.ListSlackWebhooks{
			Rows: rows,
		},
	), nil
}
