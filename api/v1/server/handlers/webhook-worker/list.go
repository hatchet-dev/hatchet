package webhookworker

import (
	"context"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
)

func (s *WebhookWorkersService) WebhookList(ctx echo.Context, request gen.WebhookListRequestObject) (gen.WebhookListResponseObject, error) {
	tenant := ctx.Get("tenant").(*db.TenantModel)

	webhooks, err := s.config.APIRepository.WebhookWorker().ListWebhookWorkers(context.TODO(), tenant.ID)

	if err != nil {
		return nil, err
	}

	rows := make([]gen.WebhookWorker, len(webhooks))

	for i := range webhooks {
		rows[i] = *transformers.ToWebhookWorker(&webhooks[i])
	}

	return gen.WebhookList200JSONResponse(
		gen.WebhookWorkerListResponse{
			Rows: &rows,
		},
	), nil
}
