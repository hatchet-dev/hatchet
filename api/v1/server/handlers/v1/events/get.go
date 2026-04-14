package eventsv1

import (
	"net/http"

	"github.com/labstack/echo/v4"

	v1handlers "github.com/hatchet-dev/hatchet/api/v1/server/handlers/v1"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers/v1"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func (t *V1EventsService) V1EventGet(ctx echo.Context, request gen.V1EventGetRequestObject) (gen.V1EventGetResponseObject, error) {
	tenant := ctx.Get("tenant").(*sqlcv1.Tenant)
	event := ctx.Get("v1-event").(*v1.EventWithPayload)

	if ts := event.EventSeenAt; ts.Valid && v1handlers.IsBeforeRetention(ts.Time, tenant.DataRetentionPeriod) {
		return nil, echo.NewHTTPError(http.StatusGone, "event is outside the data retention window")
	}

	return gen.V1EventGet200JSONResponse(
		transformers.ToV1Event(event),
	), nil
}
