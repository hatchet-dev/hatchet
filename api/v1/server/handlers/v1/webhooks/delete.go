package webhooksv1

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func (w *V1WebhooksService) V1WebhookDelete(ctx echo.Context, request gen.V1WebhookDeleteRequestObject) (gen.V1WebhookDeleteResponseObject, error) {
	webhook := ctx.Get("v1-webhook").(*sqlcv1.V1IncomingWebhook)

	webhook, err := w.config.V1.Webhooks().DeleteWebhook(
		ctx.Request().Context(),
		webhook.TenantID,
		webhook.Name,
	)

	if err != nil {
		return gen.V1WebhookDelete400JSONResponse(apierrors.NewAPIErrors("failed to delete webhook")), nil
	}

	transformed := transformers.ToV1Webhook(webhook)

	return gen.V1WebhookDelete200JSONResponse(
		transformed,
	), nil
}
