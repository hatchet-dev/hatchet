package events

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
)

func (t *EventService) EventUpdateCancel(ctx echo.Context, request gen.EventUpdateCancelRequestObject) (gen.EventUpdateCancelResponseObject, error) {
	return gen.EventUpdateCancel400JSONResponse(apierrors.NewAPIErrors(
		"EventUpdateCancel is deprecated; please use V1TaskCancel instead",
	)), nil
}
