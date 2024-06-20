package events

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
)

func (t *EventService) EventKeyList(ctx echo.Context, request gen.EventKeyListRequestObject) (gen.EventKeyListResponseObject, error) {
	tenant := ctx.Get("tenant").(*db.TenantModel)

	eventKeys, err := t.config.APIRepository.Event().ListEventKeys(tenant.ID)

	if err != nil {
		return nil, err
	}

	rows := make([]gen.EventKey, len(eventKeys))
	copy(rows, eventKeys)

	return gen.EventKeyList200JSONResponse(
		gen.EventKeyList{
			Rows: &rows,
		},
	), nil
}
