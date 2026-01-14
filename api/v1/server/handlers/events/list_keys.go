package events

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
)

func (t *EventService) EventKeyList(ctx echo.Context, request gen.EventKeyListRequestObject) (gen.EventKeyListResponseObject, error) {
	return gen.EventKeyList400JSONResponse(apierrors.NewAPIErrors(
		"EventKeyList is deprecated",
	)), nil
}
