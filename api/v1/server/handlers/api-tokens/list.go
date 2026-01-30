package apitokens

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func (a *APITokenService) ApiTokenList(ctx echo.Context, request gen.ApiTokenListRequestObject) (gen.ApiTokenListResponseObject, error) {
	tenant := ctx.Get("tenant").(*sqlcv1.Tenant)
	tenantId := tenant.ID.String()

	tokens, err := a.config.V1.APIToken().ListAPITokensByTenant(ctx.Request().Context(), tenantId)

	if err != nil {
		return nil, err
	}

	rows := make([]gen.APIToken, len(tokens))

	for i := range tokens {
		rows[i] = *transformers.ToAPIToken(tokens[i])
	}

	return gen.ApiTokenList200JSONResponse(
		gen.ListAPITokensResponse{
			Rows: &rows,
		},
	), nil
}
