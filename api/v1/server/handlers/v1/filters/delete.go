package filtersv1

import (
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"
	"github.com/labstack/echo/v4"
)

func (t *V1FiltersService) V1FilterDelete(ctx echo.Context, request gen.V1FilterDeleteRequestObject) (gen.V1FilterDeleteResponseObject, error) {
	tenant := ctx.Get("tenant").(*dbsqlc.Tenant)
	filter := ctx.Get("v1-filter").(*sqlcv1.V1Filter)

	filter, err := t.config.V1.Filters().DeleteFilter(
		ctx.Request().Context(),
		tenant.ID.String(),
		filter.ID.String(),
	)

	if err != nil {
		return gen.V1FilterDelete400JSONResponse(apierrors.NewAPIErrors("failed to delete filter")), nil
	}

	transformed := transformers.ToV1Filter(filter)

	return gen.V1FilterDelete200JSONResponse(
		transformed,
	), nil
}
