package events

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

func (t *EventService) EventKeyList(ctx echo.Context, request gen.EventKeyListRequestObject) (gen.EventKeyListResponseObject, error) {
	tenant := ctx.Get("tenant").(*dbsqlc.Tenant)
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	eventKeys, err := t.config.APIRepository.Event().ListEventKeys(tenantId)

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
