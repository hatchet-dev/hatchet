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

	if scopes != nil && workflowIds != nil && len(*scopes) != len(*workflowIds) {
		return gen.V1FilterList400JSONResponse(apierrors.NewAPIErrors("scopes and workflow ids must be the same length")), nil
	}

	numScopesOrIds := 1

	if scopes != nil {
		numScopesOrIds = len(*scopes)
	} else if workflowIds != nil {
		numScopesOrIds = len(*workflowIds)
	}

	workflowIdParams := make([]pgtype.UUID, numScopesOrIds)

	if workflowIds != nil {
		for ix, id := range *workflowIds {
			workflowIdParams[ix] = sqlchelpers.UUIDFromStr(id.String())
		}
	}

	scopeParams := make([]*string, numScopesOrIds)

	if scopes != nil {
		for ix, scope := range *scopes {
			scopeParams[ix] = &scope
		}
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
