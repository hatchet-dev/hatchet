package apitokens

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

func (a *APITokenService) ApiTokenUpdateRevoke(ctx echo.Context, request gen.ApiTokenUpdateRevokeRequestObject) (gen.ApiTokenUpdateRevokeResponseObject, error) {
	apiToken := ctx.Get("api-token").(*dbsqlc.APIToken)

	if apiToken.Internal {
		return gen.ApiTokenUpdateRevoke403JSONResponse(
			apierrors.NewAPIErrors("Cannot revoke internal API tokens"),
		), nil
	}

	err := a.config.APIRepository.APIToken().RevokeAPIToken(ctx.Request().Context(), sqlchelpers.UUIDToStr(apiToken.ID))

	if err != nil {
		return nil, err
	}

	return gen.ApiTokenUpdateRevoke204Response{}, nil
}
