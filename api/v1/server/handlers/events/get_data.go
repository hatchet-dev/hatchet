package events

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func (t *EventService) EventDataGet(ctx echo.Context, request gen.EventDataGetRequestObject) (gen.EventDataGetResponseObject, error) {
	eventInterface := ctx.Get("event")
	if eventInterface == nil {
		return nil, echo.NewHTTPError(404, "event not found")
	}

	event, ok := eventInterface.(*sqlcv1.Event)
	if !ok {
		return nil, echo.NewHTTPError(500, "invalid event type in context")
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

func (t *EventService) EventDataGetWithTenant(ctx echo.Context, request gen.EventDataGetWithTenantRequestObject) (gen.EventDataGetWithTenantResponseObject, error) {
	// hack to use the tenant id to populate the event in the v1 case
	eventInterface := ctx.Get("event-with-tenant")

	if eventInterface == nil {
		return nil, echo.NewHTTPError(404, "event not found")
	}

	event, ok := eventInterface.(*sqlcv1.Event)
	if !ok {
		return nil, echo.NewHTTPError(500, "invalid event type in context")
	}

	var dataStr string

	if len(event.Data) > 0 {
		dataStr = string(event.Data)
	}

	return gen.EventDataGetWithTenant200JSONResponse(
		gen.EventData{
			Data: dataStr,
		},
	), nil
}
