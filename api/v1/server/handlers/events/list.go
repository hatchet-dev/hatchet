package events

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
)

func (t *EventService) EventList(ctx echo.Context, request gen.EventListRequestObject) (gen.EventListResponseObject, error) {
	return gen.EventList400JSONResponse(apierrors.NewAPIErrors(
		"EventList is deprecated",
	)), nil
}
