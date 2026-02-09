package webhookworker

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
)

func (i *WebhookWorkersService) WebhookList(ctx echo.Context, request gen.WebhookListRequestObject) (gen.WebhookListResponseObject, error) {
	return gen.WebhookList400JSONResponse(apierrors.NewAPIErrors(
		"WebhookList is deprecated",
	)), nil
}
