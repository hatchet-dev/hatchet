package events

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
)

func (t *EventService) EventGet(ctx echo.Context, request gen.EventGetRequestObject) (gen.EventGetResponseObject, error) {
	event := ctx.Get("event").(*db.EventModel)

	return gen.EventGet200JSONResponse(
		*transformers.ToEvent(event),
	), nil
}
