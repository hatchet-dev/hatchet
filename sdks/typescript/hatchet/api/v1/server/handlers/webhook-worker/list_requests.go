package webhookworker

import (
	"context"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
)

func (i *WebhookWorkersService) WebhookRequestsList(ctx echo.Context, request gen.WebhookRequestsListRequestObject) (gen.WebhookRequestsListResponseObject, error) {
	dbCtx, cancel := context.WithTimeout(ctx.Request().Context(), 30*time.Second)
	defer cancel()

	requests, err := i.config.EngineRepository.WebhookWorker().ListWebhookWorkerRequests(dbCtx, request.Webhook.String())

	if err != nil {
		return nil, err
	}

	rows := make([]gen.WebhookWorkerRequest, len(requests))

	for i := range requests {
		rows[i] = *transformers.ToWebhookWorkerRequest(requests[i])
	}

	return gen.WebhookRequestsList200JSONResponse(
		gen.WebhookWorkerRequestListResponse{
			Requests: &rows,
		},
	), nil
}
