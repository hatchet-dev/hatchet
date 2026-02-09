package events

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
)

func (t *EventService) EventCreate(ctx echo.Context, request gen.EventCreateRequestObject) (gen.EventCreateResponseObject, error) {
	return gen.EventCreate400JSONResponse(apierrors.NewAPIErrors(
		"EventCreate is deprecated",
	)), nil
}
