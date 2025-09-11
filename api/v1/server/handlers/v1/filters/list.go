package filtersv1

import (
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository/v1"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v4"
)

func (t *V1FiltersService) V1FilterList(ctx echo.Context, request gen.V1FilterListRequestObject) (gen.V1FilterListResponseObject, error) {
	tenant := ctx.Get("tenant").(*dbsqlc.Tenant)

	scopes := request.Params.Scopes
	workflowIds := request.Params.WorkflowIds

	var workflowIdParams []pgtype.UUID
	var scopeParams []string

	if workflowIds != nil {
		for _, id := range *workflowIds {
			workflowIdParams = append(workflowIdParams, sqlchelpers.UUIDFromStr(id.String()))
		}
	}

	if scopes != nil {
		scopeParams = append(scopeParams, *scopes...)
	}

	filters, err := t.config.V1.Filters().ListFilters(
		ctx.Request().Context(),
		tenant.ID.String(),
		v1.ListFiltersOpts{
			WorkflowIds:  workflowIdParams,
			Scopes:       scopeParams,
			FilterLimit:  request.Params.Limit,
			FilterOffset: request.Params.Offset,
		},
	)

	if err != nil {
		return gen.V1FilterList400JSONResponse(apierrors.NewAPIErrors("failed to list filters")), nil
	}

	transformed := transformers.ToV1FilterList(filters)

	return gen.V1FilterList200JSONResponse(
		transformed,
	), nil
}
