package apitokens

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/constants"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

func (a *APITokenService) ApiTokenUpdateRevoke(ctx echo.Context, request gen.ApiTokenUpdateRevokeRequestObject) (gen.ApiTokenUpdateRevokeResponseObject, error) {
	apiToken := ctx.Get("api-token").(*dbsqlc.APIToken)
	user := ctx.Get("user").(*dbsqlc.User)

	if apiToken.Internal {
		return gen.ApiTokenUpdateRevoke403JSONResponse(
			apierrors.NewAPIErrors("Cannot revoke internal API tokens"),
		), nil
	}

	err := a.config.V1.APIToken().RevokeAPIToken(ctx.Request().Context(), sqlchelpers.UUIDToStr(apiToken.ID))

	if err != nil {
		return nil, err
	}

	ctx.Set(constants.ResourceIdKey.String(), apiToken.ID.String())
	ctx.Set(constants.ResourceTypeKey.String(), constants.ResourceTypeApiToken.String())

	tenantId := sqlchelpers.UUIDToStr(apiToken.TenantId)

	a.config.Analytics.Enqueue(
		"api-token:revoke",
		sqlchelpers.UUIDToStr(user.ID),
		&tenantId,
		nil,
		map[string]interface{}{
			"token_id": apiToken.ID,
		},
	)
	return gen.ApiTokenUpdateRevoke204Response{}, nil
}
