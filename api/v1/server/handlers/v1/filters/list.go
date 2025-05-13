package filtersv1

import (
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v4"
)

func (t *V1FiltersService) V1FilterList(ctx echo.Context, request gen.V1FilterListRequestObject) (gen.V1FilterListResponseObject, error) {
	tenant := ctx.Get("tenant").(*dbsqlc.Tenant)

	resourceHints := request.Params.ResourceHints
	workflowIds := request.Params.WorkflowIds

	if resourceHints != nil && workflowIds != nil && len(*resourceHints) != len(*workflowIds) {
		return gen.V1FilterList400JSONResponse(apierrors.NewAPIErrors("resource hints and workflow ids must be the same length")), nil
	}

	numHintsOrIds := 1

	if resourceHints != nil {
		numHintsOrIds = len(*resourceHints)
	} else if workflowIds != nil {
		numHintsOrIds = len(*workflowIds)
	}

	tenantIds := make([]pgtype.UUID, numHintsOrIds)

	for ix := 0; ix < numHintsOrIds; ix++ {
		tenantIds[ix] = sqlchelpers.UUIDFromStr(tenant.ID.String())
	}

	workflowIdParams := make([]pgtype.UUID, numHintsOrIds)

	if workflowIds != nil {
		for _, id := range *workflowIds {
			workflowIdParams = append(workflowIdParams, sqlchelpers.UUIDFromStr(id.String()))
		}
	}

	resourceHintParams := make([]*string, numHintsOrIds)

	if resourceHints != nil {
		for _, hint := range *resourceHints {
			resourceHintParams = append(resourceHintParams, &hint)
		}
	}

	params := sqlcv1.ListFiltersParams{
		Tenantids:     tenantIds,
		Workflowids:   workflowIdParams,
		Resourcehints: resourceHintParams,
		FilterLimit:   request.Params.Limit,
		FilterOffset:  request.Params.Offset,
	}

	filters, err := t.config.V1.Filters().ListFilters(
		ctx.Request().Context(),
		params,
	)

	if err != nil {
		return gen.V1FilterList400JSONResponse(apierrors.NewAPIErrors("failed to list filters")), nil
	}

	transformed := transformers.ToV1FilterList(filters)

	return gen.V1FilterList200JSONResponse(
		transformed,
	), nil
}
