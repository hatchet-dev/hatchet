package webhooksv1

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func (w *V1WebhooksService) V1WebhookGet(ctx echo.Context, request gen.V1WebhookGetRequestObject) (gen.V1WebhookGetResponseObject, error) {
	webhook := ctx.Get("v1-webhook").(*sqlcv1.V1IncomingWebhook)

	transformed := transformers.ToV1Webhook(webhook)

	return gen.V1WebhookGet200JSONResponse(
		transformed,
	), nil
}
