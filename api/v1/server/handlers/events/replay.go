package events

import (
	"github.com/hashicorp/go-multierror"
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/sqlchelpers"
)

func (t *EventService) EventUpdateReplay(ctx echo.Context, request gen.EventUpdateReplayRequestObject) (gen.EventUpdateReplayResponseObject, error) {
	tenant := ctx.Get("tenant").(*db.TenantModel)

	eventIds := make([]string, len(request.Body.EventIds))

	for i := range request.Body.EventIds {
		eventIds[i] = request.Body.EventIds[i].String()
	}

	events, err := t.config.EngineRepository.Event().ListEventsByIds(tenant.ID, eventIds)

	if err != nil {
		return nil, err
	}

	newEventIds := make([]string, len(events))

	var allErrs error

	for i := range events {
		event := events[i]

		newEvent, err := t.config.Ingestor.IngestReplayedEvent(ctx.Request().Context(), tenant.ID, event)

		if err != nil {
			allErrs = multierror.Append(allErrs, err)
		}

		newEventIds[i] = sqlchelpers.UUIDToStr(newEvent.ID)
	}

	if allErrs != nil {
		return nil, allErrs
	}

	newEvents, err := t.config.APIRepository.Event().ListEventsById(tenant.ID, newEventIds)

	if err != nil {
		return nil, err
	}

	rows := make([]gen.Event, len(newEvents))

	for i := range newEvents {
		rows[i] = *transformers.ToEvent(&newEvents[i])
	}

	return gen.EventUpdateReplay200JSONResponse(
		gen.EventList{
			Rows: &rows,
		},
	), nil
}
