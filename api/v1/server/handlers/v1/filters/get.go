package filtersv1

import (
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"
	"github.com/labstack/echo/v4"
)

func (t *V1FiltersService) V1FilterGet(ctx echo.Context, request gen.V1FilterGetRequestObject) (gen.V1FilterGetResponseObject, error) {
	filter := ctx.Get("v1-filter").(*sqlcv1.V1Filter)

	transformed := transformers.ToV1Filter(filter)

	return gen.V1FilterGet200JSONResponse(
		transformed,
	), nil
}
