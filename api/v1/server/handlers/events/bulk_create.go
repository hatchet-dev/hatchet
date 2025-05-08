package events

import (
	"encoding/json"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/metered"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

func (t *EventService) EventCreateBulk(ctx echo.Context, request gen.EventCreateBulkRequestObject) (gen.EventCreateBulkResponseObject, error) {
	tenant := ctx.Get("tenant").(*dbsqlc.Tenant)
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	eventOpts := make([]*repository.CreateEventOpts, len(request.Body.Events))

	for i, event := range request.Body.Events {
		dataBytes, err := json.Marshal(event.Data)

		if err != nil {
			return nil, err
		}

		var additionalMetadata []byte

		if event.AdditionalMetadata != nil {
			additionalMetadata, err = json.Marshal(event.AdditionalMetadata)

			if err != nil {
				return nil, err
			}
		}

		eventOpts[i] = &repository.CreateEventOpts{
			TenantId:           tenantId,
			Key:                event.Key,
			Data:               dataBytes,
			AdditionalMetadata: additionalMetadata,
			Priority:           event.Priority,
		}
	}
	events, err := t.config.Ingestor.BulkIngestEvent(ctx.Request().Context(), tenant, eventOpts)

	if err != nil {

		if err == metered.ErrResourceExhausted {
			return gen.EventCreateBulk429JSONResponse(
				apierrors.NewAPIErrors("Event limit exceeded"),
			), nil
		}

		return gen.EventCreateBulk400JSONResponse{}, err

	}

	return gen.EventCreateBulk200JSONResponse{
		Events: transformers.ToEventList(events)}, nil

}
