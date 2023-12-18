package events

import (
	"encoding/json"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
	"github.com/labstack/echo/v4"
)

func (t *EventService) EventDataGet(ctx echo.Context, request gen.EventDataGetRequestObject) (gen.EventDataGetResponseObject, error) {
	event := ctx.Get("event").(*db.EventModel)

	var dataStr string

	if dataType, ok := event.Data(); ok {
		dataStr = string(json.RawMessage(dataType))
	}

	return gen.EventDataGet200JSONResponse(
		gen.EventData{
			Data: dataStr,
		},
	), nil
}
