package events

import (
	"encoding/json"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
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
