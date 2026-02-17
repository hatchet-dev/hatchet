package webhooksv1

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers/v1"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func (w *V1WebhooksService) V1WebhookList(ctx echo.Context, request gen.V1WebhookListRequestObject) (gen.V1WebhookListResponseObject, error) {
	tenant := ctx.Get("tenant").(*sqlcv1.Tenant)

	var sourceNames []sqlcv1.V1IncomingWebhookSourceName
	var webhookNames []string

	if request.Params.SourceNames != nil {
		for _, sourceName := range *request.Params.SourceNames {
			sourceNames = append(sourceNames, sqlcv1.V1IncomingWebhookSourceName(sourceName))
		}
	}

	if request.Params.WebhookNames != nil {
		webhookNames = *request.Params.WebhookNames
	}

	webhooks, err := w.config.V1.Webhooks().ListWebhooks(
		ctx.Request().Context(),
		tenant.ID,
		v1.ListWebhooksOpts{
			WebhookNames:       webhookNames,
			WebhookSourceNames: sourceNames,
			Limit:              request.Params.Limit,
			Offset:             request.Params.Offset,
		},
	)

	if err != nil {
		return gen.V1WebhookList400JSONResponse(apierrors.NewAPIErrors("failed to list webhooks")), nil
	}

	transformed := transformers.ToV1WebhookList(webhooks)

	return gen.V1WebhookList200JSONResponse(
		transformed,
	), nil
}
