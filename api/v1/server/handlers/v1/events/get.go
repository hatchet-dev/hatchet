package eventsv1

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers/v1"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
)

func (t *V1EventsService) V1EventGet(ctx echo.Context, request gen.V1EventGetRequestObject) (gen.V1EventGetResponseObject, error) {
	event := ctx.Get("v1-event").(*v1.EventWithPayload)

	return gen.V1EventGet200JSONResponse(
		transformers.ToV1Event(event),
	), nil
}
