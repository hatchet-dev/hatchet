package events

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
)

func (t *EventService) EventCreateBulk(ctx echo.Context, request gen.EventCreateBulkRequestObject) (gen.EventCreateBulkResponseObject, error) {
	panic("no longer implemented")

	// tenant := ctx.Get("tenant").(*db.TenantModel)

	// eventOpts := make([]*repository.CreateEventOpts, len(request.Body.Events))

	// for i, event := range request.Body.Events {
	// 	dataBytes, err := json.Marshal(event.Data)

	// 	if err != nil {
	// 		return nil, err
	// 	}

	// 	var additionalMetadata []byte

	// 	if event.AdditionalMetadata != nil {
	// 		additionalMetadata, err = json.Marshal(event.AdditionalMetadata)

	// 		if err != nil {
	// 			return nil, err
	// 		}
	// 	}

	// 	eventOpts[i] = &repository.CreateEventOpts{
	// 		TenantId:           tenant.ID,
	// 		Key:                event.Key,
	// 		Data:               dataBytes,
	// 		AdditionalMetadata: additionalMetadata,
	// 	}
	// }
	// events, err := t.config.Ingestor.BulkIngestEvent(ctx.Request().Context(), tenant.ID, eventOpts)

	// if err != nil {

	// 	if err == metered.ErrResourceExhausted {
	// 		return gen.EventCreateBulk429JSONResponse(
	// 			apierrors.NewAPIErrors("Event limit exceeded"),
	// 		), nil
	// 	}

	// 	return gen.EventCreateBulk400JSONResponse{}, err

	// }

	// return gen.EventCreateBulk200JSONResponse{
	// 	Events: transformers.ToEventList(events)}, nil

}
