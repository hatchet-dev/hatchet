package apitokens

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/middleware/populator"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

func (a *APITokenService) ApiTokenList(ctx echo.Context, request gen.ApiTokenListRequestObject) (gen.ApiTokenListResponseObject, error) {
	tenant, err := populator.FromContext(ctx).GetTenant()
	if err != nil {
		return nil, err
	}
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	tokens, err := a.config.APIRepository.APIToken().ListAPITokensByTenant(ctx.Request().Context(), tenantId)

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
