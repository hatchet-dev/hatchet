package events

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
)

func (t *EventService) EventUpdateReplay(ctx echo.Context, request gen.EventUpdateReplayRequestObject) (gen.EventUpdateReplayResponseObject, error) {
	return gen.EventUpdateReplay400JSONResponse(apierrors.NewAPIErrors(
		"EventUpdateReplay is deprecated",
	)), nil
}
