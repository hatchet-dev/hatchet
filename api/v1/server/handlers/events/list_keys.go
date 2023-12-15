package events

import (
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
	"github.com/labstack/echo/v4"
)

func (t *EventService) EventKeyList(ctx echo.Context, request gen.EventKeyListRequestObject) (gen.EventKeyListResponseObject, error) {
	tenant := ctx.Get("tenant").(*db.TenantModel)

	eventKeys, err := t.config.Repository.Event().ListEventKeys(tenant.ID)

	if err != nil {
		return nil, err
	}

	rows := make([]gen.EventKey, len(eventKeys))

	for i, eventKey := range eventKeys {
		rows[i] = gen.EventKey(eventKey)
	}

	return gen.EventKeyList200JSONResponse(
		gen.EventKeyList{
			Rows: &rows,
		},
	), nil
}
