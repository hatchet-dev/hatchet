package eventsv1

import (
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"
)

func (t *V1EventsService) V1EventList(ctx echo.Context, request gen.V1EventListRequestObject) (gen.V1EventListResponseObject, error) {
	tenant := ctx.Get("tenant").(*dbsqlc.Tenant)
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	limit := int64(50)
	offset := int64(0)

	if request.Params.Limit != nil {
		limit = *request.Params.Limit
	}

	if request.Params.Offset != nil {
		offset = *request.Params.Offset
	}

	opts := sqlcv1.ListEventsParams{
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
		Limit: pgtype.Int8{
			Int64: limit,
			Valid: true,
		},
		Offset: pgtype.Int8{
			Int64: offset,
			Valid: true,
		},
	}

	if request.Params.Keys != nil {
		keys := make([]string, len(*request.Params.Keys))
		copy(keys, *request.Params.Keys)
	}

	events, err := t.config.V1.OLAP().ListEvents(ctx.Request().Context(), opts)

	if err != nil {
		return nil, fmt.Errorf("failed to list events: %w", err)
	}

	rows := transformers.ToV1EventList(events)

	return gen.V1EventList200JSONResponse(
		rows,
	), nil
}
