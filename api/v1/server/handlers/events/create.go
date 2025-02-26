package events

import (
	"encoding/json"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/pkg/repository/metered"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

func (t *EventService) EventCreate(ctx echo.Context, request gen.EventCreateRequestObject) (gen.EventCreateResponseObject, error) {
	tenant := ctx.Get("tenant").(*dbsqlc.Tenant)
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	// marshal the data object to bytes
	dataBytes, err := json.Marshal(request.Body.Data)

	if err != nil {
		return nil, err
	}

	var additionalMetadata []byte

	if request.Body.AdditionalMetadata != nil {
		additionalMetadata, err = json.Marshal(request.Body.AdditionalMetadata)

		if err != nil {
			return nil, err
		}
	}

	newEvent, err := t.config.Ingestor.IngestEvent(ctx.Request().Context(), tenantId, request.Body.Key, dataBytes, additionalMetadata)

	if err != nil {
		if err == metered.ErrResourceExhausted {
			return gen.EventCreate429JSONResponse(
				apierrors.NewAPIErrors("Event limit exceeded"),
			), nil
		}

		return nil, err
	}

	dbNewEvent, err := t.config.APIRepository.Event().GetEventById(ctx.Request().Context(), sqlchelpers.UUIDToStr(newEvent.ID))

	if err != nil {
		return nil, err
	}

	return gen.EventCreate200JSONResponse(
		transformers.ToEvent(dbNewEvent),
	), nil
}
