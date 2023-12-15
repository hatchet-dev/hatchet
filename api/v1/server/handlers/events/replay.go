package events

import (
	"github.com/hashicorp/go-multierror"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
	"github.com/labstack/echo/v4"
)

func (t *EventService) EventUpdateReplay(ctx echo.Context, request gen.EventUpdateReplayRequestObject) (gen.EventUpdateReplayResponseObject, error) {
	tenant := ctx.Get("tenant").(*db.TenantModel)

	eventIds := make([]string, len(request.Body.EventIds))

	for i := range request.Body.EventIds {
		eventIds[i] = request.Body.EventIds[i].String()
	}

	events, err := t.config.Repository.Event().ListEventsById(tenant.ID, eventIds)

	if err != nil {
		return nil, err
	}

	newEvents := make([]db.EventModel, len(events))

	var allErrs error

	for i := range events {
		event := events[i]

		newEvent, err := t.config.Ingestor.IngestReplayedEvent(tenant.ID, &event)

		if err != nil {
			allErrs = multierror.Append(allErrs, err)
		}

		newEvents[i] = *newEvent
	}

	if allErrs != nil {
		return nil, allErrs
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
