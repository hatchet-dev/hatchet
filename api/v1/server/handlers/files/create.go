package events

import (
	"encoding/json"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/sqlchelpers"
)

func (t *FileService) FileCreate(ctx echo.Context, request gen.FileCreateRequestObject) (gen.FileCreateResponseObject, error) {
	tenant := ctx.Get("tenant").(*db.TenantModel)

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

	// newEvent, err := t.config.Ingestor.IngestEvent(ctx.Request().Context(), tenant.ID, request.Body.Key, dataBytes, additionalMetadata)

	// dbNewEvent, err := t.config.APIRepository.Event().GetEventById(sqlchelpers.UUIDToStr(newEvent.ID))

	// if err != nil {
	// 	return nil, err
	// }

	return gen.FileCreate200JSONResponse(
		*transformers.ToFile(dbNewEvent),
	), nil
}
