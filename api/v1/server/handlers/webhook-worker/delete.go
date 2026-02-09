package webhookworker

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
)

func (i *WebhookWorkersService) WebhookDelete(ctx echo.Context, request gen.WebhookDeleteRequestObject) (gen.WebhookDeleteResponseObject, error) {
	return gen.WebhookDelete400JSONResponse(apierrors.NewAPIErrors(
		"WebhookDelete is deprecated",
	)), nil
}
