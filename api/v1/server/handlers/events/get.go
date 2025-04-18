package events

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/middleware/populator"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
)

func (t *EventService) EventGet(ctx echo.Context, request gen.EventGetRequestObject) (gen.EventGetResponseObject, error) {
	event, err := populator.FromContext(ctx).GetEvent()
	if err != nil {
		return nil, err
	}

	return gen.EventGet200JSONResponse(
		transformers.ToEvent(event),
	), nil
}
