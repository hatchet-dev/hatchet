package eventsv1

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func (t *V1EventsService) V1EventKeyList(ctx echo.Context, request gen.V1EventKeyListRequestObject) (gen.V1EventKeyListResponseObject, error) {
	tenant := ctx.Get("tenant").(*sqlcv1.Tenant)
	tenantId := tenant.ID.String()

	eventKeys, err := t.config.V1.OLAP().ListEventKeys(ctx.Request().Context(), tenantId)

	if err != nil {
		return nil, err
	}

	rows := make([]gen.EventKey, len(eventKeys))
	copy(rows, eventKeys)

	return gen.V1EventKeyList200JSONResponse(
		gen.EventKeyList{
			Rows: &rows,
		},
	), nil
}
