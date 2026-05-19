package events

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
)

func (t *EventService) EventCreateBulk(ctx echo.Context, request gen.EventCreateBulkRequestObject) (gen.EventCreateBulkResponseObject, error) {
	return gen.EventCreateBulk400JSONResponse(apierrors.NewAPIErrors(
		"EventCreateBulk is deprecated",
	)), nil
}
