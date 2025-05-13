package filtersv1

import (
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"
	"github.com/labstack/echo/v4"
)

func (t *V1FiltersService) V1FilterGet(ctx echo.Context, request gen.V1FilterGetRequestObject) (gen.V1FilterGetResponseObject, error) {
	tenant := ctx.Get("tenant").(*dbsqlc.Tenant)

	params := sqlcv1.GetFilterParams{
		Tenantid: tenant.ID,
		ID:       sqlchelpers.UUIDFromStr(request.FilterId.String()),
	}

	filter, err := t.config.V1.Filters().GetFilter(
		ctx.Request().Context(),
		params,
	)

	if err != nil {
		return gen.V1FilterGet400JSONResponse(apierrors.NewAPIErrors("failed to get filter")), nil
	}

	transformed := transformers.ToV1Filter(filter)

	return gen.V1FilterGet200JSONResponse(
		transformed,
	), nil
}
