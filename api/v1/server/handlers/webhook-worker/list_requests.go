package webhookworker

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
)

func (i *WebhookWorkersService) WebhookRequestsList(ctx echo.Context, request gen.WebhookRequestsListRequestObject) (gen.WebhookRequestsListResponseObject, error) {
	return gen.WebhookRequestsList400JSONResponse(apierrors.NewAPIErrors(
		"WebhookRequestsList is deprecated",
	)), nil
}
