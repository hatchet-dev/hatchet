package webhooksv1

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func (w *V1WebhooksService) V1WebhookUpdate(ctx echo.Context, request gen.V1WebhookUpdateRequestObject) (gen.V1WebhookUpdateResponseObject, error) {
	webhook := ctx.Get("v1-webhook").(*sqlcv1.V1IncomingWebhook)

	webhook, err := w.config.V1.Webhooks().UpdateWebhook(
		ctx.Request().Context(),
		webhook.TenantID,
		webhook.Name,
		request.Body.EventKeyExpression,
	)

	if err != nil {
		return gen.V1WebhookUpdate400JSONResponse(apierrors.NewAPIErrors("failed to update webhook")), nil
	}

	transformed := transformers.ToV1Webhook(webhook)

	return gen.V1WebhookUpdate200JSONResponse(
		transformed,
	), nil
}
