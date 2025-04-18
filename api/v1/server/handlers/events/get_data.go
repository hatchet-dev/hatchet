package events

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/middleware/populator"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
)

func (t *EventService) EventDataGet(ctx echo.Context, request gen.EventDataGetRequestObject) (gen.EventDataGetResponseObject, error) {
	event, err := populator.FromContext(ctx).GetEvent()
	if err != nil {
		return nil, err
	}

	var dataStr string

	if len(event.Data) > 0 {
		dataStr = string(event.Data)
	}

	return gen.EventDataGet200JSONResponse(
		gen.EventData{
			Data: dataStr,
		},
	), nil
}
